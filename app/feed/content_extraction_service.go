package feed

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/database"
)

// ContentExtractionService handles content extraction for feed items
type ContentExtractionService struct {
	extractor  *ContentExtractor
	itemRepo   database.ItemContentExtractor
	itemReader database.ItemReader
	itemWriter database.ItemWriter
}

// NewContentExtractionService creates a new content extraction service
func NewContentExtractionService(itemRepo ItemRepositoryInterface) *ContentExtractionService {
	return &ContentExtractionService{
		extractor:  NewContentExtractor(10 * time.Second), // Default timeout
		itemRepo:   itemRepo,
		itemReader: itemRepo,
		itemWriter: itemRepo,
	}
}

// ExtractContentForFeed extracts content for items in the specified feed
func (s *ContentExtractionService) ExtractContentForFeed(ctx context.Context, feedID string, feedConfig *config.FeedConfig) error {
	if !feedConfig.Settings.ExtractContent {
		return nil
	}

	// Get items that need content extraction
	items, err := s.itemRepo.GetItemsForExtraction(feedID, feedConfig.Settings.MaxItems)
	if err != nil {
		return fmt.Errorf("failed to get items for extraction: %w", err)
	}

	if len(items) == 0 {
		slog.Debug("No items need content extraction", "feed_id", feedID)
		return nil
	}

	slog.Debug("Starting content extraction", "feed_id", feedID, "items", len(items))

	successCount := 0
	failCount := 0

	for _, item := range items {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Increment attempts first
		if err := s.itemRepo.IncrementExtractionAttempts(item.ID); err != nil {
			slog.Warn("Failed to increment extraction attempts", "item_id", item.ID, "error", err)
		}

		// Extract content with timeout
		extractCtx, cancel := context.WithTimeout(ctx, feedConfig.Settings.GetExtractionTimeout())
		content, err := s.extractor.ExtractContent(extractCtx, item.Link)
		cancel()

		now := time.Now()
		
		if err != nil {
			// Log error and update status
			slog.Debug("Content extraction failed", 
				"item_id", item.ID, 
				"link", item.Link, 
				"error", err)
			
			if updateErr := s.itemRepo.UpdateExtractionStatus(item.ID, "failed", &now, err.Error()); updateErr != nil {
				slog.Warn("Failed to update extraction status", "item_id", item.ID, "error", updateErr)
			}
			
			failCount++
			continue
		}

		// Update the item with extracted content
		if updateErr := s.itemRepo.UpdateExtractedContent(item.ID, content); updateErr != nil {
			slog.Warn("Failed to update extracted content", "item_id", item.ID, "error", updateErr)
			
			if statusErr := s.itemRepo.UpdateExtractionStatus(item.ID, "failed", &now, updateErr.Error()); statusErr != nil {
				slog.Warn("Failed to update extraction status", "item_id", item.ID, "error", statusErr)
			}
			
			failCount++
			continue
		}

		// Update extraction status to success
		if err := s.itemRepo.UpdateExtractionStatus(item.ID, "success", &now, ""); err != nil {
			slog.Warn("Failed to update extraction status", "item_id", item.ID, "error", err)
		}

		slog.Debug("Content extracted successfully", 
			"item_id", item.ID, 
			"link", item.Link,
			"content_length", len(content))

		successCount++
	}

	slog.Info("Content extraction completed",
		"feed_id", feedID,
		"total_items", len(items),
		"success", successCount,
		"failed", failCount)

	return nil
}