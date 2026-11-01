package jobs

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/types"
)

func rssItem(guid, title string) string {
	return fmt.Sprintf(`<item><guid>%s</guid><title>%s</title><link>https://example.com/%s</link><pubDate>Mon, 03 Jul 2023 10:00:00 GMT</pubDate></item>`, guid, title, guid)
}

type feedFixture struct {
	t        *testing.T
	srv      *httptest.Server
	feedRepo *database.FeedRepository
	itemRepo *database.ItemRepository
	jobRepo  *database.JobRepository
	body     string
}

func newFeedFixture(t *testing.T, feedType string, settings types.Settings) *feedFixture {
	t.Helper()
	db, err := database.NewConnection(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, _, err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	f := &feedFixture{
		t:        t,
		feedRepo: database.NewFeedRepository(db),
		itemRepo: database.NewItemRepository(db),
		jobRepo:  database.NewJobRepository(db),
	}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<?xml version="1.0"?><rss version="2.0"><channel><title>T</title><link>https://example.com</link>%s</channel></rss>`, f.body)
	}))
	t.Cleanup(f.srv.Close)

	settings.RefreshInterval, settings.MaxItems, settings.Timeout = 1800, 10, 5
	if err := f.feedRepo.UpsertFeedConfig("f", f.srv.URL, "", feedType, true, settings, []types.Filter{}, "h"); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *feedFixture) process(body string) {
	f.t.Helper()
	f.body = body
	if err := processFeed(context.Background(), "f", f.feedRepo, f.itemRepo, f.jobRepo, f.srv.Client(), "test"); err != nil {
		f.t.Fatal(err)
	}
}

func (f *feedFixture) drainJobs() int {
	f.t.Helper()
	count := 0
	for {
		job, err := f.jobRepo.ClaimJob()
		if err != nil {
			f.t.Fatal(err)
		}
		if job == nil {
			return count
		}
		if err := f.jobRepo.CompleteJob(job.ID); err != nil {
			f.t.Fatal(err)
		}
		count++
	}
}

func (f *feedFixture) item(guid string) database.Item {
	f.t.Helper()
	items, err := f.itemRepo.GetAllItems("f")
	if err != nil {
		f.t.Fatal(err)
	}
	for _, item := range items {
		if item.GUID == guid {
			return item
		}
	}
	f.t.Fatalf("item %q not stored", guid)
	return database.Item{}
}

func TestProcessFeedFindsNewItemBelowKnownFirstItem(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{})

	f.process(rssItem("pinned", "Pinned"))
	f.process(rssItem("pinned", "Pinned") + rssItem("new", "New"))

	f.item("new")
}

func TestProcessFeedEditedTitleKeepsReadyMedia(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})

	first := rssItem("first", "First")
	f.process(first + rssItem("lower", "Old title"))
	if got := f.drainJobs(); got != 2 {
		t.Fatalf("created %d jobs, want 2", got)
	}

	lower := f.item("lower")
	if err := f.itemRepo.UpdateMediaStatus(lower.ID, "ready", "lower.mp3", 123, 456); err != nil {
		t.Fatal(err)
	}

	f.process(first + rssItem("lower", "New title"))

	got := f.item("lower")
	if got.Title != "New title" {
		t.Errorf("title = %q, want the edit applied", got.Title)
	}
	if got.MediaStatus == nil || *got.MediaStatus != "ready" {
		t.Errorf("media_status = %v, want ready", got.MediaStatus)
	}
	if got.MediaPath != "lower.mp3" || got.MediaSize != 123 || got.ITunesDuration != 456 {
		t.Errorf("media = %q/%d/%d, want it kept", got.MediaPath, got.MediaSize, got.ITunesDuration)
	}
	if n := f.drainJobs(); n != 0 {
		t.Errorf("created %d jobs for a processed item, want 0", n)
	}
}

func TestProcessFeedEditedTitleKeepsExtractedContent(t *testing.T) {
	f := newFeedFixture(t, "", types.Settings{ExtractContent: true})

	f.process(rssItem("a", "Old title"))
	if got := f.drainJobs(); got != 1 {
		t.Fatalf("created %d jobs, want 1", got)
	}
	if err := f.itemRepo.UpdateContentExtractionStatus(f.item("a").ID, "ready", "extracted"); err != nil {
		t.Fatal(err)
	}

	f.process(rssItem("a", "New title"))

	got := f.item("a")
	if got.Title != "New title" || got.Content != "extracted" {
		t.Errorf("title/content = %q/%q, want edit applied and content kept", got.Title, got.Content)
	}
	if n := f.drainJobs(); n != 0 {
		t.Errorf("created %d jobs for a processed item, want 0", n)
	}
}

func TestProcessFeedDoesNotRequeueItemEditedIntoFilter(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})
	settings := types.Settings{RefreshInterval: 1800, MaxItems: 10, Timeout: 5}
	filters := []types.Filter{{Field: "title", Excludes: []string{"banned"}}}
	if err := f.feedRepo.UpsertFeedConfig("f", f.srv.URL, "", "youtube", true, settings, filters, "h2"); err != nil {
		t.Fatal(err)
	}

	f.process(rssItem("a", "Old title"))
	f.drainJobs() // the job is lost, as after a crash; the item stays pending

	f.process(rssItem("a", "Now banned"))

	if n := f.drainJobs(); n != 0 {
		t.Errorf("created %d jobs for a filtered item, want 0", n)
	}
}

func TestProcessFeedRequeuesPendingItemAfterTitleEdit(t *testing.T) {
	f := newFeedFixture(t, "youtube", types.Settings{})

	f.process(rssItem("a", "Old title"))
	f.drainJobs() // the job is lost, as after a crash; the item stays pending

	f.process(rssItem("a", "New title"))

	if n := f.drainJobs(); n != 1 {
		t.Errorf("created %d jobs, want 1", n)
	}
}
