package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/media"
)

// RescheduleError signals that a job should be rescheduled to a later time
// without incrementing retry count. Used for live/upcoming videos that aren't
// ready for download yet.
type RescheduleError struct {
	RunAfter time.Time
	Reason   string
}

func (e *RescheduleError) Error() string {
	return fmt.Sprintf("rescheduled to %s: %s", e.RunAfter.Format(time.RFC3339), e.Reason)
}

// FetchFeedHandler returns a HandlerFunc that processes a feed by resolving
// the feed name from the job's FeedID. After processing youtube feeds, it
// runs global media cleanup.
func FetchFeedHandler(
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	jobRepo *database.JobRepository,
	httpClient *http.Client,
	userAgent string,
	mediaDir string,
) HandlerFunc {
	return func(ctx context.Context, job *database.Job) error {
		dbFeed, err := feedRepo.GetFeedByID(job.FeedID)
		if err != nil {
			return fmt.Errorf("failed to get feed by ID: %w", err)
		}
		if dbFeed == nil {
			return fmt.Errorf("feed not found for ID: %s", job.FeedID)
		}

		if err := processFeed(ctx, dbFeed.Name, feedRepo, itemRepo, jobRepo, httpClient, userAgent); err != nil {
			return fmt.Errorf("[%s] %w", dbFeed.Name, err)
		}

		if dbFeed.FeedType == "youtube" {
			if err := cleanupMedia(itemRepo, mediaDir); err != nil {
				slog.Error("Media cleanup failed", "error", err)
			}
		}

		return nil
	}
}

// mediaFiles keeps cleanup away from running downloads. A download holds it shared from
// its first file check until its row is ready; cleanup needs it exclusively, because
// its keep list says nothing about jobs created after the list was read.
var mediaFiles sync.RWMutex

// cleanupMedia keeps the files of ready items in the window and of pending items. It
// skips its turn while a download runs: it runs after every YouTube fetch, so a later
// one catches up.
func cleanupMedia(itemRepo *database.ItemRepository, mediaDir string) error {
	if !mediaFiles.TryLock() {
		return nil
	}
	defer mediaFiles.Unlock()

	pending, err := itemRepo.GetPendingMediaGUIDs()
	if err != nil {
		return err
	}
	keepPaths, err := itemRepo.ReconcileMedia()
	if err != nil {
		return err
	}
	for _, guid := range pending {
		keepPaths = append(keepPaths, media.MediaFileID(guid)+".mp3")
	}

	deleted, err := media.CleanupMedia(mediaDir, keepPaths)
	if err != nil {
		return err
	}
	if deleted > 0 {
		slog.Info("Media cleanup completed", "deleted", deleted)
	}
	return nil
}

// ExtractContentHandler returns a HandlerFunc that fetches HTML content
// from an item's link and extracts clean text using go-readability.
func ExtractContentHandler(
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	httpClient *http.Client,
	userAgent string,
) HandlerFunc {
	return func(ctx context.Context, job *database.Job) error {
		if job.ItemID == nil {
			return fmt.Errorf("extract_content job has no item_id")
		}

		item, err := itemRepo.GetItemByID(*job.ItemID)
		if err != nil {
			return fmt.Errorf("failed to get item: %w", err)
		}
		if item == nil {
			return fmt.Errorf("item not found for ID: %s", *job.ItemID)
		}

		dbFeed, err := feedRepo.GetFeedByID(job.FeedID)
		if err != nil {
			return fmt.Errorf("failed to get feed: %w", err)
		}
		if dbFeed == nil {
			return fmt.Errorf("feed not found for ID: %s", job.FeedID)
		}

		settings, err := dbFeed.GetSettings()
		if err != nil {
			return fmt.Errorf("failed to get feed settings: %w", err)
		}

		if !dbFeed.IsEnabled {
			return disabledFeedDeferral()
		}

		if item.Link == "" {
			return fmt.Errorf("content extraction failed: item has no link")
		}

		data, pageURL, err := fetchURL(ctx, item.Link, settings.Timeout, httpClient, userAgent, true)
		if err != nil {
			return fmt.Errorf("content extraction failed: %w", err)
		}

		extractedContent, err := feed.Extract(data, pageURL)
		if err != nil {
			return fmt.Errorf("content extraction failed: %w", err)
		}

		if err := itemRepo.UpdateContentExtractionStatus(*job.ItemID, "ready", extractedContent); err != nil {
			return fmt.Errorf("failed to update extraction status: %w", err)
		}

		return nil
	}
}

