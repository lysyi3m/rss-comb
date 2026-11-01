package feed

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
)

const validConfig = "url: https://example.com/feed.xml\nenabled: true\n"

type configFixture struct {
	t    *testing.T
	dir  string
	repo *database.FeedRepository
}

func newConfigFixture(t *testing.T, files map[string]string) *configFixture {
	t.Helper()
	db, err := database.NewConnection(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if _, _, err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	f := &configFixture{t: t, dir: t.TempDir(), repo: database.NewFeedRepository(db)}
	for name, body := range files {
		f.write(name, body)
		if _, err := ConfigSync(context.Background(), f.dir, name, f.repo); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f *configFixture) write(name, body string) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, name+".yml"), []byte(body), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *configFixture) remove(name string) {
	f.t.Helper()
	if err := os.Remove(filepath.Join(f.dir, name+".yml")); err != nil {
		f.t.Fatal(err)
	}
}

func (f *configFixture) enabled(name string) bool {
	f.t.Helper()
	feed, err := f.repo.GetFeed(name)
	if err != nil {
		f.t.Fatal(err)
	}
	if feed == nil {
		f.t.Fatalf("feed %q is gone", name)
	}
	return feed.IsEnabled
}

func TestDisableRemovedFeedsDisablesOnlyMissingFiles(t *testing.T) {
	f := newConfigFixture(t, map[string]string{"keep": validConfig, "gone": validConfig})
	f.remove("gone")

	removed, err := DisableRemovedFeeds(f.dir, f.repo)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(removed, []string{"gone"}) {
		t.Errorf("removed = %v, want [gone]", removed)
	}
	if !f.enabled("keep") || f.enabled("gone") {
		t.Errorf("keep/gone enabled = %v/%v, want true/false", f.enabled("keep"), f.enabled("gone"))
	}
}

func TestDisableRemovedFeedsKeepsFeedWithBrokenFile(t *testing.T) {
	f := newConfigFixture(t, map[string]string{"broken": validConfig})
	f.write("broken", "url: [not yaml")

	if _, err := DisableRemovedFeeds(f.dir, f.repo); err != nil {
		t.Fatal(err)
	}

	if !f.enabled("broken") {
		t.Error("a feed whose file failed to parse was disabled")
	}
}

func TestDisableRemovedFeedsIgnoresEmptyDirectory(t *testing.T) {
	f := newConfigFixture(t, map[string]string{"a": validConfig})
	f.remove("a")

	if _, err := DisableRemovedFeeds(f.dir, f.repo); err != nil {
		t.Fatal(err)
	}

	if !f.enabled("a") {
		t.Error("an empty directory disabled a feed")
	}
}

func TestRestoredFileEnablesFeedAgain(t *testing.T) {
	f := newConfigFixture(t, map[string]string{"keep": validConfig, "back": validConfig})
	back, err := f.repo.GetFeed("back")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.repo.UpdateNextFetch("back", back.ConfigHash, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	f.remove("back")
	if _, err := DisableRemovedFeeds(f.dir, f.repo); err != nil {
		t.Fatal(err)
	}

	f.write("back", validConfig) // the same bytes, so the same hash
	if _, err := ConfigSync(context.Background(), f.dir, "back", f.repo); err != nil {
		t.Fatal(err)
	}

	if !f.enabled("back") {
		t.Error("restoring an unchanged file did not enable the feed")
	}

	due, err := f.repo.GetDueFeeds()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(due, func(d database.FeedScheduleInfo) bool { return d.Name == "back" }) {
		t.Error("restored feed kept its old future fetch time")
	}
}
