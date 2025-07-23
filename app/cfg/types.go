package cfg

type Cfg struct {
	// Database configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Application configuration
	FeedsDir          string
	Port              string
	BaseUrl           string
	WorkerCount       int
	SchedulerInterval int
	APIAccessKey      string

	// Application metadata
	UserAgent string
	Timezone  string
	Debug     bool
	Version   string
}
