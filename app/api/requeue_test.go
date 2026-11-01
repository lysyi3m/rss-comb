package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/types"
)

func TestHealthReportsDatabaseFailure(t *testing.T) {
	db, err := database.NewConnection(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, _, err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	cfgValue := &cfg.Cfg{Location: time.UTC, MediaDir: t.TempDir()}
	server := NewServer(NewHandler(cfgValue, database.NewFeedRepository(db), nil, nil), cfgValue)
	status := func() int {
		rec := httptest.NewRecorder()
		server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
		return rec.Code
	}

	if got := status(); got != http.StatusOK {
		t.Errorf("status = %d, want 200", got)
	}
	db.Close()
	if got := status(); got != http.StatusServiceUnavailable {
		t.Errorf("status with a closed database = %d, want 503", got)
	}
}

func TestRequeueMediaEndpoint(t *testing.T) {
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
	jobRepo := database.NewJobRepository(db)

	settings := types.Settings{RefreshInterval: 1800, MaxItems: 10, Timeout: 5}
	if err := feedRepo.UpsertFeedConfig("yt", "https://example.com/yt", "", "youtube", true, settings, []types.Filter{}, "y"); err != nil {
		t.Fatal(err)
	}
	if err := feedRepo.UpsertFeedConfig("plain", "https://example.com/p", "", "", true, settings, []types.Filter{}, "p"); err != nil {
		t.Fatal(err)
	}
	failed := "failed"
	if _, err := itemRepo.UpsertItem("yt", types.Item{GUID: "a", ContentHash: "a", PublishedAt: time.Unix(0, 0), MediaStatus: &failed}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(NewHandler(&cfg.Cfg{}, feedRepo, itemRepo, jobRepo), &cfg.Cfg{APIAccessKey: "key", MediaDir: t.TempDir()})

	tests := []struct {
		name       string
		feed       string
		key        string
		wantStatus int
		wantBody   string
	}{
		{"no key", "yt", "", http.StatusUnauthorized, ""},
		{"unknown feed", "nope", "key", http.StatusNotFound, ""},
		{"feed without media", "plain", "key", http.StatusBadRequest, ""},
		{"youtube feed", "yt", "key", http.StatusOK, `"requeued":1`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/feeds/"+tt.feed+"/requeue-media", nil)
			if tt.key != "" {
				req.Header.Set("X-API-Key", tt.key)
			}
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.wantBody) {
				t.Errorf("body = %s, want %s", rec.Body.String(), tt.wantBody)
			}
		})
	}
}
