package jobs

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/media"
	"github.com/lysyi3m/rss-comb/app/types"
)

const videoGUID = "yt:video:abc"

// fakeYTDLP is a yt-dlp stand-in: metadata calls print metadataJSON, and downloads
// leave a marker and run downloadScript, with the -o path in $out.
func fakeYTDLP(t *testing.T, metadataJSON, downloadScript string) (cmd string, called func() bool) {
	t.Helper()
	dir := t.TempDir()
	marker := filepath.Join(dir, "downloaded")
	script := filepath.Join(dir, "yt-dlp.sh")
	body := fmt.Sprintf(`#!/bin/sh
case "$*" in
  *--dump-json*) echo '%s' ;;
  *)
    touch %s
    prev=""
    for arg in "$@"; do
      [ "$prev" = "-o" ] && out=$(printf %%s "$arg" | sed 's/%%(ext)s/mp3/')
      prev=$arg
    done
    mkdir -p "$(dirname "$out")"
    %s ;;
esac
`, metadataJSON, marker, downloadScript)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return "sh " + script, func() bool {
		_, err := os.Stat(marker)
		return err == nil
	}
}

// fakeFFprobe puts an ffprobe stand-in on PATH. An empty duration makes it fail.
func fakeFFprobe(t *testing.T, duration string) {
	t.Helper()
	dir := t.TempDir()
	body := "#!/bin/sh\nexit 1\n"
	if duration != "" {
		body = "#!/bin/sh\necho " + duration + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "ffprobe"), []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// claimVideo stores one video and returns its queued download job.
func (f *feedFixture) claimVideo() *database.Job {
	f.t.Helper()
	f.process(rssItem(videoGUID, "Video"))
	job, err := f.jobRepo.ClaimJob()
	if err != nil {
		f.t.Fatal(err)
	}
	if job == nil || job.JobType != "download_media" {
		f.t.Fatalf("claimed %+v, want a download_media job", job)
	}
	return job
}

func statusOf(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func TestDownloadMediaSuccess(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()
	mediaDir := t.TempDir()
	fakeFFprobe(t, "95.5")
	cmd, _ := fakeYTDLP(t, `{"id":"abc","live_status":"not_live","duration":100,"timestamp":1700000000}`, `printf audio > "$out"`)

	handler := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", mediaDir)
	if err := handler(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	got := f.item(videoGUID)
	if statusOf(got.MediaStatus) != "ready" || got.MediaPath != "abc.mp3" || got.MediaSize != 5 || got.ITunesDuration != 95 {
		t.Errorf("media = %q/%q/%d/%d", statusOf(got.MediaStatus), got.MediaPath, got.MediaSize, got.ITunesDuration)
	}
	if got.PublishedAt.Unix() != 1700000000 {
		t.Errorf("published_at = %s, want the upload time", got.PublishedAt)
	}
}

func TestDownloadMediaDefersVideoThatIsNotReady(t *testing.T) {
	tests := []string{"is_live", "is_upcoming", "post_live"}

	for _, status := range tests {
		t.Run(status, func(t *testing.T) {
			f := newFeedFixture(t, "youtube", types.Settings{})
			job := f.claimVideo()
			cmd, downloaded := fakeYTDLP(t, `{"id":"abc","live_status":"`+status+`"}`, `exit 1`)

			err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", t.TempDir())(context.Background(), job)

			var reschedule *RescheduleError
			if !errors.As(err, &reschedule) {
				t.Fatalf("err = %v, want RescheduleError", err)
			}
			if !reschedule.RunAfter.After(time.Now()) {
				t.Errorf("run_after = %s, want the future", reschedule.RunAfter)
			}
			if downloaded() {
				t.Error("downloaded a video that is not ready")
			}
			if got := statusOf(f.item(videoGUID).MediaStatus); got != "pending" {
				t.Errorf("media_status = %q, want pending", got)
			}
		})
	}
}

func TestDownloadMediaRejectsTruncatedFile(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()
	mediaDir := t.TempDir()
	fakeFFprobe(t, "100")
	cmd, _ := fakeYTDLP(t, `{"id":"abc","live_status":"was_live","duration":1000}`, `printf audio > "$out"`)

	err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", mediaDir)(context.Background(), job)

	if err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Errorf("err = %v, want a truncation error", err)
	}
	if _, statErr := os.Stat(filepath.Join(mediaDir, "abc.mp3")); statErr == nil {
		t.Error("truncated file was kept")
	}
	if got := statusOf(f.item(videoGUID).MediaStatus); got != "pending" {
		t.Errorf("media_status = %q, want pending", got)
	}
}

func TestDownloadMediaFiltersShortFileAfterDownload(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{MinDuration: 300})
	job := f.claimVideo()
	mediaDir := t.TempDir()
	fakeFFprobe(t, "100")
	cmd, _ := fakeYTDLP(t, `{"id":"abc","live_status":"not_live"}`, `printf audio > "$out"`)

	if err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", mediaDir)(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(mediaDir, "abc.mp3")); err == nil {
		t.Error("short file was kept")
	}
	got := f.item(videoGUID)
	if !got.IsFiltered || statusOf(got.MediaStatus) != "skipped" {
		t.Errorf("filtered/status = %v/%q, want true/skipped", got.IsFiltered, statusOf(got.MediaStatus))
	}
}

