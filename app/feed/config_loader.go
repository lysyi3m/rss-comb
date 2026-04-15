package feed

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func LoadConfig(feedsDir, name string) (*Config, string, error) {
	configPath := filepath.Join(feedsDir, name+".yml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read config file: %w", err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(data))

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	config.Name = name

	if err := validateConfig(&config); err != nil {
		return nil, "", fmt.Errorf("invalid config: %w", err)
	}

	applyDefaults(&config)

	return &config, hash, nil
}

func validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.URL == "" {
		return fmt.Errorf("url is required")
	}

	if config.Settings.RefreshInterval < 0 {
		return fmt.Errorf("refresh_interval must be >= 0")
	}

	if config.Settings.MaxItems < 0 {
		return fmt.Errorf("max_items must be >= 0")
	}

	if config.Settings.Timeout < 0 {
		return fmt.Errorf("timeout must be >= 0")
	}

	validTypes := map[string]bool{"": true, "podcast": true, "youtube": true}
	if !validTypes[config.Type] {
		return fmt.Errorf("invalid type %q (must be one of: podcast, youtube, or omitted)", config.Type)
	}

	if config.Settings.ExtractContent && config.Type != "" {
		return fmt.Errorf("extract_content is only supported for basic (no type) feeds")
	}

	if config.Settings.MinDuration < 0 {
		return fmt.Errorf("min_duration must be >= 0")
	}

	if config.Settings.MinDuration > 0 && config.Type != "youtube" {
		return fmt.Errorf("min_duration is only supported for youtube feeds")
	}

	for i, filter := range config.Filters {
		if filter.Field == "" {
			return fmt.Errorf("filter %d: field is required", i)
		}

		validFields := map[string]bool{
			"title":       true,
			"description": true,
			"content":     true,
			"link":        true,
			"authors":     true,
			"categories":  true,
		}

		if !validFields[filter.Field] {
			return fmt.Errorf("filter %d: invalid field '%s' (must be one of: title, description, content, link, authors, categories)", i, filter.Field)
		}
	}

	return nil
}

func applyDefaults(config *Config) {
	if config.Settings.RefreshInterval == 0 {
		config.Settings.RefreshInterval = 1800 // 30 minutes
	}

	if config.Settings.MaxItems == 0 {
		config.Settings.MaxItems = 50
	}

	if config.Settings.Timeout == 0 {
		config.Settings.Timeout = 30 // seconds
	}
}
