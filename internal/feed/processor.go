package feed

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/lysyi3m/rss-comb/internal/config"
	"github.com/lysyi3m/rss-comb/internal/database"
	"github.com/lysyi3m/rss-comb/internal/parser"
)

// Processor handles feed processing including fetching, parsing, filtering, and storage
type Processor struct {
	parser    *parser.Parser
	feedRepo  *database.FeedRepository
	itemRepo  *database.ItemRepository
	configs   map[string]*config.FeedConfig
	client    *http.Client
}

// NewProcessor creates a new feed processor
func NewProcessor(p *parser.Parser, fr *database.FeedRepository,
	ir *database.ItemRepository, configs map[string]*config.FeedConfig) *Processor {
	return &Processor{
		parser:   p,
		feedRepo: fr,
		itemRepo: ir,
		configs:  configs,
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
	}
}

// ProcessFeed processes a single feed
func (p *Processor) ProcessFeed(feedID, configFile string) error {
	feedConfig, ok := p.configs[configFile]
	if !ok {
		return fmt.Errorf("configuration not found: %s", configFile)
	}

	if !feedConfig.Settings.Enabled {
		log.Printf("Feed %s is disabled, skipping", feedConfig.Feed.Name)
		return nil
	}

	log.Printf("Processing feed: %s (%s)", feedConfig.Feed.Name, feedConfig.Feed.URL)
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
	if err := p.feedRepo.UpdateFeedMetadata(feedID, metadata.IconURL, metadata.Language); err != nil {
		return fmt.Errorf("failed to update feed metadata: %w", err)
	}

	// Process items
	processedCount, skippedCount, filteredCount := 0, 0, 0
	for i, item := range items {
		// Stop if we've reached the max items limit
		if processedCount >= feedConfig.Settings.MaxItems {
			log.Printf("Reached max items limit (%d) for feed %s", feedConfig.Settings.MaxItems, feedConfig.Feed.Name)
			break
		}

		// Check for duplicates BEFORE storing (skip duplicates entirely)
		if feedConfig.Settings.Deduplication {
			isDup, _, err := p.itemRepo.CheckDuplicate(item.ContentHash, feedID, false)
			if err != nil {
				log.Printf("Warning: failed to check duplicate for item %d: %v", i, err)
			} else if isDup {
				skippedCount++
				log.Printf("Item %d is duplicate, skipping", i)
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
			log.Printf("Item %d filtered: %s", i, reason)
		}

		// Store item (duplicates already skipped)
		if err := p.itemRepo.StoreItem(feedID, item); err != nil {
			log.Printf("Warning: failed to store item %d: %v", i, err)
			continue
		}

		processedCount++
	}

	// Update next fetch time
	nextFetch := time.Now().Add(feedConfig.Settings.GetRefreshInterval())
	if err := p.feedRepo.UpdateNextFetch(feedID, nextFetch); err != nil {
		return fmt.Errorf("failed to update next fetch time: %w", err)
	}

	duration := time.Since(startTime)
	newItems := processedCount - filteredCount
	log.Printf("Processed feed %s: %d items (%d new, %d skipped duplicates, %d filtered) in %v",
		feedConfig.Feed.Name, len(items), newItems, skippedCount, filteredCount, duration)

	return nil
}

// fetchFeed fetches feed data from the given URL
func (p *Processor) fetchFeed(url string, settings config.FeedSettings) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", config.GetUserAgent())
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
		log.Printf("Warning: unexpected content type: %s", contentType)
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
func (p *Processor) applyFilters(item parser.NormalizedItem, filters []config.Filter) (bool, string) {
	for _, filter := range filters {
		value := p.getFieldValue(item, filter.Field)

		// Check excludes first (if any exclude matches, item is filtered)
		for _, exclude := range filter.Excludes {
			if strings.Contains(strings.ToLower(value), strings.ToLower(exclude)) {
				return true, fmt.Sprintf("Excluded by %s filter: contains '%s'", filter.Field, exclude)
			}
		}

		// If includes are specified, at least one must match
		if len(filter.Includes) > 0 {
			matched := false
			for _, include := range filter.Includes {
				if strings.Contains(strings.ToLower(value), strings.ToLower(include)) {
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

// getFieldValue extracts the value of a specific field from an item
func (p *Processor) getFieldValue(item parser.NormalizedItem, field string) string {
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

// ReloadConfigs reloads the feed configurations
func (p *Processor) ReloadConfigs(configs map[string]*config.FeedConfig) {
	p.configs = configs
	log.Printf("Reloaded %d feed configurations", len(configs))
}

// IsFeedEnabled checks if a feed is enabled in its configuration
func (p *Processor) IsFeedEnabled(configFile string) bool {
	feedConfig, ok := p.configs[configFile]
	if !ok {
		return false // Configuration not found, treat as disabled
	}
	return feedConfig.Settings.Enabled
}

// GetStats returns processing statistics
func (p *Processor) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"loaded_configs": len(p.configs),
		"client_timeout": p.client.Timeout.String(),
	}
}