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
	"github.com/lysyi3m/rss-comb/app/jobs"
	"github.com/lysyi3m/rss-comb/app/media"
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

	hasMediaFeeds, err := loadFeedConfigurations(cfg.FeedsDir, feedRepo)
	if err != nil {
		slog.Error("Configuration loading failed", "directory", cfg.FeedsDir, "error", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(cfg.MediaDir, 0755); err != nil {
		slog.Error("Failed to create media directory", "path", cfg.MediaDir, "error", err)
		os.Exit(1)
	}

	if hasMediaFeeds {
		if cfg.YTDLPUpdate {
			slog.Info("Updating yt-dlp...")
			if err := media.Update(cfg.YTDLPCmd); err != nil {
				slog.Warn("yt-dlp update failed, continuing with current version", "error", err)
			}
		}
		if err := media.Validate(cfg.YTDLPCmd); err != nil {
			slog.Error("yt-dlp validation failed — media feeds are configured but yt-dlp is not available", "error", err)
			os.Exit(1)
		}
		slog.Info("yt-dlp validated", "command", cfg.YTDLPCmd)
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

	jobRepo := database.NewJobRepository(db)

	pool := jobs.NewWorkerPool(jobRepo, cfg.WorkerCount)
	pool.RegisterHandler("fetch_feed", jobs.FetchFeedHandler(feedRepo, itemRepo, jobRepo, httpClient, cfg.UserAgent, cfg.MediaDir))
	pool.RegisterHandler("extract_content", jobs.ExtractContentHandler(feedRepo, itemRepo, httpClient, cfg.UserAgent))
	pool.RegisterHandler("download_media", jobs.DownloadMediaHandler(feedRepo, itemRepo, cfg.YTDLPCmd, cfg.YTDLPArgs, cfg.MediaDir))

	scheduler := jobs.NewScheduler(time.Duration(cfg.SchedulerInterval)*time.Second, feedRepo, jobRepo)

	jobCtx, jobCancel := context.WithCancel(context.Background())
	var jobWg sync.WaitGroup
	jobWg.Add(1)
	go func() {
		defer jobWg.Done()
		scheduler.Run(jobCtx)
	}()
	pool.Start(jobCtx)
	defer func() {
		jobCancel()
		pool.Wait()
		jobWg.Wait()
	}()

	apiHandler := api.NewHandler(cfg, feedRepo, itemRepo, jobRepo)
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

func loadFeedConfigurations(feedsDir string, feedRepo *database.FeedRepository) (bool, error) {
	if _, err := os.Stat(feedsDir); os.IsNotExist(err) {
		slog.Info("Feeds directory does not exist, skipping config loading", "directory", feedsDir)
		return false, nil
	}

	files, err := filepath.Glob(filepath.Join(feedsDir, "*.yml"))
	if err != nil {
		return false, fmt.Errorf("failed to find YAML files: %w", err)
	}

	if len(files) == 0 {
		slog.Info("No feed configuration files found", "directory", feedsDir)
		return false, nil
	}

	var totalCount int
	var enabledCount int
	var enabledNames []string
	var hasMediaFeeds bool

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
			if config.Settings.ExtractMedia {
				hasMediaFeeds = true
			}
		}
	}

	slog.Info("Configuration loaded",
		"total", totalCount,
		"enabled", enabledCount,
		"feeds", enabledNames,
		"directory", feedsDir)

	return hasMediaFeeds, nil
}
