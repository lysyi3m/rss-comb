package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/lysyi3m/rss-comb/app/api"
	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
	"github.com/lysyi3m/rss-comb/app/parser"
	"github.com/lysyi3m/rss-comb/app/scheduler"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Load configuration from environment variables and command-line flags
	appConfig := loadConfig()
	if appConfig == nil {
		// Help was shown or parsing failed, exit gracefully
		return
	}

	log.Println("Starting RSS Comb server...")

	// Database connection
	log.Println("Connecting to database...")
	db, err := database.NewConnection(
		appConfig.DBHost, appConfig.DBPort, appConfig.DBUser, 
		appConfig.DBPassword, appConfig.DBName)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	log.Printf("Connected to database successfully")


	// Load feed configurations
	log.Printf("Loading feed configurations from %s...", appConfig.FeedsDir)
	loader := config.NewLoader(appConfig.FeedsDir)
	configs, err := loader.LoadAll()
	if err != nil {
		log.Fatal("Failed to load configurations:", err)
	}
	log.Printf("Loaded %d feed configurations", len(configs))

	// Initialize repositories
	feedRepo := database.NewFeedRepository(db)
	itemRepo := database.NewItemRepository(db)

	// Register feeds in database
	log.Println("Registering feeds in database...")
	registeredCount := 0
	urlChangedCount := 0
	for configFile, cfg := range configs {
		dbID, urlChanged, err := feedRepo.UpsertFeedWithChangeDetection(configFile, cfg.Feed.ID, cfg.Feed.URL, cfg.Feed.Name)
		if err != nil {
			log.Printf("Warning: Failed to register feed %s: %v", configFile, err)
			continue
		}
		
		if urlChanged {
			log.Printf("Feed URL updated: %s (ID: %s, DB ID: %s, New URL: %s)", cfg.Feed.Name, cfg.Feed.ID, dbID, cfg.Feed.URL)
			urlChangedCount++
		} else {
			log.Printf("Registered feed: %s (ID: %s, DB ID: %s, URL: %s)", cfg.Feed.Name, cfg.Feed.ID, dbID, cfg.Feed.URL)
		}
		registeredCount++
	}
	log.Printf("Successfully registered %d/%d feeds", registeredCount, len(configs))
	if urlChangedCount > 0 {
		log.Printf("Updated URLs for %d feeds", urlChangedCount)
	}

	// Initialize core components
	feedParser := parser.NewParser()
	feedProcessor := feed.NewProcessor(feedParser, feedRepo, itemRepo, configs)

	// Initialize and start scheduler
	log.Printf("Starting background scheduler with %d workers...", appConfig.WorkerCount)
	feedScheduler := scheduler.NewScheduler(feedProcessor, feedRepo, 
		time.Duration(appConfig.SchedulerInterval)*time.Second, appConfig.WorkerCount)
	feedScheduler.Start()
	defer feedScheduler.Stop()

	// Initialize HTTP server
	log.Println("Initializing HTTP server...")
	apiHandler := api.NewHandler(feedRepo, itemRepo, configs, feedProcessor, appConfig.Port)
	server := api.NewServer(apiHandler, appConfig.APIAccessKey)

	// Create HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         ":" + appConfig.Port,
		Handler:      server,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start HTTP server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		log.Printf("Starting HTTP server on port %s", appConfig.Port)
		log.Printf("API endpoints available:")
		log.Printf("  Feed:          http://localhost:%s/feeds/<id>", appConfig.Port)
		log.Printf("  Health check:  http://localhost:%s/health", appConfig.Port)
		log.Printf("  Statistics:    http://localhost:%s/stats", appConfig.Port)
		
		if appConfig.APIAccessKey != "" {
			log.Printf("  List feeds:    http://localhost:%s/api/feeds (requires API key)", appConfig.Port)
			log.Printf("  Feed details:  http://localhost:%s/api/feeds/<id>/details (requires API key)", appConfig.Port)
			log.Printf("  Refilter:      http://localhost:%s/api/feeds/<id>/refilter (POST, requires API key)", appConfig.Port)
		} else {
			log.Printf("  API endpoints: DISABLED (API_ACCESS_KEY not set)")
		}
		
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("RSS Comb server started successfully!")
	log.Println("Press Ctrl+C to shutdown gracefully...")

	select {
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	case err := <-serverErrChan:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown
	log.Println("Shutting down server gracefully...")
	
	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	} else {
		log.Println("HTTP server stopped")
	}

	// Scheduler is stopped via defer
	log.Println("Background scheduler stopped")

	log.Println("RSS Comb server shutdown complete")
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

	// Application metadata
	UserAgent string `long:"user-agent" env:"USER_AGENT" default:"RSS Comb/1.0" description:"User agent string for HTTP requests"`
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
		log.Fatalf("Failed to parse configuration: %v", err)
	}

	// Additional validation can be added here if needed
	log.Printf("Configuration loaded successfully")
	
	return &appConfig
}
