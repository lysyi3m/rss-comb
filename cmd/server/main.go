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

	"github.com/lysyi3m/rss-comb/internal/api"
	"github.com/lysyi3m/rss-comb/internal/config"
	"github.com/lysyi3m/rss-comb/internal/database"
	"github.com/lysyi3m/rss-comb/internal/feed"
	"github.com/lysyi3m/rss-comb/internal/parser"
	"github.com/lysyi3m/rss-comb/internal/scheduler"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting RSS Comb server...")

	// Load environment configuration
	envConfig := loadEnvironmentConfig()
	log.Printf("Environment configuration loaded")

	// Database connection
	log.Println("Connecting to database...")
	db, err := database.NewConnection(
		envConfig.DBHost, envConfig.DBPort, envConfig.DBUser, 
		envConfig.DBPassword, envConfig.DBName)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	log.Printf("Connected to database successfully")


	// Load feed configurations
	log.Printf("Loading feed configurations from %s...", envConfig.FeedsDir)
	loader := config.NewLoader(envConfig.FeedsDir)
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
	for configFile, cfg := range configs {
		feedID, err := feedRepo.UpsertFeed(configFile, cfg.Feed.URL, cfg.Feed.Name)
		if err != nil {
			log.Printf("Warning: Failed to register feed %s: %v", configFile, err)
			continue
		}
		log.Printf("Registered feed: %s (ID: %s, URL: %s)", cfg.Feed.Name, feedID, cfg.Feed.URL)
		registeredCount++
	}
	log.Printf("Successfully registered %d/%d feeds", registeredCount, len(configs))

	// Initialize core components
	feedParser := parser.NewParser()
	feedProcessor := feed.NewProcessor(feedParser, feedRepo, itemRepo, configs)

	// Initialize and start scheduler
	log.Printf("Starting background scheduler with %d workers...", envConfig.WorkerCount)
	feedScheduler := scheduler.NewScheduler(feedProcessor, feedRepo, 
		time.Duration(envConfig.SchedulerInterval)*time.Second, envConfig.WorkerCount)
	feedScheduler.Start()
	defer feedScheduler.Stop()

	// Initialize HTTP server
	log.Println("Initializing HTTP server...")
	apiHandler := api.NewHandler(feedRepo, itemRepo, configs, feedProcessor)
	server := api.NewServer(apiHandler)

	// Create HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         ":" + envConfig.Port,
		Handler:      server,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start HTTP server in a goroutine
	serverErrChan := make(chan error, 1)
	go func() {
		log.Printf("Starting HTTP server on port %s", envConfig.Port)
		log.Printf("API endpoints available:")
		log.Printf("  Main feed:     http://localhost:%s/feed?url=<feed-url>", envConfig.Port)
		log.Printf("  Health check:  http://localhost:%s/health", envConfig.Port)
		log.Printf("  Statistics:    http://localhost:%s/stats", envConfig.Port)
		log.Printf("  List feeds:    http://localhost:%s/api/v1/feeds", envConfig.Port)
		log.Printf("  Feed details:  http://localhost:%s/api/v1/feeds/details?url=<feed-url>", envConfig.Port)
		log.Printf("  Reapply filters: http://localhost:%s/api/v1/feeds/reapply-filters?url=<feed-url> (POST)", envConfig.Port)
		
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

// EnvironmentConfig holds all environment configuration
type EnvironmentConfig struct {
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	FeedsDir          string
	Port              string
	WorkerCount       int
	SchedulerInterval int // seconds
}

// loadEnvironmentConfig loads configuration from environment variables
func loadEnvironmentConfig() *EnvironmentConfig {
	config := &EnvironmentConfig{
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "rss_user"),
		DBPassword:        getEnv("DB_PASSWORD", "rss_password"),
		DBName:            getEnv("DB_NAME", "rss_comb"),
		FeedsDir:          getEnv("FEEDS_DIR", "./feeds"),
		Port:              getEnv("PORT", "8080"),
		WorkerCount:       getEnvInt("WORKER_COUNT", 5),
		SchedulerInterval: getEnvInt("SCHEDULER_INTERVAL", 30),
	}

	// Validate critical configuration
	if config.DBPassword == "" {
		log.Fatal("DB_PASSWORD environment variable is required")
	}

	return config
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets environment variable as integer with default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := parseInt(value); err == nil {
			return intValue
		}
		log.Printf("Warning: Invalid integer value for %s: %s, using default: %d", key, value, defaultValue)
	}
	return defaultValue
}

// parseInt parses string to int (simple implementation)
func parseInt(s string) (int, error) {
	result := 0
	for _, char := range s {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		result = result*10 + int(char-'0')
	}
	return result, nil
}