package feed

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type ConfigCache struct {
	feedsDir string
	cache    map[string]*Config
	mu       sync.RWMutex
}

func NewConfigCache(feedsDir string) *ConfigCache {
	return &ConfigCache{
		feedsDir: feedsDir,
		cache:    make(map[string]*Config),
	}
}

func (cc *ConfigCache) Run() error {
	if _, err := os.Stat(cc.feedsDir); os.IsNotExist(err) {
		return nil
	}

	files, err := filepath.Glob(filepath.Join(cc.feedsDir, "*.yml"))
	if err != nil {
		return fmt.Errorf("failed to find YML files: %w", err)
	}

	for _, file := range files {
		// Derive feed name from filename (remove .yml extension)
		fileName := filepath.Base(file)
		feedName := fileName[:len(fileName)-4] // Remove .yml extension

		config, err := cc.LoadConfig(feedName)
		if err != nil {
			return fmt.Errorf("error loading %s: %w", file, err)
		}

		slog.Debug("Configuration loaded", "feed", feedName, "enabled", config.Settings.Enabled, "refresh_interval", config.Settings.RefreshInterval)
	}

	return nil
}

func (cc *ConfigCache) LoadConfig(feedName string) (*Config, error) {
	configFile := cc.getConfigFilePath(feedName)
	feedConfig, err := cc.parseConfig(configFile)
	if err != nil {
		return nil, err
	}

	// Set feed name from parameter
	feedConfig.Name = feedName

	if err := cc.validateConfig(feedConfig); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", configFile, err)
	}

	// Store in cache
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cc.cache[feedConfig.Name] = feedConfig

	return feedConfig, nil
}

func (cc *ConfigCache) GetConfig(feedName string) (*Config, error) {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	feedConfig, ok := cc.cache[feedName]
	if !ok {
		return nil, fmt.Errorf("feed config with name '%s' not found", feedName)
	}
	return feedConfig, nil
}

func (cc *ConfigCache) GetConfigs() map[string]*Config {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	configsCopy := make(map[string]*Config, len(cc.cache))
	for k, v := range cc.cache {
		configsCopy[k] = v
	}
	return configsCopy
}

func (cc *ConfigCache) GetEnabledConfigs() map[string]*Config {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	enabledConfigs := make(map[string]*Config)
	for k, v := range cc.cache {
		if v.Settings.Enabled {
			enabledConfigs[k] = v
		}
	}
	return enabledConfigs
}

func (cc *ConfigCache) GetConfigCount() int {
	cc.mu.RLock()
	defer cc.mu.RUnlock()
	return len(cc.cache)
}

func (cc *ConfigCache) parseConfig(configFile string) (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var feedConfig Config
	if err := yaml.Unmarshal(data, &feedConfig); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if feedConfig.Settings.RefreshInterval == 0 {
		feedConfig.Settings.RefreshInterval = 3600
	}
	if feedConfig.Settings.MaxItems == 0 {
		feedConfig.Settings.MaxItems = 100
	}
	if feedConfig.Settings.Timeout == 0 {
		feedConfig.Settings.Timeout = 30
	}

	return &feedConfig, nil
}

func (cc *ConfigCache) validateConfig(feedConfig *Config) error {
	if feedConfig == nil {
		return fmt.Errorf("feedConfig is nil")
	}

	requiredFeedFields := map[string]string{
		"feed name": feedConfig.Name,
		"feed URL":  feedConfig.URL,
	}

	for fieldName, fieldValue := range requiredFeedFields {
		if fieldValue == "" {
			return fmt.Errorf("%s is required", fieldName)
		}
	}

	nonNegativeFields := map[string]int{
		"refresh interval": feedConfig.Settings.RefreshInterval,
		"max items":        feedConfig.Settings.MaxItems,
		"timeout":          feedConfig.Settings.Timeout,
	}

	for fieldName, fieldValue := range nonNegativeFields {
		if fieldValue < 0 {
			return fmt.Errorf("%s must be non-negative", fieldName)
		}
	}

	validFields := map[string]bool{
		"title":       true,
		"description": true,
		"content":     true,
		"authors":     true,
		"link":        true,
		"categories":  true,
	}

	for i, filter := range feedConfig.Filters {
		if !validFields[filter.Field] {
			return fmt.Errorf("invalid filter field at index %d: %s", i, filter.Field)
		}
		if len(filter.Includes) == 0 && len(filter.Excludes) == 0 {
			return fmt.Errorf("filter at index %d must have at least one include or exclude rule", i)
		}
	}

	return nil
}

func (cc *ConfigCache) getConfigFilePath(feedName string) string {
	return filepath.Join(cc.feedsDir, feedName+".yml")
}
