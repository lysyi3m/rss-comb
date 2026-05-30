package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/types"
)

// newTestDB opens a fresh temp-file SQLite database and runs the real
// embedded migrations against it. Each test gets an isolated database.
func newTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := NewConnection(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	version, dirty, err := RunMigrations(db)
	if err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if dirty {
		t.Fatal("migrations left database in dirty state")
	}
	// Confirm migrations actually applied without pinning a specific version,
	// so adding a future migration doesn't break every database test.
	if version == 0 {
		t.Fatal("no migrations were applied")
	}
	return db
}

// seedFeed inserts a basic enabled feed config and returns its DB id.
func seedFeed(t *testing.T, db *DB, name string) string {
	t.Helper()
	repo := NewFeedRepository(db)
	settings := types.Settings{RefreshInterval: 1800, MaxItems: 10, Timeout: 30}
	if err := repo.UpsertFeedConfig(name, "https://example.com/feed.xml", "Custom Title", "", true, settings, []types.Filter{}, "hash-"+name); err != nil {
		t.Fatalf("UpsertFeedConfig: %v", err)
	}
	feed, err := repo.GetFeed(name)
	if err != nil {
		t.Fatalf("GetFeed: %v", err)
	}
	if feed == nil {
		t.Fatal("seeded feed not found")
	}
	return feed.ID
}

// TestFeedConfigRoundTrip guards the regression where timestamp columns
// declared as TEXT could not be scanned into time.Time (modernc only
// auto-parses DATETIME/TIMESTAMP columns). GetFeed scans created_at/updated_at.
func TestFeedConfigRoundTrip(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeedRepository(db)

	settings := types.Settings{RefreshInterval: 900, MaxItems: 25, Timeout: 30}
	if err := repo.UpsertFeedConfig("acme", "https://acme.test/rss", "Acme Custom", "podcast", true, settings, []types.Filter{}, "h1"); err != nil {
		t.Fatalf("UpsertFeedConfig: %v", err)
	}

	feed, err := repo.GetFeed("acme")
	if err != nil {
		t.Fatalf("GetFeed: %v", err)
	}
	if feed == nil {
		t.Fatal("feed not found after upsert")
	}

	if feed.Name != "acme" || feed.FeedURL != "https://acme.test/rss" {
		t.Errorf("unexpected feed identity: name=%q url=%q", feed.Name, feed.FeedURL)
	}
	if feed.DisplayTitle() != "Acme Custom" {
		t.Errorf("DisplayTitle = %q, want %q", feed.DisplayTitle(), "Acme Custom")
	}
	if feed.FeedType != "podcast" || !feed.IsEnabled {
		t.Errorf("feed_type=%q is_enabled=%v", feed.FeedType, feed.IsEnabled)
	}
	// The core regression assertion: timestamps must scan into time.Time.
	if feed.CreatedAt.IsZero() || feed.UpdatedAt.IsZero() {
		t.Errorf("timestamps not parsed: created_at=%v updated_at=%v", feed.CreatedAt, feed.UpdatedAt)
	}

	got, err := feed.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if got.MaxItems != 25 || got.RefreshInterval != 900 {
		t.Errorf("settings round-trip mismatch: %+v", got)
	}

	count, err := repo.GetFeedCount()
	if err != nil {
		t.Fatalf("GetFeedCount: %v", err)
	}
	if count != 1 {
		t.Errorf("GetFeedCount = %d, want 1", count)
	}

	missing, err := repo.GetFeed("does-not-exist")
	if err != nil {
		t.Fatalf("GetFeed(missing): %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for missing feed, got %+v", missing)
	}
}

