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

	// Check if feeds directory exists
	if _, err := os.Stat(l.feedsDir); os.IsNotExist(err) {
		return configs, nil // Return empty map if directory doesn't exist
	}

	files, err := filepath.Glob(filepath.Join(l.feedsDir, "*.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find YAML files: %w", err)
	}

	// Also check for .yml extension
	ymlFiles, err := filepath.Glob(filepath.Join(l.feedsDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find YML files: %w", err)
	}
	files = append(files, ymlFiles...)

	for _, file := range files {
		config, err := l.loadFile(file)
		if err != nil {
			return nil, fmt.Errorf("error loading %s: %w", file, err)
		}

		if err := l.validate(config); err != nil {
			return nil, fmt.Errorf("invalid config %s: %w", file, err)
		}

		configs[file] = config
		log.Printf("Loaded configuration from %s", file)
	}

	return configs, nil
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

// validate validates the configuration
func (l *Loader) validate(config *FeedConfig) error {
	// Validate feed information
	if config.Feed.URL == "" {
		return fmt.Errorf("feed URL is required")
	}
	if config.Feed.Name == "" {
		return fmt.Errorf("feed name is required")
	}

	// Validate settings
	if config.Settings.RefreshInterval < 0 {
		return fmt.Errorf("refresh interval must be non-negative")
	}
	if config.Settings.MaxItems < 0 {
		return fmt.Errorf("max items must be non-negative")
	}
	if config.Settings.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}

	// Validate filter fields
	validFields := map[string]bool{
		"title":       true,
		"description": true,
		"content":     true,
		"author":      true,
		"link":        true,
		"categories":  true,
	}

	for i, filter := range config.Filters {
		if !validFields[filter.Field] {
			return fmt.Errorf("invalid filter field at index %d: %s", i, filter.Field)
		}
		if len(filter.Includes) == 0 && len(filter.Excludes) == 0 {
			return fmt.Errorf("filter at index %d must have at least one include or exclude rule", i)
		}
	}

	return nil
}