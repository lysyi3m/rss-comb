package config_loader

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"github.com/lysyi3m/rss-comb/app/config"
)

// Loader handles loading and validation of feed configurations
type Loader struct {
	feedsDir string
}

// NewLoader creates a new configuration loader
func NewLoader(feedsDir string) *Loader {
	return &Loader{feedsDir: feedsDir}
}

// LoadAll loads all YAML configuration files from the feeds directory
func (l *Loader) LoadAll() (map[string]*config.FeedConfig, error) {
	configs := make(map[string]*config.FeedConfig)
	feedIDs := make(map[string]string) // Track feed ID to file mapping for uniqueness validation

	// Check if feeds directory exists
	if _, err := os.Stat(l.feedsDir); os.IsNotExist(err) {
		return configs, nil // Return empty map if directory doesn't exist
	}

	files, err := filepath.Glob(filepath.Join(l.feedsDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find YML files: %w", err)
	}

	for _, file := range files {
		cfg, err := l.loadFile(file)
		if err != nil {
			return nil, fmt.Errorf("error loading %s: %w", file, err)
		}

		if err := config.ValidateConfig(cfg); err != nil {
			return nil, fmt.Errorf("invalid config %s: %w", file, err)
		}

		// Check for duplicate feed IDs
		if existingFile, exists := feedIDs[cfg.Feed.ID]; exists {
			return nil, fmt.Errorf("duplicate feed ID '%s' found in %s (also in %s)", 
				cfg.Feed.ID, file, existingFile)
		}
		feedIDs[cfg.Feed.ID] = file

		configs[file] = cfg
		log.Printf("Loaded configuration from %s (ID: %s)", file, cfg.Feed.ID)
	}

	return configs, nil
}

// Load loads and validates a single configuration file
func (l *Loader) Load(path string) (*config.FeedConfig, error) {
	cfg, err := l.loadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error loading %s: %w", path, err)
	}

	if err := config.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	return cfg, nil
}

// loadFile loads a single YAML configuration file
func (l *Loader) loadFile(path string) (*config.FeedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var cfg config.FeedConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set defaults
	l.setDefaults(&cfg)

	return &cfg, nil
}

// setDefaults applies default values to configuration
func (l *Loader) setDefaults(cfg *config.FeedConfig) {
	if cfg.Settings.RefreshInterval == 0 {
		cfg.Settings.RefreshInterval = 3600 // seconds
	}
	if cfg.Settings.MaxItems == 0 {
		cfg.Settings.MaxItems = 100
	}
	if cfg.Settings.Timeout == 0 {
		cfg.Settings.Timeout = 30 // seconds
	}
}