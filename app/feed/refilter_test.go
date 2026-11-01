package feed

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/types"
)

func TestRefilterWithNoFiltersClearsFilteredItems(t *testing.T) {
	db, err := database.NewConnection(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, _, err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	feedRepo := database.NewFeedRepository(db)
	itemRepo := database.NewItemRepository(db)

	settings := types.Settings{RefreshInterval: 1800, MaxItems: 10, Timeout: 5}
	if err := feedRepo.UpsertFeedConfig("f", "https://example.com", "", "", true, settings, []types.Filter{}, "h"); err != nil {
		t.Fatal(err)
	}
	if _, err := itemRepo.UpsertItem("f", types.Item{GUID: "a", Title: "a", ContentHash: "a", PublishedAt: time.Unix(0, 0), IsFiltered: true}); err != nil {
		t.Fatal(err)
	}

	if err := Refilter(context.Background(), "f", feedRepo, itemRepo); err != nil {
		t.Fatal(err)
	}

	items, err := itemRepo.GetAllItems("f")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].IsFiltered {
		t.Errorf("items = %+v, want one unfiltered item", items)
	}
}
