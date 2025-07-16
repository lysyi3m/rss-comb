package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var _ ItemReader = (*ItemRepository)(nil)
var _ ItemWriter = (*ItemRepository)(nil)
var _ ItemDuplicateChecker = (*ItemRepository)(nil)
var _ ItemContentExtractor = (*ItemRepository)(nil)

type ItemRepository struct {
	db *DB
}

func NewItemRepository(db *DB) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) CheckDuplicate(contentHash, feedID string) (bool, *string, error) {
	var duplicateID sql.NullString
	
	// Scope duplicate check to feed to allow same content across different feeds
	query := `SELECT id FROM feed_items WHERE feed_id = $1 AND content_hash = $2 LIMIT 1`
	err := r.db.QueryRow(query, feedID, contentHash).Scan(&duplicateID)
	if err == sql.ErrNoRows {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to check duplicate: %w", err)
	}

	id := duplicateID.String
	return true, &id, nil
}

func (r *ItemRepository) StoreItem(feedID string, item FeedItem) error {
	// Ensure authors is never nil to prevent NULL constraint violations
	authors := item.Authors
	if authors == nil {
		authors = []string{}
	}
	
	// Ensure categories is never nil to prevent potential issues
	categories := item.Categories
	if categories == nil {
		categories = []string{}
	}
	
	_, err := r.db.Exec(`
		INSERT INTO feed_items (
			feed_id, guid, link, title, description, content,
			published_at, updated_at, authors,
			categories, is_filtered, filter_reason, content_hash
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (feed_id, guid) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			updated_at = EXCLUDED.updated_at,
			authors = EXCLUDED.authors,
			categories = EXCLUDED.categories,
			is_filtered = EXCLUDED.is_filtered,
			filter_reason = EXCLUDED.filter_reason,
			content_hash = EXCLUDED.content_hash
	`, feedID, item.GUID, item.Link, item.Title, item.Description, item.Content,
		item.PublishedAt, item.UpdatedAt, pq.Array(authors),
		pq.Array(categories), item.IsFiltered, item.FilterReason,
		item.ContentHash)

	if err != nil {
		return fmt.Errorf("failed to store item: %w", err)
	}

	return nil
}

func (r *ItemRepository) GetVisibleItems(feedID string, limit int) ([]Item, error) {
	rows, err := r.db.Query(`
		SELECT id, feed_id, guid, COALESCE(link, ''), COALESCE(title, ''), 
		       COALESCE(description, ''), COALESCE(content, ''),
		       published_at, updated_at, COALESCE(authors, '{}'), 
		       COALESCE(categories, '{}'),
		       is_filtered, COALESCE(filter_reason, ''),
		       content_hash, created_at
		FROM feed_items
		WHERE feed_id = $1
		  AND is_filtered = false
		ORDER BY COALESCE(published_at, created_at) DESC
		LIMIT $2
	`, feedID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get visible items: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.ID, &item.FeedID, &item.GUID, &item.Link, &item.Title,
			&item.Description, &item.Content, &item.PublishedAt, &item.UpdatedAt,
			pq.Array(&item.Authors), pq.Array(&item.Categories),
			&item.IsFiltered, &item.FilterReason,
			&item.ContentHash, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating item rows: %w", err)
	}

	return items, nil
}

func (r *ItemRepository) GetItemCount(feedID string) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM feed_items WHERE feed_id = $1", feedID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get item count: %w", err)
	}
	return count, nil
}

func (r *ItemRepository) GetItemStats(feedID string) (total, visible, filtered int, err error) {
	err = r.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN is_filtered = false THEN 1 ELSE 0 END) as visible,
			SUM(CASE WHEN is_filtered = true THEN 1 ELSE 0 END) as filtered
		FROM feed_items 
		WHERE feed_id = $1
	`, feedID).Scan(&total, &visible, &filtered)

	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get item stats: %w", err)
	}

	return total, visible, filtered, nil
}

func (r *ItemRepository) GetAllItems(feedID string) ([]Item, error) {
	rows, err := r.db.Query(`
		SELECT id, feed_id, guid, COALESCE(link, ''), COALESCE(title, ''), 
		       COALESCE(description, ''), COALESCE(content, ''),
		       published_at, updated_at, COALESCE(authors, '{}'), 
		       COALESCE(categories, '{}'),
		       is_filtered, COALESCE(filter_reason, ''),
		       content_hash, created_at
		FROM feed_items
		WHERE feed_id = $1
		ORDER BY COALESCE(published_at, created_at) DESC
	`, feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all items: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.ID, &item.FeedID, &item.GUID, &item.Link, &item.Title,
			&item.Description, &item.Content, &item.PublishedAt, &item.UpdatedAt,
			pq.Array(&item.Authors), pq.Array(&item.Categories),
			&item.IsFiltered, &item.FilterReason,
			&item.ContentHash, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item row: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating item rows: %w", err)
	}

	return items, nil
}

func (r *ItemRepository) UpdateItemFilterStatus(itemID string, isFiltered bool, filterReason string) error {
	_, err := r.db.Exec(`
		UPDATE feed_items 
		SET is_filtered = $2, filter_reason = $3
		WHERE id = $1
	`, itemID, isFiltered, filterReason)
	
	if err != nil {
		return fmt.Errorf("failed to update item filter status: %w", err)
	}
	
	return nil
}

// GetItemsForExtraction returns items that need content extraction (only ID and link)
func (r *ItemRepository) GetItemsForExtraction(feedID string, limit int) ([]ItemForExtraction, error) {
	query := `
		SELECT id, link
		FROM feed_items 
		WHERE feed_id = $1 
		  AND link IS NOT NULL 
		  AND link != ''
		  AND is_filtered = false
		  AND (content_extraction_status = 'pending' OR content_extraction_status IS NULL)
		  AND extraction_attempts < 3
		ORDER BY published_at DESC
		LIMIT $2`

	rows, err := r.db.Query(query, feedID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query items for extraction: %w", err)
	}
	defer rows.Close()

	var items []ItemForExtraction
	for rows.Next() {
		var item ItemForExtraction
		
		err := rows.Scan(&item.ID, &item.Link)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item row: %w", err)
		}
		
		items = append(items, item)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over rows: %w", err)
	}
	
	return items, nil
}

// UpdateExtractionStatus updates the content extraction status for an item
func (r *ItemRepository) UpdateExtractionStatus(itemID string, status string, extractedAt *time.Time, errorMsg string) error {
	query := `
		UPDATE feed_items 
		SET content_extraction_status = $1, 
			content_extracted_at = $2, 
			content_extraction_error = $3
		WHERE id = $4`

	_, err := r.db.Exec(query, status, extractedAt, errorMsg, itemID)
	if err != nil {
		return fmt.Errorf("failed to update extraction status: %w", err)
	}
	
	return nil
}

// IncrementExtractionAttempts increments the extraction attempts counter
func (r *ItemRepository) IncrementExtractionAttempts(itemID string) error {
	query := `
		UPDATE feed_items 
		SET extraction_attempts = extraction_attempts + 1
		WHERE id = $1`

	_, err := r.db.Exec(query, itemID)
	if err != nil {
		return fmt.Errorf("failed to increment extraction attempts: %w", err)
	}
	
	return nil
}

// UpdateExtractedContent updates the content field with extracted content
func (r *ItemRepository) UpdateExtractedContent(itemID string, content string) error {
	query := `
		UPDATE feed_items 
		SET content = $1
		WHERE id = $2`

	_, err := r.db.Exec(query, content, itemID)
	if err != nil {
		return fmt.Errorf("failed to update extracted content: %w", err)
	}
	
	return nil
}
