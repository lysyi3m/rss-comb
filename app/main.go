package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lysyi3m/rss-comb/app/api"
	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/feed_config"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/tasks"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		// Help was requested, exit gracefully
		return
	}

	initializeLogger(cfg.IsDebugEnabled())

	slog.Info("Starting RSS Comb server", "version", cfg.GetVersion())
	db, err := database.NewConnection(
		cfg.GetDBHost(), cfg.GetDBPort(), cfg.GetDBUser(),
		cfg.GetDBPassword(), cfg.GetDBName())
	if err != nil {
		slog.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Migrations run automatically unless explicitly disabled for faster startup
	if !cfg.IsMigrationDisabled() {
		if err := database.RunMigrations(db); err != nil {
			slog.Error("Database migration failed", "error", err)
			os.Exit(1)
		}

		version, dirty, err := database.GetMigrationVersion(db)
		if err != nil {
			slog.Warn("Failed to get migration version", "error", err)
		} else {
			slog.Info("Database migrations completed", "version", version, "dirty", dirty)
		}
	} else {
		slog.Debug("Auto-migration disabled")
	}


	configLoader := feed_config.NewLoader(cfg.GetFeedsDir())
	configs, err := configLoader.LoadAll()
	if err != nil {
		slog.Error("Configuration loading failed", "directory", cfg.GetFeedsDir(), "error", err)
		os.Exit(1)
	}
	configCache := feed_config.NewConfigCacheHandler(configs)
	slog.Info("Configuration loaded", "feeds", len(configs), "directory", cfg.GetFeedsDir())

	feedRepo := database.NewFeedRepository(db)
	itemRepo := database.NewItemRepository(db)

	feedProcessor := feed.NewProcessor(feedRepo, itemRepo)
	contentExtractionService := feed.NewContentExtractionService(itemRepo)

	slog.Info("Starting task scheduler", "workers", cfg.GetWorkerCount(), "interval", fmt.Sprintf("%ds", cfg.GetSchedulerInterval()))
	taskScheduler := tasks.NewTaskScheduler(configCache, feedRepo, feedProcessor, contentExtractionService)
	taskScheduler.Start()
	defer taskScheduler.Stop()

	apiHandler := api.NewHandler(configCache, feedRepo, itemRepo, feedProcessor, taskScheduler)

	server := api.NewServer(apiHandler)

	httpServer := &http.Server{
		Addr:         ":" + cfg.GetPort(),
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

	slog.Info("Server started successfully", "port", cfg.GetPort(), "api_enabled", cfg.GetAPIAccessKey() != "")

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


// initializeLogger sets up the global logger with appropriate configuration
func initializeLogger(debug bool) {
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Simplify time format for better readability
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