func TestDownloadMediaFiltersShortVideoBeforeDownload(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{MinDuration: 300})
	job := f.claimVideo()
	cmd, downloaded := fakeYTDLP(t, `{"id":"abc","live_status":"not_live","duration":100}`, `exit 1`)

	if err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", t.TempDir())(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	if downloaded() {
		t.Error("downloaded a video below min_duration")
	}
	got := f.item(videoGUID)
	if !got.IsFiltered || statusOf(got.MediaStatus) != "skipped" {
		t.Errorf("filtered/status = %v/%q, want true/skipped", got.IsFiltered, statusOf(got.MediaStatus))
	}
}

func TestDownloadMediaFailureReturnsError(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()
	cmd, _ := fakeYTDLP(t, `{"id":"abc","live_status":"not_live"}`, `exit 1`)

	if err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", t.TempDir())(context.Background(), job); err == nil {
		t.Error("expected an error")
	}
	if got := statusOf(f.item(videoGUID).MediaStatus); got != "pending" {
		t.Errorf("media_status = %q, want pending until retries run out", got)
	}
}

// runQueuedJob runs the queued job through a real worker and handler, from retries
// seeded so that this failure is the last one.
func (f *feedFixture) runQueuedJob(jobType string, handler HandlerFunc, retries int) {
	f.t.Helper()
	if _, err := f.db.Exec(`UPDATE jobs SET retries = ?1`, retries); err != nil {
		f.t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool := NewWorkerPool(f.jobRepo, 1)
	pool.RegisterHandler(jobType, handler)
	pool.Start(ctx)
	defer func() {
		cancel()
		pool.Wait()
	}()

	f.waitForJob(func(row *jobRow) bool { return row == nil })
}

func TestDownloadMediaDoesNotPublishPartialFile(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()
	mediaDir := t.TempDir()
	cmd, _ := fakeYTDLP(t, `{"id":"abc","live_status":"not_live"}`, `printf partial > "$out"; exit 1`)

	if err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", mediaDir)(context.Background(), job); err == nil {
		t.Fatal("expected an error")
	}

	entries, err := os.ReadDir(mediaDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() == "abc.mp3" {
			t.Error("partial file published under the final name")
		}
	}
	if _, err := os.Stat(filepath.Join(mediaDir, ".incoming", "abc")); err == nil {
		t.Error("download directory was left behind")
	}
}

func TestExhaustedDownloadMarksItemFailed(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	f.process(rssItem(videoGUID, "Video"))
	cmd, _ := fakeYTDLP(t, `{"id":"abc","live_status":"not_live"}`, `exit 1`)

	f.runQueuedJob("download_media", DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", t.TempDir()), database.MediaJobMaxRetries-1)

	if got := statusOf(f.item(videoGUID).MediaStatus); got != "failed" {
		t.Errorf("media_status = %q, want failed", got)
	}
}

func TestDownloadMediaDefersWhenSameVideoIsInProgress(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()

	release := filepath.Join(t.TempDir(), "release")
	cmd, downloading := fakeYTDLP(t, `{"id":"abc","live_status":"not_live"}`,
		`while [ ! -e `+release+` ]; do sleep 0.02; done; printf audio > "$out"`)
	handler := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", t.TempDir())

	done := make(chan error, 1)
	go func() { done <- handler(context.Background(), job) }()

	for deadline := time.Now().Add(10 * time.Second); !downloading(); time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("first download did not start")
		}
	}

	var reschedule *RescheduleError
	if err := handler(context.Background(), job); !errors.As(err, &reschedule) {
		t.Errorf("err = %v, want RescheduleError", err)
	}

	if err := os.WriteFile(release, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Errorf("first download: %v", err)
	}
}

