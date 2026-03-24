package cfg

import "time"

type Cfg struct {
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
	MediaDir          string `long:"media-dir" env:"MEDIA_DIR" default:"./media" description:"Directory for downloaded media files"`
	YTDLPCmd          string `long:"yt-dlp-cmd" env:"YT_DLP_CMD" default:"yt-dlp" description:"yt-dlp command (supports multi-word for docker, e.g. 'docker compose run --rm yt-dlp')"`

	// Application metadata
	UserAgent string         `long:"user-agent" env:"USER_AGENT" default:"RSS Comb/1.0" description:"User agent string for HTTP requests"`
	Timezone  string         `long:"timezone" env:"TZ" default:"UTC" description:"Timezone for timestamps (e.g., UTC, America/New_York)"`
	Version   string         // Set at runtime from build version
	Location  *time.Location // Parsed timezone location
}