// DownloadMediaHandler returns a HandlerFunc that downloads audio from
// a video URL using yt-dlp. Uses three-layer dedup: DB → filesystem → download.
func DownloadMediaHandler(
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	ytdlpCmd string,
	ytdlpArgs string,
	mediaDir string,
) HandlerFunc {
	// File IDs being worked on. Two feeds can hold the same video, and both jobs
	// would write the same output file.
	var inProgress sync.Map

	return func(ctx context.Context, job *database.Job) error {
		if job.ItemID == nil {
			return fmt.Errorf("download_media job has no item_id")
		}

		item, err := itemRepo.GetItemByID(*job.ItemID)
		if err != nil {
			return fmt.Errorf("failed to get item: %w", err)
		}
		if item == nil {
			return fmt.Errorf("item not found for ID: %s", *job.ItemID)
		}

		fileID := media.MediaFileID(item.GUID)
		mediaPath := fileID + ".mp3"

		if _, busy := inProgress.LoadOrStore(fileID, struct{}{}); busy {
			return &RescheduleError{
				RunAfter: time.Now().Add(1 * time.Minute),
				Reason:   "another job is working on the same video",
			}
		}
		defer inProgress.Delete(fileID)

		mediaFiles.RLock()
		defer mediaFiles.RUnlock()

		// Load min_duration setting for filtering short videos
		dbFeed, err := feedRepo.GetFeedByID(job.FeedID)
		if err != nil {
			return fmt.Errorf("failed to get feed: %w", err)
		}
		if dbFeed == nil {
			return fmt.Errorf("feed not found for ID: %s", job.FeedID)
		}
		settings, err := dbFeed.GetSettings()
		if err != nil {
			return fmt.Errorf("failed to get feed settings: %w", err)
		}
		minDuration := settings.MinDuration

		if !dbFeed.IsEnabled {
			return disabledFeedDeferral()
		}

		// Layer 1: DB check — does any item already have this file ready?
		existing, err := itemRepo.GetReadyMediaByPath(mediaPath)
		if err != nil {
			return fmt.Errorf("failed to look up ready media: %w", err)
		}
		if _, onDisk := media.FileExists(mediaDir, mediaPath); existing != nil && onDisk {
			existingDuration := existing.ITunesDuration
			// If duration wasn't recorded (e.g. downloaded before duration tracking),
			// probe the file so min_duration can be enforced.
			if minDuration > 0 && existingDuration <= 0 {
				if probedDuration, err := media.GetAudioDuration(mediaDir, existing.MediaPath); err == nil && probedDuration > 0 {
					existingDuration = probedDuration
				}
			}
			if minDuration > 0 && existingDuration > 0 && existingDuration < minDuration {
				return filterShortVideo(itemRepo, *job.ItemID, existingDuration, minDuration)
			}
			if err := itemRepo.UpdateMediaStatus(*job.ItemID, "ready", existing.MediaPath, existing.MediaSize, existingDuration); err != nil {
				return fmt.Errorf("failed to update media status (reuse): %w", err)
			}
			return nil
		}

		// Layer 2: Filesystem check — file exists but DB doesn't know
		if size, ok := media.FileExists(mediaDir, mediaPath); ok {
			duration, err := media.GetAudioDuration(mediaDir, mediaPath)
			if err != nil && minDuration > 0 {
				slog.Warn("Could not probe duration for min_duration check, proceeding without filtering", "item_id", *job.ItemID, "error", err)
			}
			if minDuration > 0 && duration > 0 && duration < minDuration {
				return filterShortVideo(itemRepo, *job.ItemID, duration, minDuration)
			}
			if err := itemRepo.UpdateMediaStatus(*job.ItemID, "ready", mediaPath, size, duration); err != nil {
				return fmt.Errorf("failed to update media status (filesystem): %w", err)
			}
			return nil
		}

		// Layer 3: Check video info before downloading
		if item.Link == "" {
			return fmt.Errorf("media download failed: item has no link")
		}

		var duration int
		videoInfo, err := media.GetVideoInfo(ctx, ytdlpCmd, ytdlpArgs, item.Link)
		if err != nil {
			// Metadata failure is not fatal — proceed and rely on post-download
			// ffprobe for the authoritative duration check.
			slog.Warn("Video info check failed, proceeding with download", "item_id", *job.ItemID, "error", err)
		} else if reschedule := videoReschedule(videoInfo); reschedule != nil {
			slog.Info("Video not ready for download",
				"item_id", *job.ItemID, "live_status", videoInfo.LiveStatus, "reschedule_at", reschedule.RunAfter)
			return reschedule
		} else {
			duration = videoInfo.Duration
		}

		if minDuration > 0 && duration > 0 && duration < minDuration {
			return filterShortVideo(itemRepo, *job.ItemID, duration, minDuration)
		}

		// Layer 4: Actually download
		path, size, err := media.Download(ctx, ytdlpCmd, ytdlpArgs, mediaDir, item.Link, fileID)
		if err != nil {
			return fmt.Errorf("media download failed: %w", err)
		}

		// Use ffprobe for authoritative duration (yt-dlp metadata can be null for live VODs)
		if probeDuration, probeErr := media.GetAudioDuration(mediaDir, path); probeErr == nil && probeDuration > 0 {
			// Detect truncated VOD downloads: if metadata reports a much longer duration
			// than the actual file, YouTube hasn't finished processing the full recording.
			if duration > 0 && probeDuration < duration*80/100 {
				os.Remove(filepath.Join(mediaDir, path))
				return fmt.Errorf("media download failed: downloaded file is truncated: got %ds, expected ~%ds (VOD likely still processing)", probeDuration, duration)
			}
			duration = probeDuration
		}

		// Post-download min_duration check using authoritative ffprobe duration.
		// Catches cases where pre-download metadata was unavailable or inaccurate.
		if minDuration > 0 && duration > 0 && duration < minDuration {
			os.Remove(filepath.Join(mediaDir, path))
			return filterShortVideo(itemRepo, *job.ItemID, duration, minDuration)
		}

		if err := itemRepo.UpdateMediaStatus(*job.ItemID, "ready", path, size, duration); err != nil {
			return fmt.Errorf("failed to update media status: %w", err)
		}

		// Update published_at from yt-dlp metadata — the Atom feed's <published>
		// date for live streams reflects when the stream was scheduled, not when
		// the VOD became available.
		if videoInfo.UploadTimestamp > 0 {
			publishedAt := time.Unix(videoInfo.UploadTimestamp, 0)
			if err := itemRepo.UpdateItemPublishedAt(*job.ItemID, publishedAt); err != nil {
				slog.Warn("Failed to update published_at from yt-dlp metadata", "item_id", *job.ItemID, "error", err)
			}
		}

		slog.Info("Media downloaded successfully", "item_id", *job.ItemID, "media_path", path, "size", size, "duration", duration)
		return nil
	}
}

