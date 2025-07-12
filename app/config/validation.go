package config

import "fmt"

// ValidateConfig performs comprehensive validation on a feed configuration.
// This function validates all aspects of the configuration including feed info,
// settings, and filter definitions to ensure the configuration is complete and valid.
func ValidateConfig(config *FeedConfig) error {
	if config == nil {
		return fmt.Errorf("config is nil")
	}

	// Validate feed information - check required string fields
	requiredFeedFields := map[string]string{
		"feed ID":    config.Feed.ID,
		"feed URL":   config.Feed.URL,
		"feed title": config.Feed.Title,
	}
	
	for fieldName, fieldValue := range requiredFeedFields {
		if fieldValue == "" {
			return fmt.Errorf("%s is required", fieldName)
		}
	}

	// Validate settings - check non-negative integer fields
	nonNegativeFields := map[string]int{
		"refresh interval": config.Settings.RefreshInterval,
		"max items":        config.Settings.MaxItems,
		"timeout":          config.Settings.Timeout,
	}
	
	for fieldName, fieldValue := range nonNegativeFields {
		if fieldValue < 0 {
			return fmt.Errorf("%s must be non-negative", fieldName)
		}
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