func TestDownloadMediaRedownloadsWhenReadyFileIsMissing(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()

	settings := types.Settings{RefreshInterval: 1800, MaxItems: 10, Timeout: 5}
	if err := f.feedRepo.UpsertFeedConfig("g", "https://example.com/g", "", "youtube", true, settings, []types.Filter{}, "g"); err != nil {
		t.Fatal(err)
	}
	otherID, err := f.itemRepo.UpsertItem("g", types.Item{GUID: videoGUID, ContentHash: "x", PublishedAt: time.Unix(0, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.itemRepo.UpdateMediaStatus(otherID, "ready", "abc.mp3", 7, 60); err != nil {
		t.Fatal(err)
	}

	fakeFFprobe(t, "60")
	cmd, downloaded := fakeYTDLP(t, `{"id":"abc","live_status":"not_live"}`, `printf audio > "$out"`)
	if err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", t.TempDir())(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	if !downloaded() {
		t.Error("reused a ready row whose file is missing")
	}
}

func TestDownloadMediaProceedsWhenMetadataFails(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()
	fakeFFprobe(t, "") // a failed probe does not block the item either
	cmd, downloaded := fakeYTDLP(t, `not json`, `printf audio > "$out"`)

	if err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", t.TempDir())(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	if !downloaded() {
		t.Error("metadata failure stopped the download")
	}
	if got := statusOf(f.item(videoGUID).MediaStatus); got != "ready" {
		t.Errorf("media_status = %q, want ready", got)
	}
}

func TestDownloadMediaReusesFileOnDisk(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()
	mediaDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(mediaDir, "abc.mp3"), []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	fakeFFprobe(t, "60")
	cmd, downloaded := fakeYTDLP(t, `{"id":"abc"}`, `exit 1`)

	if err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", mediaDir)(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	if downloaded() {
		t.Error("downloaded a file already on disk")
	}
	got := f.item(videoGUID)
	if statusOf(got.MediaStatus) != "ready" || got.MediaSize != 5 || got.ITunesDuration != 60 {
		t.Errorf("media = %q/%d/%d, want ready/5/60", statusOf(got.MediaStatus), got.MediaSize, got.ITunesDuration)
	}
}

