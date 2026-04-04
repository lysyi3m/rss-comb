package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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
			keepPaths, err := itemRepo.GetAllActiveMediaPaths()
			if err != nil {
				slog.Error("Failed to get active media paths for cleanup", "error", err)
				return nil
			}
			deleted, err := media.CleanupMedia(mediaDir, keepPaths)
			if err != nil {
				slog.Error("Media cleanup failed", "error", err)
			} else if deleted > 0 {
				slog.Info("Media cleanup completed", "deleted", deleted)
			}
		}

		return nil
	}
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

		if item.Link == "" {
			return handleExtractionFailure(itemRepo, *job.ItemID, job, fmt.Errorf("item has no link"))
		}

		data, err := fetchURL(ctx, item.Link, settings.Timeout, httpClient, userAgent, true)
		if err != nil {
			return handleExtractionFailure(itemRepo, *job.ItemID, job, err)
		}

		extractedContent, err := feed.Extract(data)
		if err != nil {
			return handleExtractionFailure(itemRepo, *job.ItemID, job, err)
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

		// Layer 1: DB check — does any item already have this file ready?
		if existing, _ := itemRepo.GetReadyMediaByPath(mediaPath); existing != nil {
			if err := itemRepo.UpdateMediaStatus(*job.ItemID, "ready", existing.MediaPath, existing.MediaSize, existing.ITunesDuration); err != nil {
				return fmt.Errorf("failed to update media status (reuse): %w", err)
			}
			return nil
		}

		// Layer 2: Filesystem check — file exists but DB doesn't know
		if size, ok := media.FileExists(mediaDir, mediaPath); ok {
			duration, _ := media.GetAudioDuration(mediaDir, mediaPath)
			if err := itemRepo.UpdateMediaStatus(*job.ItemID, "ready", mediaPath, size, duration); err != nil {
				return fmt.Errorf("failed to update media status (filesystem): %w", err)
			}
			return nil
		}

		// Layer 3: Check video info before downloading
		if item.Link == "" {
			return handleMediaFailure(itemRepo, *job.ItemID, job, fmt.Errorf("item has no link"))
		}

		var duration int
		videoInfo, err := media.GetVideoInfo(ctx, ytdlpCmd, item.Link)
		if err != nil {
			slog.Warn("Video info check failed, proceeding with download", "item_id", *job.ItemID, "error", err)
		} else if reschedule := videoReschedule(videoInfo); reschedule != nil {
			slog.Info("Video not ready for download",
				"item_id", *job.ItemID, "live_status", videoInfo.LiveStatus, "reschedule_at", reschedule.RunAfter)
			return reschedule
		} else {
			duration = videoInfo.Duration
		}

		// Layer 4: Actually download
		path, size, err := media.Download(ctx, ytdlpCmd, ytdlpArgs, mediaDir, item.Link, fileID)
		if err != nil {
			return handleMediaFailure(itemRepo, *job.ItemID, job, err)
		}

		// Use ffprobe for authoritative duration (yt-dlp metadata can be null for live VODs)
		if probeDuration, probeErr := media.GetAudioDuration(mediaDir, path); probeErr == nil && probeDuration > 0 {
			// Detect truncated VOD downloads: if metadata reports a much longer duration
			// than the actual file, YouTube hasn't finished processing the full recording.
			if duration > 0 && probeDuration < duration*80/100 {
				os.Remove(filepath.Join(mediaDir, path))
				return handleMediaFailure(itemRepo, *job.ItemID, job,
					fmt.Errorf("downloaded file is truncated: got %ds, expected ~%ds (VOD likely still processing)", probeDuration, duration))
			}
			duration = probeDuration
		}

		if err := itemRepo.UpdateMediaStatus(*job.ItemID, "ready", path, size, duration); err != nil {
			return fmt.Errorf("failed to update media status: %w", err)
		}

		slog.Info("Media downloaded successfully", "item_id", *job.ItemID, "media_path", path, "size", size, "duration", duration)
		return nil
	}
}

// handleExtractionFailure checks if this is the last retry attempt.
// On final failure, marks the item as 'failed' and returns nil (job completes).
// Otherwise returns the error so the job will be retried.
func handleExtractionFailure(itemRepo *database.ItemRepository, itemID string, job *database.Job, extractionErr error) error {
	if job.Retries >= job.MaxRetries-1 {
		slog.Warn("Content extraction permanently failed, item will use original content",
			"item_id", itemID, "error", extractionErr, "retries", job.Retries+1)
		if err := itemRepo.UpdateContentExtractionStatus(itemID, "failed", ""); err != nil {
			slog.Error("Failed to mark item extraction as failed", "item_id", itemID, "error", err)
		}
		return nil
	}
	return fmt.Errorf("content extraction failed: %w", extractionErr)
}

// handleMediaFailure checks if this is the last retry attempt.
// On final failure, marks the item as 'failed' and returns nil (item stays hidden forever).
// Otherwise returns the error so the job will be retried.
func handleMediaFailure(itemRepo *database.ItemRepository, itemID string, job *database.Job, mediaErr error) error {
	if job.Retries >= job.MaxRetries-1 {
		slog.Warn("Media download permanently failed, item will stay hidden",
			"item_id", itemID, "error", mediaErr, "retries", job.Retries+1)
		if err := itemRepo.UpdateMediaStatus(itemID, "failed", "", 0, 0); err != nil {
			slog.Error("Failed to mark item media as failed", "item_id", itemID, "error", err)
		}
		return nil
	}
	return fmt.Errorf("media download failed: %w", mediaErr)
}

// videoReschedule returns a RescheduleError only for unambiguous pre-download
// signals: upcoming, currently live, or post-live processing. All other statuses
// (including empty, "was_live", "not_live") proceed to download — the format
// filter in Download() prevents premature HLS-only downloads of unprocessed VODs.
func videoReschedule(info media.VideoInfo) *RescheduleError {
	switch info.LiveStatus {
	case "is_upcoming":
		if info.ReleaseTimestamp > 0 {
			return &RescheduleError{
				RunAfter: time.Unix(info.ReleaseTimestamp, 0),
				Reason:   "video is upcoming, scheduled for " + time.Unix(info.ReleaseTimestamp, 0).Format(time.RFC3339),
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
