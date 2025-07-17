package feed_config

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Loader struct {
	feedsDir string
}

func NewLoader(feedsDir string) *Loader {
	return &Loader{feedsDir: feedsDir}
}

func (l *Loader) LoadAll() (map[string]*FeedConfig, error) {
	configs := make(map[string]*FeedConfig)
	feedIDs := make(map[string]string) // Enforce unique feed IDs across all configuration files

	if _, err := os.Stat(l.feedsDir); os.IsNotExist(err) {
		return configs, nil // Graceful handling when feeds directory is missing
	}

	files, err := filepath.Glob(filepath.Join(l.feedsDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find YML files: %w", err)
	}

	for _, file := range files {
		feedConfig, err := l.loadFile(file)
		if err != nil {
			return nil, fmt.Errorf("error loading %s: %w", file, err)
		}

		if err := ValidateConfig(feedConfig); err != nil {
			return nil, fmt.Errorf("invalid config %s: %w", file, err)
		}

		// Prevent routing conflicts by ensuring feed ID uniqueness
		if existingFile, exists := feedIDs[feedConfig.Feed.ID]; exists {
			return nil, fmt.Errorf("duplicate feed ID '%s' found in %s (also in %s)", 
				feedConfig.Feed.ID, file, existingFile)
		}
		feedIDs[feedConfig.Feed.ID] = file

		configs[file] = feedConfig
		slog.Debug("Configuration loaded", "file", file, "feed_id", feedConfig.Feed.ID)
	}

	return configs, nil
}

func (l *Loader) Load(path string) (*FeedConfig, error) {
	feedConfig, err := l.loadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error loading %s: %w", path, err)
	}

	if err := ValidateConfig(feedConfig); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}

	return feedConfig, nil
}

func (l *Loader) loadFile(path string) (*FeedConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var feedConfig FeedConfig
	if err := yaml.Unmarshal(data, &feedConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	l.setDefaults(&feedConfig)

	return &feedConfig, nil
}

func (l *Loader) setDefaults(feedConfig *FeedConfig) {
	if feedConfig.Settings.RefreshInterval == 0 {
		feedConfig.Settings.RefreshInterval = 3600 // seconds
	}
	if feedConfig.Settings.MaxItems == 0 {
		feedConfig.Settings.MaxItems = 100
	}
	if feedConfig.Settings.Timeout == 0 {
		feedConfig.Settings.Timeout = 30 // seconds
	}
}