// TestItemRoundTrip verifies item upsert/read, JSON array columns, and that
// published_at round-trips with second precision through DATETIME storage.
func TestItemRoundTrip(t *testing.T) {
	db := newTestDB(t)
	seedFeed(t, db, "feed1")
	repo := NewItemRepository(db)

	pub := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)
	item := types.Item{
		GUID:        "guid-1",
		Title:       "Hello World",
		Link:        "https://example.com/1",
		Description: "desc",
		PublishedAt: pub,
		Authors:     []string{"alice", "bob"},
		Categories:  []string{"tech"},
		ContentHash: "chash-1",
	}
	id, err := repo.UpsertItem("feed1", item)
	if err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	if id == "" {
		t.Fatal("UpsertItem returned empty id")
	}

	got, err := repo.GetItemByID(id)
	if err != nil {
		t.Fatalf("GetItemByID: %v", err)
	}
	if got == nil {
		t.Fatal("item not found by id")
	}
	if !got.PublishedAt.Equal(pub) {
		t.Errorf("published_at round-trip: got %v, want %v", got.PublishedAt, pub)
	}
	if got.Title != "Hello World" || got.GUID != "guid-1" {
		t.Errorf("item identity mismatch: %+v", got)
	}
	if len(got.Authors) != 2 || got.Authors[0] != "alice" {
		t.Errorf("authors JSON round-trip mismatch: %v", got.Authors)
	}
	if len(got.Categories) != 1 || got.Categories[0] != "tech" {
		t.Errorf("categories JSON round-trip mismatch: %v", got.Categories)
	}

	visible, err := repo.GetVisibleItems("feed1", 10)
	if err != nil {
		t.Fatalf("GetVisibleItems: %v", err)
	}
	if len(visible) != 1 {
		t.Fatalf("GetVisibleItems len = %d, want 1", len(visible))
	}
	if !visible[0].PublishedAt.Equal(pub) {
		t.Errorf("visible published_at: got %v, want %v", visible[0].PublishedAt, pub)
	}
}

// TestVisibleItemsNullableColumns guards the COALESCE fix in GetVisibleItems:
// a row with NULL enclosure columns must scan without error.
func TestVisibleItemsNullableColumns(t *testing.T) {
	db := newTestDB(t)
	feedID := seedFeed(t, db, "feed1")
	repo := NewItemRepository(db)

	pub := sqliteTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	// Insert directly leaving enclosure_* (and other nullable columns) NULL —
	// this is the shape that broke scanning before COALESCE was added.
	_, err := db.Exec(`
		INSERT INTO feed_items (feed_id, guid, title, published_at, content_hash)
		VALUES (?1, ?2, ?3, ?4, ?5)
	`, feedID, "raw-guid", "Raw Item", pub, "raw-hash")
	if err != nil {
		t.Fatalf("raw insert: %v", err)
	}

	visible, err := repo.GetVisibleItems("feed1", 10)
	if err != nil {
		t.Fatalf("GetVisibleItems with NULL enclosure columns: %v", err)
	}
	if len(visible) != 1 {
		t.Fatalf("expected 1 visible item, got %d", len(visible))
	}
	if visible[0].EnclosureLength != 0 {
		t.Errorf("NULL enclosure_length should COALESCE to 0, got %d", visible[0].EnclosureLength)
	}
}

// TestItemDedupFilterAndPublishedAtUpdate covers CheckDuplicate, filter
// visibility, and UpdateItemPublishedAt (which must store via sqliteTime).
func TestItemDedupFilterAndPublishedAtUpdate(t *testing.T) {
	db := newTestDB(t)
	seedFeed(t, db, "feed1")
	repo := NewItemRepository(db)

	if dup, _, err := repo.CheckDuplicate("feed1", "chash-1"); err != nil || dup {
		t.Fatalf("CheckDuplicate before insert: dup=%v err=%v", dup, err)
	}

	id, err := repo.UpsertItem("feed1", types.Item{
		GUID:        "guid-1",
		Title:       "Item",
		PublishedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
		ContentHash: "chash-1",
	})
	if err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}

	dup, dupID, err := repo.CheckDuplicate("feed1", "chash-1")
	if err != nil || !dup {
		t.Fatalf("CheckDuplicate after insert: dup=%v err=%v", dup, err)
	}
	if dupID == nil || *dupID != id {
		t.Errorf("CheckDuplicate id = %v, want %s", dupID, id)
	}

	// UpdateItemPublishedAt must store a value scannable back into time.Time.
	newPub := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	if err := repo.UpdateItemPublishedAt(id, newPub); err != nil {
		t.Fatalf("UpdateItemPublishedAt: %v", err)
	}
	got, err := repo.GetItemByID(id)
	if err != nil {
		t.Fatalf("GetItemByID: %v", err)
	}
	if !got.PublishedAt.Equal(newPub) {
		t.Errorf("published_at after update: got %v, want %v", got.PublishedAt, newPub)
	}

	// Filtering should hide the item from the visible query.
	if err := repo.UpdateItemFilterStatus(id, true); err != nil {
		t.Fatalf("UpdateItemFilterStatus: %v", err)
	}
	visible, err := repo.GetVisibleItems("feed1", 10)
	if err != nil {
		t.Fatalf("GetVisibleItems: %v", err)
	}
	if len(visible) != 0 {
		t.Errorf("filtered item should be hidden, got %d visible", len(visible))
	}
}