// disabledFeedDeferral parks a job of a disabled feed without doing its work. The job
// stays queued rather than being deleted, so its hidden item finishes if the feed is
// enabled again.
func disabledFeedDeferral() *RescheduleError {
	return &RescheduleError{
		RunAfter: time.Now().Add(1 * time.Hour),
		Reason:   "feed is disabled",
	}
}

func filterShortVideo(itemRepo *database.ItemRepository, itemID string, duration, minDuration int) error {
	slog.Info("Video below min_duration, filtering",
		"item_id", itemID, "duration", duration, "min_duration", minDuration)
	if err := itemRepo.UpdateItemFilterStatus(itemID, true); err != nil {
		return fmt.Errorf("failed to filter short video: %w", err)
	}
	// Also mark media as skipped so the item stays hidden even if Refilter()
	// clears is_filtered (refilter only knows about pattern-based filters).
	if err := itemRepo.UpdateMediaStatus(itemID, "skipped", "", 0, duration); err != nil {
		return fmt.Errorf("failed to update media status for short video: %w", err)
	}
	return nil
}

// videoReschedule returns a RescheduleError only for unambiguous pre-download
// signals: upcoming, currently live, or post-live processing. All other statuses
// (including empty, "was_live", "not_live") proceed to download — the format
// filter in Download() prevents premature HLS-only downloads of unprocessed VODs.
func videoReschedule(info media.VideoInfo) *RescheduleError {
	switch info.LiveStatus {
	case "is_upcoming":
		// An overdue stream can still report is_upcoming. Its past release time would
		// recheck it immediately, so it falls through to the delay below.
		if release := time.Unix(info.ReleaseTimestamp, 0); info.ReleaseTimestamp > 0 && release.After(time.Now()) {
			return &RescheduleError{
				RunAfter: release,
				Reason:   "video is upcoming, scheduled for " + release.Format(time.RFC3339),
			}
		}
		return &RescheduleError{
			RunAfter: time.Now().Add(1 * time.Hour),
			Reason:   "video is upcoming, no scheduled time available",
		}
	case "is_live":
		return &RescheduleError{
			RunAfter: time.Now().Add(15 * time.Minute),
			Reason:   "video is currently live",
		}
	case "post_live":
		return &RescheduleError{
			RunAfter: time.Now().Add(15 * time.Minute),
			Reason:   "video VOD is being processed",
		}
	default:
		return nil
	}
}
