package feed

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/database"
)

// Compile-time interface compliance checks
var _ ProcessorInterface = (*Processor)(nil)


// NewProcessor creates a new feed processor
func NewProcessor(fr database.FeedRepositoryInterface, ir database.ItemRepositoryInterface, 
	userAgent string, port string) *Processor {
	return &Processor{
		parser:      NewParser(),
		generator:   NewGenerator(port),
		feedRepo:    fr,
		itemRepo:    ir,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				DisableKeepAlives:   false,
				MaxIdleConnsPerHost: 5,
			},
		},
		userAgent: userAgent,
	}
}

// ProcessFeed processes a single feed
func (p *Processor) ProcessFeed(feedID string, feedConfig *config.FeedConfig) error {

	if !feedConfig.Settings.Enabled {
		slog.Debug("Feed disabled, skipping", "title", feedConfig.Feed.Title)
		return nil
	}

	slog.Debug("Processing feed", "title", feedConfig.Feed.Title, "url", feedConfig.Feed.URL)
	startTime := time.Now()

	// Fetch feed data
	data, err := p.fetchFeed(feedConfig.Feed.URL, feedConfig.Settings)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	// Parse feed
	metadata, items, err := p.parser.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse feed: %w", err)
	}

	// Update feed metadata
	if err := p.feedRepo.UpdateFeedMetadata(feedID, metadata.Link, metadata.ImageURL, metadata.Language); err != nil {
		return fmt.Errorf("failed to update feed metadata: %w", err)
	}

	// Process items
	processedCount, skippedCount, filteredCount := 0, 0, 0
	for i, item := range items {
		// Stop if we've reached the max items limit
		if processedCount >= feedConfig.Settings.MaxItems {
			slog.Debug("Max items limit reached", "limit", feedConfig.Settings.MaxItems, "title", feedConfig.Feed.Title)
			break
		}

		// Check for duplicates BEFORE storing (skip duplicates entirely)
		if feedConfig.Settings.Deduplication {
			isDup, _, err := p.itemRepo.CheckDuplicate(item.ContentHash, feedID)
			if err != nil {
				slog.Warn("Failed to check duplicate", "item_index", i, "error", err)
			} else if isDup {
				skippedCount++
				// Skip storing duplicates
				continue
			}
		}

		// Apply filters
		filtered, reason := p.applyFilters(item, feedConfig.Filters)
		if filtered {
			item.IsFiltered = true
			item.FilterReason = reason
			filteredCount++
			slog.Debug("Item filtered", "item_index", i, "reason", reason)
		}

		// Store item (duplicates already skipped)
		dbItem := p.convertToDBItem(item)
		if err := p.itemRepo.StoreItem(feedID, dbItem); err != nil {
			slog.Warn("Failed to store item", "item_index", i, "error", err)
			continue
		}

		processedCount++
	}

	// Update next fetch time (use UTC for database consistency)
	nextFetch := time.Now().UTC().Add(feedConfig.Settings.GetRefreshInterval())
	if err := p.feedRepo.UpdateNextFetch(feedID, nextFetch); err != nil {
		return fmt.Errorf("failed to update next fetch time: %w", err)
	}

	duration := time.Since(startTime)
	newItems := processedCount - filteredCount
	slog.Info("Feed processed",
		"title", feedConfig.Feed.Title,
		"total", len(items),
		"new", newItems,
		"duplicates", skippedCount,
		"filtered", filteredCount,
		"duration", duration.String())

	return nil
}

// fetchFeed fetches feed data from the given URL
func (p *Processor) fetchFeed(url string, settings config.FeedSettings) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/atom+xml, application/xml, text/xml")

	// Update client timeout if specified
	if settings.GetTimeout() > 0 {
		p.client.Timeout = settings.GetTimeout()
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "xml") && !strings.Contains(contentType, "rss") {
		slog.Warn("Unexpected content type", "content_type", contentType)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	return data, nil
}