// TestJobQueueClaim guards the ClaimJob UPDATE...RETURNING path, which scans
// created_at/updated_at — the exact failure surfaced by the smoke test.
func TestJobQueueClaim(t *testing.T) {
	db := newTestDB(t)
	feedID := seedFeed(t, db, "feed1")
	repo := NewJobRepository(db)

	created, err := repo.CreateJob("fetch_feed", feedID, nil, 0)
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if !created {
		t.Fatal("first CreateJob should report created=true")
	}

	// Duplicate (same feed+type+nil item, still pending) must be deduped.
	dup, err := repo.CreateJob("fetch_feed", feedID, nil, 0)
	if err != nil {
		t.Fatalf("CreateJob dup: %v", err)
	}
	if dup {
		t.Error("duplicate pending job should not be created")
	}

	job, err := repo.ClaimJob()
	if err != nil {
		t.Fatalf("ClaimJob: %v", err)
	}
	if job == nil {
		t.Fatal("ClaimJob returned nil for a pending job")
	}
	if job.JobType != "fetch_feed" || job.FeedID != feedID {
		t.Errorf("claimed job mismatch: type=%q feed=%q", job.JobType, job.FeedID)
	}
	if job.ItemID != nil {
		t.Errorf("expected nil item id, got %v", *job.ItemID)
	}
	// RETURNING timestamp columns must scan into time.Time.
	if job.CreatedAt.IsZero() || job.UpdatedAt.IsZero() {
		t.Errorf("job timestamps not parsed: created=%v updated=%v", job.CreatedAt, job.UpdatedAt)
	}

	// No more pending jobs (the only one is now processing).
	again, err := repo.ClaimJob()
	if err != nil {
		t.Fatalf("ClaimJob (second): %v", err)
	}
	if again != nil {
		t.Errorf("expected no claimable job, got %v", again.ID)
	}

	if err := repo.CompleteJob(job.ID); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
}

