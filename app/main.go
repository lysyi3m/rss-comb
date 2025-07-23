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
	"github.com/lysyi3m/rss-comb/app/cfg"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/tasks"
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

	initializeLogger(cfg.Debug)

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

	configCache := feed.NewConfigCache(cfg.FeedsDir)
	if err := configCache.Run(); err != nil {
		slog.Error("Configuration loading failed", "directory", cfg.FeedsDir, "error", err)
		os.Exit(1)
	}
	slog.Info("Configuration loaded", "total", configCache.GetConfigCount(), "enabled", configCache.GetEnabledConfigs(), "directory", cfg.FeedsDir)

	feedRepo := database.NewFeedRepository(db)
	itemRepo := database.NewItemRepository(db)

	// Create feed processing components
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        10,
			IdleConnTimeout:     30 * time.Second,
			DisableCompression:  false,
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 5,
		},
	}
	parser := feed.NewParser()
	filterer := feed.NewFilterer()
	contentExtractor := feed.NewContentExtractor()

	scheduler := tasks.NewScheduler(configCache, feedRepo, itemRepo, httpClient, parser, filterer, contentExtractor)
	scheduler.Start()
	defer scheduler.Stop()

	apiHandler := api.NewHandler(configCache, feedRepo, itemRepo, filterer, scheduler)
	server := api.NewServer(apiHandler)
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
