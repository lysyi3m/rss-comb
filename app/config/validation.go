package config

import "fmt"

// ValidateConfig performs comprehensive validation on a feed configuration.
// This function validates all aspects of the configuration including feed info,
// settings, and filter definitions to ensure the configuration is complete and valid.
func ValidateConfig(config *FeedConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Validate feed information
	if config.Feed.ID == "" {
		return fmt.Errorf("feed ID is required")
	}
	if config.Feed.URL == "" {
		return fmt.Errorf("feed URL is required")
	}
	if config.Feed.Title == "" {
		return fmt.Errorf("feed title is required")
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

