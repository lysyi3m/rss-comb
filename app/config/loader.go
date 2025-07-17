package config

import (
	"fmt"
	"time"

	"github.com/jessevdk/go-flags"
)

// Version is the current version of the application
// This variable is set at build time via -ldflags
var Version = "dev"

// GetVersion returns the current version of the application
func GetVersion() string {
	if Version == "" {
		return "unknown"
	}
	return Version
}

// rawConfig is used for parsing command-line flags and environment variables
type rawConfig struct {
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

// global instance of config
var globalConfig *Config

// Load loads configuration from environment variables and command-line flags
func Load() (Interface, error) {
	var raw rawConfig

	parser := flags.NewParser(&raw, flags.Default)

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				// Help was requested, return nil without error
				return nil, nil
			}
		}
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Create the config instance
	config := &Config{
		DBHost:            raw.DBHost,
		DBPort:            raw.DBPort,
		DBUser:            raw.DBUser,
		DBPassword:        raw.DBPassword,
		DBName:            raw.DBName,
		FeedsDir:          raw.FeedsDir,
		Port:              raw.Port,
		WorkerCount:       raw.WorkerCount,
		SchedulerInterval: raw.SchedulerInterval,
		APIAccessKey:      raw.APIAccessKey,
		DisableMigrate:    raw.DisableMigrate,
		UserAgent:         raw.UserAgent,
		Timezone:          raw.Timezone,
		Debug:             raw.Debug,
		Version:           GetVersion(),
	}

	// Apply timezone configuration
	if err := applyTimezone(config.Timezone); err != nil {
		// Use fmt.Printf as requested, not slog
		fmt.Printf("Warning: Invalid timezone '%s', using system default: %v\n", config.Timezone, err)
	}

	// Store globally for Get() function
	globalConfig = config

	return config, nil
}

// Get returns the global configuration instance
// This function provides access to the configuration from anywhere in the app
func Get() Interface {
	if globalConfig == nil {
		// This should not happen in normal operation
		panic("configuration not loaded - call config.Load() first")
	}
	return globalConfig
}

// applyTimezone applies timezone configuration
func applyTimezone(timezone string) error {
	if timezone != "" {
		if loc, err := time.LoadLocation(timezone); err != nil {
			return err
		} else {
			time.Local = loc
			fmt.Printf("Timezone configured: %s\n", timezone)
		}
	}
	return nil
}