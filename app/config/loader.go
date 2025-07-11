package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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
func (l *Loader) LoadAll() (map[string]*FeedConfig, error) {
	configs := make(map[string]*FeedConfig)
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
		config, err := l.loadFile(file)
		if err != nil {
			return nil, fmt.Errorf("error loading %s: %w", file, err)
		}

		if err := ValidateConfig(config); err != nil {
			return nil, fmt.Errorf("invalid config %s: %w", file, err)
		}

		// Check for duplicate feed IDs
		if existingFile, exists := feedIDs[config.Feed.ID]; exists {
			return nil, fmt.Errorf("duplicate feed ID '%s' found in %s (also in %s)", 
				config.Feed.ID, file, existingFile)
		}
		feedIDs[config.Feed.ID] = file

		configs[file] = config
		log.Printf("Loaded configuration from %s (ID: %s)", file, config.Feed.ID)
	}

	return configs, nil
}

// Load loads and validates a single configuration file
func (l *Loader) Load(path string) (*FeedConfig, error) {
	config, err := l.loadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error loading %s: %w", path, err)
	}

	if err := ValidateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	return config, nil
}

// loadFile loads a single YAML configuration file
func (l *Loader) loadFile(path string) (*FeedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config FeedConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set defaults
	l.setDefaults(&config)

	return &config, nil
}

// setDefaults applies default values to configuration
func (l *Loader) setDefaults(config *FeedConfig) {
	if config.Settings.RefreshInterval == 0 {
		config.Settings.RefreshInterval = 3600 // seconds
	}
	if config.Settings.MaxItems == 0 {
		config.Settings.MaxItems = 100
	}
	if config.Settings.Timeout == 0 {
		config.Settings.Timeout = 30 // seconds
	}
}