// applyFilters applies configured filters to an item
func (p *Processor) applyFilters(item Item, filters []config.Filter) (bool, string) {
	for _, filter := range filters {
		value := p.getFieldValue(item, filter.Field)

		// Check excludes first (if any exclude matches, item is filtered)
		for _, exclude := range filter.Excludes {
			if p.matchesFilter(value, exclude) {
				return true, fmt.Sprintf("Excluded by %s filter: contains '%s'", filter.Field, exclude)
			}
		}

		// If includes are specified, at least one must match
		if len(filter.Includes) > 0 {
			matched := false
			for _, include := range filter.Includes {
				if p.matchesFilter(value, include) {
					matched = true
					break
				}
			}
			if !matched {
				return true, fmt.Sprintf("Excluded by %s filter: does not contain any of %v", filter.Field, filter.Includes)
			}
		}
	}

	return false, ""
}

// matchesFilter performs case-insensitive substring matching
func (p *Processor) matchesFilter(value, pattern string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(pattern))
}

// getFieldValue extracts the value of a specific field from an item
func (p *Processor) getFieldValue(item Item, field string) string {
	switch field {
	case "title":
		return item.Title
	case "description":
		return item.Description
	case "content":
		return item.Content
	case "author":
		return item.AuthorName
	case "link":
		return item.Link
	case "categories":
		return strings.Join(item.Categories, " ")
	default:
		return ""
	}
}


// IsFeedEnabled checks if a feed is enabled in its configuration
func (p *Processor) IsFeedEnabled(feedConfig *config.FeedConfig) bool {
	if feedConfig == nil {
		return false // Configuration not found, treat as disabled
	}
	return feedConfig.Settings.Enabled
}

// GetStats returns processing statistics
func (p *Processor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"client_timeout": p.client.Timeout.String(),
	}
}



// ReapplyFilters re-applies filters to all items of a specific feed
func (p *Processor) ReapplyFilters(feedID string, feedConfig *config.FeedConfig) (int, int, error) {

	slog.Debug("Re-applying filters", "title", feedConfig.Feed.Title)

	// Get all items for the feed
	items, err := p.itemRepo.GetAllItems(feedID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get feed items: %w", err)
	}

	if len(items) == 0 {
		slog.Debug("No items to re-filter", "title", feedConfig.Feed.Title)
		return 0, 0, nil
	}

	slog.Debug("Starting re-filter process", "items", len(items), "title", feedConfig.Feed.Title)

	updatedCount := 0
	errorCount := 0

	for _, item := range items {
		// Convert database item to normalized item for filtering
		normalizedItem := Item{
			GUID:         item.GUID,
			Link:         item.Link,
			Title:        item.Title,
			Description:  item.Description,
			Content:      item.Content,
			AuthorName:   item.AuthorName,
			Categories:   item.Categories,
		}

		// Apply filters
		shouldFilter, reason := p.applyFilters(normalizedItem, feedConfig.Filters)
		
		// Check if filter status changed
		if shouldFilter != item.IsFiltered || reason != item.FilterReason {
			err := p.itemRepo.UpdateItemFilterStatus(item.ID, shouldFilter, reason)
			if err != nil {
				slog.Warn("Failed to update filter status", "item_id", item.ID, "error", err)
				errorCount++
				continue
			}
			updatedCount++
			
			if shouldFilter && !item.IsFiltered {
				slog.Debug("Item newly filtered", "title", item.Title, "reason", reason)
			} else if !shouldFilter && item.IsFiltered {
				slog.Debug("Item unfiltered", "title", item.Title)
			}
		}
	}

	slog.Info("Re-filter completed",
		"title", feedConfig.Feed.Title,
		"updated", updatedCount,
		"errors", errorCount)

	return updatedCount, errorCount, nil
}

// convertToDBItem converts a feed.Item to database.FeedItem for database operations
func (p *Processor) convertToDBItem(item Item) database.FeedItem {
	return database.FeedItem{
		GUID:          item.GUID,
		Title:         item.Title,
		Link:          item.Link,
		Description:   item.Description,
		Content:       item.Content,
		PublishedDate: item.PublishedDate,
		UpdatedDate:   item.UpdatedDate,
		AuthorName:    item.AuthorName,
		AuthorEmail:   item.AuthorEmail,
		Categories:    item.Categories,
		ContentHash:   item.ContentHash,
		IsFiltered:    item.IsFiltered,
		FilterReason:  item.FilterReason,
	}
}