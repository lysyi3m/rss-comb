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

	"github.com/jessevdk/go-flags"
	"github.com/lysyi3m/rss-comb/app/api"
	"github.com/lysyi3m/rss-comb/app/config_loader"
	"github.com/lysyi3m/rss-comb/app/config_sync"
	"github.com/lysyi3m/rss-comb/app/config_watcher"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/logger"
	"github.com/lysyi3m/rss-comb/app/tasks"
	"github.com/lysyi3m/rss-comb/app/version"
)

func main() {
	appConfig := loadConfig()
	if appConfig == nil {
		return
	}

	logger.Initialize(appConfig.Debug)

	// Apply timezone after logger initialization to ensure proper error reporting
	applyTimezoneConfig(appConfig.Timezone)

	slog.Info("Starting RSS Comb server", "version", version.GetVersion())
	db, err := database.NewConnection(
		appConfig.DBHost, appConfig.DBPort, appConfig.DBUser,
		appConfig.DBPassword, appConfig.DBName)
	if err != nil {
		slog.Error("Database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("Database connected successfully")

	// Migrations run automatically unless explicitly disabled for faster startup
	if !appConfig.DisableMigrate {
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


	configLoader := config_loader.NewLoader(appConfig.FeedsDir)
	configs, err := configLoader.LoadAll()
	if err != nil {
		slog.Error("Configuration loading failed", "directory", appConfig.FeedsDir, "error", err)
		os.Exit(1)
	}
	slog.Info("Configuration loaded", "feeds", len(configs), "directory", appConfig.FeedsDir)

	configWatcher, err := config_watcher.NewConfigWatcher(configLoader, appConfig.FeedsDir)
	if err != nil {
		slog.Error("Config watcher initialization failed", "error", err)
		os.Exit(1)
	}

	feedRepo := database.NewFeedRepository(db)
	itemRepo := database.NewItemRepository(db)

	// Sync configuration files with database state
	registeredCount := 0
	urlChangedCount := 0
	for configFile, cfg := range configs {
		_, urlChanged, err := feedRepo.UpsertFeedWithChangeDetection(configFile, cfg.Feed.ID, cfg.Feed.URL, cfg.Feed.Title)
		if err != nil {
			slog.Warn("Feed registration failed", "file", configFile, "error", err)
			continue
		}

		if urlChanged {
			slog.Info("Feed URL updated", "title", cfg.Feed.Title, "id", cfg.Feed.ID, "url", cfg.Feed.URL)
			urlChangedCount++
		} else {
			slog.Debug("Feed registered", "title", cfg.Feed.Title, "id", cfg.Feed.ID, "url", cfg.Feed.URL)
		}
		registeredCount++
	}
	slog.Info("Feed registration completed", "registered", registeredCount, "total", len(configs), "url_changes", urlChangedCount)

	feedProcessor := feed.NewProcessor(feedRepo, itemRepo, appConfig.UserAgent, appConfig.Port)

	// Create config cache handlers for hot-reload
	taskSchedulerConfigCache := config_sync.NewConfigCacheHandler("Task scheduler", configs)
	apiConfigCache := config_sync.NewConfigCacheHandler("API handler", configs)

	slog.Info("Starting task scheduler", "workers", appConfig.WorkerCount, "interval", fmt.Sprintf("%ds", appConfig.SchedulerInterval))
	taskScheduler := tasks.NewTaskScheduler(feedProcessor, feedRepo, taskSchedulerConfigCache,
		time.Duration(appConfig.SchedulerInterval)*time.Second, appConfig.WorkerCount)
	taskScheduler.Start()
	defer taskScheduler.Stop()
	
	apiHandler := api.NewHandler(feedRepo, itemRepo, apiConfigCache, feedProcessor, taskScheduler, appConfig.Port, appConfig.UserAgent)

	// Enable hot-reload by registering handlers for configuration changes
	databaseSyncHandler := config_sync.NewDatabaseSyncHandler(feedRepo, appConfig.FeedsDir)
	configWatcher.AddUpdateHandler(databaseSyncHandler)
	configWatcher.AddUpdateHandler(taskSchedulerConfigCache)
	configWatcher.AddUpdateHandler(apiConfigCache)
	server := api.NewServer(apiHandler, appConfig.APIAccessKey)

	// Production-ready timeouts prevent resource exhaustion
	httpServer := &http.Server{
		Addr:         ":" + appConfig.Port,
		Handler:      server,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start config watcher in a goroutine
	configWatcherCtx, configWatcherCancel := context.WithCancel(context.Background())
	go func() {
		if err := configWatcher.Start(configWatcherCtx); err != nil && err != context.Canceled {
			slog.Error("Config watcher error", "error", err)
		}
	}()
	defer func() {
		configWatcherCancel()
		configWatcher.Stop()
	}()

	// Start HTTP server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	slog.Info("Server started successfully", "port", appConfig.Port, "api_enabled", appConfig.APIAccessKey != "")

	select {
	case sig := <-sigChan:
		slog.Info("Shutdown signal received", "signal", sig)
	case err := <-serverErrChan:
		slog.Error("Server error occurred", "error", err)
	}

	// Graceful shutdown
	slog.Info("Starting graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	} else {
		slog.Debug("HTTP server stopped")
	}

	// Task scheduler is stopped via defer
	slog.Debug("Background task scheduler stopped")

	slog.Info("Server shutdown complete")
}

// AppConfig holds all application configuration with support for environment variables and command-line flags
type AppConfig struct {
	// Database configuration
	DBHost     string `long:"db-host" env:"DB_HOST" default:"localhost" description:"Database host"`
	DBPort     string `long:"db-port" env:"DB_PORT" default:"5432" description:"Database port"`
	DBUser     string `long:"db-user" env:"DB_USER" default:"rss_user" description:"Database user"`
	DBPassword string `long:"db-password" env:"DB_PASSWORD" default:"rss_password" description:"Database password (required)" required:"true"`
	DBName     string `long:"db-name" env:"DB_NAME" default:"rss_comb" description:"Database name"`

	// Application configuration
	FeedsDir          string `long:"feeds-dir" env:"FEEDS_DIR" default:"./feeds" description:"Directory containing feed configuration files"`
	Port              string `long:"port" env:"PORT" default:"8080" description:"HTTP server port"`
	WorkerCount       int    `long:"worker-count" env:"WORKER_COUNT" default:"5" description:"Number of background workers for feed processing"`
	SchedulerInterval int    `long:"scheduler-interval" env:"SCHEDULER_INTERVAL" default:"30" description:"Scheduler interval in seconds"`
	APIAccessKey      string `long:"api-key" env:"API_ACCESS_KEY" description:"API access key for authentication (optional)"`
	DisableMigrate    bool   `long:"disable-migrate" env:"DISABLE_MIGRATE" description:"Disable automatic database migrations on startup"`

	// Application metadata
	UserAgent string `long:"user-agent" env:"USER_AGENT" default:"RSS Comb/1.0" description:"User agent string for HTTP requests"`
	Timezone  string `long:"timezone" env:"TZ" default:"UTC" description:"Timezone for timestamps (e.g., UTC, America/New_York)"`
	Debug     bool   `long:"debug" env:"DEBUG" description:"Enable debug logging"`
}

// loadConfig loads configuration from environment variables and command-line flags
func loadConfig() *AppConfig {
	var appConfig AppConfig

	// Parse command-line arguments and environment variables
	parser := flags.NewParser(&appConfig, flags.Default)

	// Parse arguments (this will also process environment variables)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				// Help was requested, exit gracefully
				return nil
			}
		}
		// Can't use slog here since logger isn't initialized yet
		fmt.Printf("Error: Failed to parse configuration: %v\n", err)
		os.Exit(1)
	}

	return &appConfig
}

// applyTimezoneConfig applies timezone configuration after logger is initialized
func applyTimezoneConfig(timezone string) {
	if timezone != "" {
		if loc, err := time.LoadLocation(timezone); err != nil {
			slog.Warn("Invalid timezone, using system default", "timezone", timezone, "error", err)
		} else {
			time.Local = loc
			slog.Debug("Timezone configured", "timezone", timezone)
		}
	}
}
