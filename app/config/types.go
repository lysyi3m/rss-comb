package config

// Config represents the application configuration
type Config struct {
	// Database configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Application configuration
	FeedsDir          string
	Port              string
	WorkerCount       int
	SchedulerInterval int
	APIAccessKey      string
	DisableMigrate    bool

	// Application metadata
	UserAgent string
	Timezone  string
	Debug     bool
	Version   string
}

// Interface provides access to application configuration
type Interface interface {
	GetPort() string
	GetUserAgent() string
	GetWorkerCount() int
	GetSchedulerInterval() int
	GetAPIAccessKey() string
	GetVersion() string
	GetFeedsDir() string
	GetDBHost() string
	GetDBPort() string
	GetDBUser() string
	GetDBPassword() string
	GetDBName() string
	GetTimezone() string
	IsDebugEnabled() bool
	IsMigrationDisabled() bool
}

// Getter methods for Config
func (c *Config) GetPort() string { return c.Port }
func (c *Config) GetUserAgent() string { return c.UserAgent }
func (c *Config) GetWorkerCount() int { return c.WorkerCount }
func (c *Config) GetSchedulerInterval() int { return c.SchedulerInterval }
func (c *Config) GetAPIAccessKey() string { return c.APIAccessKey }
func (c *Config) GetVersion() string { return c.Version }
func (c *Config) GetFeedsDir() string { return c.FeedsDir }
func (c *Config) GetDBHost() string { return c.DBHost }
func (c *Config) GetDBPort() string { return c.DBPort }
func (c *Config) GetDBUser() string { return c.DBUser }
func (c *Config) GetDBPassword() string { return c.DBPassword }
func (c *Config) GetDBName() string { return c.DBName }
func (c *Config) GetTimezone() string { return c.Timezone }
func (c *Config) IsDebugEnabled() bool { return c.Debug }
func (c *Config) IsMigrationDisabled() bool { return c.DisableMigrate }