func TestDownloadMediaReusesReadyRowFromAnotherFeed(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()

	settings := types.Settings{RefreshInterval: 1800, MaxItems: 10, Timeout: 5}
	if err := f.feedRepo.UpsertFeedConfig("g", "https://example.com/g", "", "youtube", true, settings, []types.Filter{}, "g"); err != nil {
		t.Fatal(err)
	}
	otherID, err := f.itemRepo.UpsertItem("g", types.Item{GUID: videoGUID, ContentHash: "x", PublishedAt: time.Unix(0, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.itemRepo.UpdateMediaStatus(otherID, "ready", "abc.mp3", 7, 60); err != nil {
		t.Fatal(err)
	}
	mediaDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(mediaDir, "abc.mp3"), []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd, downloaded := fakeYTDLP(t, `{"id":"abc"}`, `exit 1`)

	if err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", mediaDir)(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	if downloaded() {
		t.Error("downloaded a video another feed already has")
	}
	got := f.item(videoGUID)
	if statusOf(got.MediaStatus) != "ready" || got.MediaSize != 7 || got.ITunesDuration != 60 {
		t.Errorf("media = %q/%d/%d", statusOf(got.MediaStatus), got.MediaSize, got.ITunesDuration)
	}
}

func (f *feedFixture) disableFeed(feedType string, settings types.Settings) {
	f.t.Helper()
	settings.RefreshInterval, settings.MaxItems, settings.Timeout = 1800, 10, 5
	if err := f.feedRepo.UpsertFeedConfig("f", f.srv.URL, "", feedType, false, settings, []types.Filter{}, "disabled"); err != nil {
		f.t.Fatal(err)
	}
}

func TestDownloadMediaDefersWhileFeedIsDisabled(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()
	f.disableFeed("youtube", types.Settings{})
	cmd, downloaded := fakeYTDLP(t, `{"id":"abc","live_status":"not_live"}`, `exit 1`)

	err := DownloadMediaHandler(f.feedRepo, f.itemRepo, cmd, "", t.TempDir())(context.Background(), job)

	var reschedule *RescheduleError
	if !errors.As(err, &reschedule) {
		t.Fatalf("err = %v, want RescheduleError", err)
	}
	if downloaded() {
		t.Error("downloaded for a disabled feed")
	}
	if got := statusOf(f.item(videoGUID).MediaStatus); got != "pending" {
		t.Errorf("media_status = %q, want pending", got)
	}
}

func TestExtractContentDefersWhileFeedIsDisabled(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	defer srv.Close()

	f := newFeedFixture(t, "", types.Settings{ExtractContent: true})
	job := f.claimArticle(srv.URL)
	f.disableFeed("", types.Settings{ExtractContent: true})

	err := ExtractContentHandler(f.feedRepo, f.itemRepo, srv.Client(), "test")(context.Background(), job)

	var reschedule *RescheduleError
	if !errors.As(err, &reschedule) {
		t.Fatalf("err = %v, want RescheduleError", err)
	}
	if requests != 0 {
		t.Errorf("fetched the article %d times for a disabled feed", requests)
	}
}

func TestVideoRescheduleDelaysOverdueStream(t *testing.T) {
	got := videoReschedule(media.VideoInfo{LiveStatus: "is_upcoming", ReleaseTimestamp: time.Now().Add(-time.Hour).Unix()})
	if got == nil || !got.RunAfter.After(time.Now()) {
		t.Errorf("reschedule = %+v, want a future time", got)
	}
}

func TestVideoRescheduleLetsOtherStatusesDownload(t *testing.T) {
	for _, status := range []string{"", "not_live", "was_live"} {
		if got := videoReschedule(media.VideoInfo{LiveStatus: status}); got != nil {
			t.Errorf("status %q deferred: %v", status, got)
		}
	}
}

func TestVideoRescheduleUsesReleaseTime(t *testing.T) {
	release := time.Now().Add(6 * time.Hour).Truncate(time.Second)
	got := videoReschedule(media.VideoInfo{LiveStatus: "is_upcoming", ReleaseTimestamp: release.Unix()})
	if got == nil || !got.RunAfter.Equal(release) {
		t.Errorf("reschedule = %+v, want %s", got, release)
	}
}

// claimArticle stores one article that links to articleURL and returns its job.
func (f *feedFixture) claimArticle(articleURL string) *database.Job {
	f.t.Helper()
	f.process(fmt.Sprintf(`<item><guid>a</guid><title>T</title><link>%s</link><description>Summary</description><pubDate>Mon, 03 Jul 2023 10:00:00 GMT</pubDate></item>`, articleURL))
	job, err := f.jobRepo.ClaimJob()
	if err != nil {
		f.t.Fatal(err)
	}
	if job == nil || job.JobType != "extract_content" {
		f.t.Fatalf("claimed %+v, want an extract_content job", job)
	}
	return job
}

func TestExtractContentSuccess(t *testing.T) {
	text := strings.Repeat("Enough article text for extraction. ", 40)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body><article><p>%s</p><p>%s</p></article></body></html>`, text, text)
	}))
	defer srv.Close()

	f := newFeedFixture(t, "", types.Settings{ExtractContent: true})
	job := f.claimArticle(srv.URL)

	if err := ExtractContentHandler(f.feedRepo, f.itemRepo, srv.Client(), "test")(context.Background(), job); err != nil {
		t.Fatal(err)
	}

	got := f.item("a")
	if statusOf(got.ContentExtractionStatus) != "ready" || !strings.Contains(got.Content, "Enough article text") {
		t.Errorf("status/content = %q/%q", statusOf(got.ContentExtractionStatus), got.Content)
	}
}

func TestExtractContentFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	f := newFeedFixture(t, "", types.Settings{ExtractContent: true})
	job := f.claimArticle(srv.URL)
	if err := f.jobRepo.DelayJob(job.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	handler := ExtractContentHandler(f.feedRepo, f.itemRepo, srv.Client(), "test")

	if err := handler(context.Background(), job); err == nil {
		t.Error("expected an error")
	}
	if got := statusOf(f.item("a").ContentExtractionStatus); got != "pending" {
		t.Errorf("status = %q, want pending until retries run out", got)
	}

	f.runQueuedJob("extract_content", handler, database.ExtractionJobMaxRetries-1)

	got := f.item("a")
	if statusOf(got.ContentExtractionStatus) != "failed" {
		t.Errorf("status = %q, want failed", statusOf(got.ContentExtractionStatus))
	}
}

func TestFetchFeedCleansOrphanedMediaOnly(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()
	if err := f.itemRepo.UpdateMediaStatus(*job.ItemID, "ready", "abc.mp3", 5, 60); err != nil {
		t.Fatal(err)
	}

	mediaDir := t.TempDir()
	for _, name := range []string{"abc.mp3", "orphan.mp3", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(mediaDir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	fetchJob := &database.Job{FeedID: f.feedID()}
	handler := FetchFeedHandler(f.feedRepo, f.itemRepo, f.jobRepo, f.srv.Client(), "test", mediaDir)
	if err := handler(context.Background(), fetchJob); err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]bool{"abc.mp3": true, "orphan.mp3": false, "notes.txt": true} {
		_, err := os.Stat(filepath.Join(mediaDir, name))
		if got := err == nil; got != want {
			t.Errorf("%s exists = %v, want %v", name, got, want)
		}
	}
}

func TestFetchFeedEvictsMediaAndRestoresItWhenEligibleAgain(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	job := f.claimVideo()
	if err := f.itemRepo.UpdateMediaStatus(*job.ItemID, "ready", "abc.mp3", 5, 60); err != nil {
		t.Fatal(err)
	}
	if err := f.jobRepo.CompleteJob(job.ID); err != nil {
		t.Fatal(err)
	}

	mediaDir := t.TempDir()
	path := filepath.Join(mediaDir, "abc.mp3")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler := FetchFeedHandler(f.feedRepo, f.itemRepo, f.jobRepo, f.srv.Client(), "test", mediaDir)
	fetchJob := &database.Job{FeedID: f.feedID()}

	filters := []types.Filter{{Field: "title", Excludes: []string{"video"}}}
	settings := types.Settings{RefreshInterval: 1800, MaxItems: 10, Timeout: 5}
	if err := f.feedRepo.UpsertFeedConfig("f", f.srv.URL, "", "youtube", true, settings, filters, "h2"); err != nil {
		t.Fatal(err)
	}
	if err := f.itemRepo.UpdateItemFilterStatus(*job.ItemID, true); err != nil {
		t.Fatal(err)
	}
	if err := handler(context.Background(), fetchJob); err != nil {
		t.Fatal(err)
	}

	got := f.item(videoGUID)
	if got.MediaStatus != nil || got.MediaPath != "" {
		t.Errorf("media = %q/%q, want the row cleared", statusOf(got.MediaStatus), got.MediaPath)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("evicted file was kept")
	}

	if err := f.itemRepo.UpdateItemFilterStatus(*job.ItemID, false); err != nil {
		t.Fatal(err)
	}
	if err := handler(context.Background(), fetchJob); err != nil {
		t.Fatal(err)
	}

	if got := statusOf(f.item(videoGUID).MediaStatus); got != "pending" {
		t.Errorf("media_status = %q, want pending", got)
	}
	if n := f.drainJobs(); n != 1 {
		t.Errorf("created %d jobs, want 1", n)
	}
}

func TestCleanupKeepsFilesOfPendingItem(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	f.process(rssItem(videoGUID, "Video"))

	mediaDir := t.TempDir()
	path := filepath.Join(mediaDir, "abc.mp3")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(mediaDir, ".incoming", "abc")
	abandoned := filepath.Join(mediaDir, ".incoming", "gone")
	for _, dir := range []string{active, abandoned} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := cleanupMedia(f.itemRepo, mediaDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("cleanup deleted the file of a pending item")
	}
	if _, err := os.Stat(active); err != nil {
		t.Error("cleanup deleted the download directory of a pending item")
	}
	if _, err := os.Stat(abandoned); err == nil {
		t.Error("cleanup kept an abandoned download directory")
	}
}

func TestCleanupSkipsTurnWhileDownloadRuns(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	mediaDir := t.TempDir()
	orphan := filepath.Join(mediaDir, "orphan.mp3")
	if err := os.WriteFile(orphan, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	mediaFiles.RLock()
	err := cleanupMedia(f.itemRepo, mediaDir)
	mediaFiles.RUnlock()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); err != nil {
		t.Error("cleanup ran during a download")
	}

	if err := cleanupMedia(f.itemRepo, mediaDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(orphan); err == nil {
		t.Error("cleanup did not catch up")
	}
}
