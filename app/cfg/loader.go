package cfg

import (
	"cmp"
	"fmt"
	"time"

	"github.com/jessevdk/go-flags"
)

// Version is set at build time via -ldflags
var Version = "dev"

func GetVersion() string {
	return cmp.Or(Version, "unknown")
}

type rawCfg struct {
	// Database configuration
	DBHost     string `long:"db-host" env:"DB_HOST" default:"localhost" description:"Database host"`
	DBPort     string `long:"db-port" env:"DB_PORT" default:"5432" description:"Database port"`
	DBUser     string `long:"db-user" env:"DB_USER" default:"rss_user" description:"Database user"`
	DBPassword string `long:"db-password" env:"DB_PASSWORD" default:"rss_password" description:"Database password (required)" required:"true"`
	DBName     string `long:"db-name" env:"DB_NAME" default:"rss_comb" description:"Database name"`

	// Application configuration
	FeedsDir          string `long:"feeds-dir" env:"FEEDS_DIR" default:"./feeds" description:"Directory containing feed configuration files"`
	Port              string `long:"port" env:"PORT" default:"8080" description:"HTTP server port"`
	BaseUrl           string `long:"base-url" env:"BASE_URL" description:"Public base URL for the service (e.g., https://feeds.example.com)"`
	WorkerCount       int    `long:"worker-count" env:"WORKER_COUNT" default:"5" description:"Number of background workers for feed processing"`
	SchedulerInterval int    `long:"scheduler-interval" env:"SCHEDULER_INTERVAL" default:"30" description:"Scheduler interval in seconds"`
	APIAccessKey      string `long:"api-key" env:"API_ACCESS_KEY" description:"API access key for authentication (optional)"`

	// Application metadata
	UserAgent string `long:"user-agent" env:"USER_AGENT" default:"RSS Comb/1.0" description:"User agent string for HTTP requests"`
	Timezone  string `long:"timezone" env:"TZ" default:"UTC" description:"Timezone for timestamps (e.g., UTC, America/New_York)"`
	Debug     bool   `long:"debug" env:"DEBUG" description:"Enable debug logging"`
}

var globalCfg *Cfg

func Load() (*Cfg, error) {
	var raw rawCfg

	parser := flags.NewParser(&raw, flags.Default)

	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok {
			if flagsErr.Type == flags.ErrHelp {
				return nil, nil
			}
		}
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	cfg := &Cfg{
		DBHost:            raw.DBHost,
		DBPort:            raw.DBPort,
		DBUser:            raw.DBUser,
		DBPassword:        raw.DBPassword,
		DBName:            raw.DBName,
		FeedsDir:          raw.FeedsDir,
		Port:              raw.Port,
		BaseUrl:           raw.BaseUrl,
		WorkerCount:       raw.WorkerCount,
		SchedulerInterval: raw.SchedulerInterval,
		APIAccessKey:      raw.APIAccessKey,
		UserAgent:         raw.UserAgent,
		Timezone:          raw.Timezone,
		Debug:             raw.Debug,
		Version:           GetVersion(),
	}

	if err := applyTimezone(cfg.Timezone); err != nil {
		fmt.Printf("Warning: Invalid timezone '%s', using system default: %v\n", cfg.Timezone, err)
	}

	globalCfg = cfg

	return cfg, nil
}

func Get() *Cfg {
	if globalCfg == nil {
		panic("configuration not loaded - call cfg.Load() first")
	}
	return globalCfg
}

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