// TestJobFailAndRequeue verifies retry/backoff bookkeeping and exhaustion
// deletion, exercising run_after (DATETIME) reads.
func TestJobFailAndRequeue(t *testing.T) {
	db := newTestDB(t)
	feedID := seedFeed(t, db, "feed1")
	repo := NewJobRepository(db)

	// Requeue path: maxRetries high enough that the job survives one failure.
	if _, err := repo.CreateJob("download_media", feedID, nil, 3); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	job, err := repo.ClaimJob()
	if err != nil || job == nil {
		t.Fatalf("ClaimJob: job=%v err=%v", job, err)
	}
	if err := repo.FailJob(job.ID, "boom"); err != nil {
		t.Fatalf("FailJob: %v", err)
	}

	var (
		retries  int
		status   string
		runAfter *time.Time
	)
	err = db.QueryRow(`SELECT retries, status, run_after FROM jobs WHERE id = ?1`, job.ID).
		Scan(&retries, &status, &runAfter)
	if err != nil {
		t.Fatalf("read failed job: %v", err)
	}
	if retries != 1 {
		t.Errorf("retries = %d, want 1", retries)
	}
	if status != "pending" {
		t.Errorf("status = %q, want pending", status)
	}
	if runAfter == nil || runAfter.IsZero() {
		t.Errorf("run_after should be set after failure, got %v", runAfter)
	}

	// Exhaustion path: maxRetries=0 means a single failure deletes the job.
	if _, err := repo.CreateJob("fetch_feed", feedID, nil, 0); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	job2, err := repo.ClaimJob()
	if err != nil || job2 == nil {
		t.Fatalf("ClaimJob: job=%v err=%v", job2, err)
	}
	if err := repo.FailJob(job2.ID, "fatal"); err != nil {
		t.Fatalf("FailJob: %v", err)
	}
	var exists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE id = ?1`, job2.ID).Scan(&exists); err != nil {
		t.Fatalf("count exhausted job: %v", err)
	}
	if exists != 0 {
		t.Errorf("exhausted job should be deleted, still present")
	}
}

// TestResetStaleJobs verifies a processing job past the timeout is returned to
// pending and becomes claimable again.
func TestResetStaleJobs(t *testing.T) {
	db := newTestDB(t)
	feedID := seedFeed(t, db, "feed1")
	repo := NewJobRepository(db)

	if _, err := repo.CreateJob("fetch_feed", feedID, nil, 0); err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if _, err := repo.ClaimJob(); err != nil {
		t.Fatalf("ClaimJob: %v", err)
	}

	// Negative timeout moves the staleness threshold into the future, so even a
	// just-claimed job counts as stale — avoids a real sleep in the test.
	n, err := repo.ResetStaleJobs(-time.Hour)
	if err != nil {
		t.Fatalf("ResetStaleJobs: %v", err)
	}
	if n != 1 {
		t.Errorf("ResetStaleJobs reset %d jobs, want 1", n)
	}

	reclaimed, err := repo.ClaimJob()
	if err != nil {
		t.Fatalf("ClaimJob after reset: %v", err)
	}
	if reclaimed == nil {
		t.Error("stale job should be claimable again after reset")
	}
}

// TestActiveMediaPathsRetention covers GetAllActiveMediaPaths, which reads
// max_items from the settings JSON via json_extract and retains only the newest
// items per feed. It also guards that the stored settings JSON stays readable by
// SQLite's JSON functions (settings are bound as []byte, so the storage class may
// be BLOB; this confirms json_extract still works against it).
func TestActiveMediaPathsRetention(t *testing.T) {
	db := newTestDB(t)
	feedRepo := NewFeedRepository(db)
	if err := feedRepo.UpsertFeedConfig("yt", "https://yt.test/feed", "", "youtube", true,
		types.Settings{RefreshInterval: 1800, MaxItems: 2, Timeout: 30}, []types.Filter{}, "hash-yt"); err != nil {
		t.Fatalf("UpsertFeedConfig: %v", err)
	}
	itemRepo := NewItemRepository(db)
	ready := "ready"

	mk := func(guid, path string, day int, filtered bool) {
		_, err := itemRepo.UpsertItem("yt", types.Item{
			GUID:        guid,
			Title:       guid,
			PublishedAt: time.Date(2026, 3, day, 0, 0, 0, 0, time.UTC),
			ContentHash: "h-" + guid,
			MediaStatus: &ready,
			MediaPath:   path,
			IsFiltered:  filtered,
		})
		if err != nil {
			t.Fatalf("UpsertItem %s: %v", guid, err)
		}
	}
	mk("v1", "v1.mp3", 1, false) // oldest -> evicted by max_items=2
	mk("v2", "v2.mp3", 2, false)
	mk("v3", "v3.mp3", 3, false)
	mk("v4", "v4.mp3", 4, true) // filtered -> excluded despite being newest

	paths, err := itemRepo.GetAllActiveMediaPaths()
	if err != nil {
		t.Fatalf("GetAllActiveMediaPaths: %v", err)
	}

	got := map[string]bool{}
	for _, p := range paths {
		got[p] = true
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 retained paths (max_items=2), got %d: %v", len(paths), paths)
	}
	if !got["v2.mp3"] || !got["v3.mp3"] {
		t.Errorf("expected the two newest (v2.mp3, v3.mp3) retained, got %v", paths)
	}
	if got["v1.mp3"] {
		t.Error("item beyond max_items should not be retained")
	}
	if got["v4.mp3"] {
		t.Error("filtered item should be excluded")
	}
}

// TestJobItemSpecificDedup covers CreateJob/ClaimJob for jobs tied to a specific
// item (non-nil item_id), exercising the `item_id IS ?3` predicate and pointer
// parameter handling that differ from feed-level (nil item_id) jobs.
func TestJobItemSpecificDedup(t *testing.T) {
	db := newTestDB(t)
	feedID := seedFeed(t, db, "feed1")
	itemRepo := NewItemRepository(db)
	itemID, err := itemRepo.UpsertItem("feed1", types.Item{
		GUID:        "guid-1",
		Title:       "Item",
		PublishedAt: time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
		ContentHash: "chash-1",
	})
	if err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}

	jobRepo := NewJobRepository(db)
	created, err := jobRepo.CreateJob("download_media", feedID, &itemID, 3)
	if err != nil || !created {
		t.Fatalf("first CreateJob: created=%v err=%v", created, err)
	}
	dup, err := jobRepo.CreateJob("download_media", feedID, &itemID, 3)
	if err != nil {
		t.Fatalf("dup CreateJob: %v", err)
	}
	if dup {
		t.Error("duplicate item-specific job should not be created")
	}

	job, err := jobRepo.ClaimJob()
	if err != nil || job == nil {
		t.Fatalf("ClaimJob: job=%v err=%v", job, err)
	}
	if job.ItemID == nil || *job.ItemID != itemID {
		t.Errorf("claimed job item id = %v, want %s", job.ItemID, itemID)
	}
	if job.JobType != "download_media" {
		t.Errorf("job type = %q, want download_media", job.JobType)
	}
}

// TestUpdateFeedMetadataRoundTrip guards the UpdateFeedMetadata write path, which
// stores nullable DATETIME columns (feed_published_at/feed_updated_at/
// next_fetch_at) via the sqliteTime helpers and later scans them back into
// *time.Time — the path that reads non-null nullable timestamps.
func TestUpdateFeedMetadataRoundTrip(t *testing.T) {
	db := newTestDB(t)
	repo := NewFeedRepository(db)
	seedFeed(t, db, "feed1")

	pub := time.Date(2026, 2, 10, 8, 0, 0, 0, time.UTC)
	upd := time.Date(2026, 2, 11, 9, 30, 0, 0, time.UTC)
	next := time.Date(2026, 2, 12, 10, 0, 0, 0, time.UTC)
	meta := &types.Metadata{
		Title:           "Source Title",
		Link:            "https://feed1.test",
		Description:     "desc",
		Language:        "en",
		FeedPublishedAt: &pub,
		FeedUpdatedAt:   &upd,
	}
	if err := repo.UpdateFeedMetadata("feed1", meta, next); err != nil {
		t.Fatalf("UpdateFeedMetadata: %v", err)
	}

	feed, err := repo.GetFeed("feed1")
	if err != nil {
		t.Fatalf("GetFeed: %v", err)
	}
	if feed.SourceTitle != "Source Title" || feed.Link != "https://feed1.test" {
		t.Errorf("metadata not stored: title=%q link=%q", feed.SourceTitle, feed.Link)
	}
	if feed.FeedPublishedAt == nil || !feed.FeedPublishedAt.Equal(pub) {
		t.Errorf("feed_published_at round-trip: got %v, want %v", feed.FeedPublishedAt, pub)
	}
	if feed.FeedUpdatedAt == nil || !feed.FeedUpdatedAt.Equal(upd) {
		t.Errorf("feed_updated_at round-trip: got %v, want %v", feed.FeedUpdatedAt, upd)
	}
	if feed.NextFetchAt == nil || !feed.NextFetchAt.Equal(next) {
		t.Errorf("next_fetch_at round-trip: got %v, want %v", feed.NextFetchAt, next)
	}
	if feed.LastFetchedAt == nil {
		t.Error("last_fetched_at should be set by UpdateFeedMetadata")
	}
}
