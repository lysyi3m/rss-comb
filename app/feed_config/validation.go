package feed_config

import "fmt"

// ValidateConfig performs comprehensive validation on a feed feedConfig.
// This function validates all aspects of the feedConfig including feed info,
// settings, and filter definitions to ensure the feedConfig is complete and valid.
func ValidateConfig(feedConfig *FeedConfig) error {
	if feedConfig == nil {
		return fmt.Errorf("feedConfig is nil")
	}

	// Validate feed information - check required string fields
	requiredFeedFields := map[string]string{
		"feed ID":    feedConfig.Feed.ID,
		"feed URL":   feedConfig.Feed.URL,
		"feed title": feedConfig.Feed.Title,
	}
	
	for fieldName, fieldValue := range requiredFeedFields {
		if fieldValue == "" {
			return fmt.Errorf("%s is required", fieldName)
		}
	}

	// Validate settings - check non-negative integer fields
	nonNegativeFields := map[string]int{
		"refresh interval":    feedConfig.Settings.RefreshInterval,
		"max items":           feedConfig.Settings.MaxItems,
		"timeout":             feedConfig.Settings.Timeout,
		"extraction timeout":  feedConfig.Settings.ExtractionTimeout,
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

