package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/lysyi3m/rss-comb/app/api"
	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/services"
)

func main() {
	cfg, err := cfg.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		// Help was requested, exit gracefully
		return
	}

	initializeLogger()

	slog.Info("Starting RSS Comb server", "version", cfg.Version)

	db, err := database.NewConnection(
		cfg.DBHost, cfg.DBPort, cfg.DBUser,
		cfg.DBPassword, cfg.DBName)
	if err != nil {
		slog.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Database connected", "host", cfg.DBHost, "port", cfg.DBPort)

	version, dirty, err := database.RunMigrations(db)
	if err != nil {
		slog.Error("Database migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Database migrations completed", "version", version, "dirty", dirty)

	feedRepo := database.NewFeedRepository(db)
	itemRepo := database.NewItemRepository(db)

	if err := loadFeedConfigurations(cfg.FeedsDir, feedRepo); err != nil {
		slog.Error("Configuration loading failed", "directory", cfg.FeedsDir, "error", err)
		os.Exit(1)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 5,
		},
	}

	tickerCtx, tickerCancel := context.WithCancel(context.Background())
	var tickerWg sync.WaitGroup
	tickerWg.Add(1)
	go func() {
		defer tickerWg.Done()
		processFeedsTicker(tickerCtx, cfg, feedRepo, itemRepo, httpClient)
	}()
	defer func() {
		tickerCancel()
		tickerWg.Wait()
	}()

	apiHandler := api.NewHandler(cfg, feedRepo, itemRepo)
	server := api.NewServer(apiHandler, cfg)
	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      server,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	serverErrChan := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()
	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	slog.Info("Server started successfully", "port", cfg.Port, "api_enabled", cfg.APIAccessKey != "")

	select {
	case sig := <-sigChan:
		slog.Info("Shutdown signal received", "signal", sig)
	case err := <-serverErrChan:
		slog.Error("Server error occurred", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}
}

func initializeLogger() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("2006-01-02 15:04:05"))
			}
			return a
		},
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func loadFeedConfigurations(feedsDir string, feedRepo *database.FeedRepository) error {
	if _, err := os.Stat(feedsDir); os.IsNotExist(err) {
		slog.Info("Feeds directory does not exist, skipping config loading", "directory", feedsDir)
		return nil
	}

	files, err := filepath.Glob(filepath.Join(feedsDir, "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to find YAML files: %w", err)
	}

	if len(files) == 0 {
		slog.Info("No feed configuration files found", "directory", feedsDir)
		return nil
	}

	var totalCount int
	var enabledCount int
	var enabledNames []string

	for _, file := range files {
		fileName := filepath.Base(file)
		feedName := fileName[:len(fileName)-4]

		config, err := services.SyncFeedConfig(context.Background(), feedsDir, feedName, feedRepo)
		if err != nil {
			slog.Warn("Failed to sync feed config, skipping", "file", file, "error", err)
			continue
		}

		totalCount++
		if config.Enabled {
			enabledCount++
			enabledNames = append(enabledNames, feedName)
		}
	}

	slog.Info("Configuration loaded",
		"total", totalCount,
		"enabled", enabledCount,
		"feeds", enabledNames,
		"directory", feedsDir)

	return nil
}

func processFeedsTicker(
	ctx context.Context,
	cfg *cfg.Cfg,
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	httpClient *http.Client,
) {
	ticker := time.NewTicker(time.Duration(cfg.SchedulerInterval) * time.Second)
	defer ticker.Stop()

	slog.Info("Feed processing ticker started", "interval_seconds", cfg.SchedulerInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("Feed processing ticker stopped")
			return
		case <-ticker.C:
			processDueFeeds(ctx, cfg, feedRepo, itemRepo, httpClient)
		}
	}
}

func processDueFeeds(
	ctx context.Context,
	cfg *cfg.Cfg,
	feedRepo *database.FeedRepository,
	itemRepo *database.ItemRepository,
	httpClient *http.Client,
) {
	feeds, err := feedRepo.GetDueFeeds()
	if err != nil {
		slog.Error("Failed to get due feeds", "error", err)
		return
	}

	for _, feed := range feeds {
		err := services.ProcessFeed(
			ctx,
			feed.Name,
			feedRepo,
			itemRepo,
			httpClient,
			cfg.UserAgent,
		)
		if err != nil {
			slog.Error("Failed to process feed", "feed", feed.Name, "error", err)
		}
	}
}
