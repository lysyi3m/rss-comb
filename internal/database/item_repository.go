package database

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	"github.com/lysyi3m/rss-comb/internal/parser"
)

// ItemRepository handles database operations for feed items
type ItemRepository struct {
	db *DB
}

// NewItemRepository creates a new item repository
func NewItemRepository(db *DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// CheckDuplicate checks if an item with the given content hash already exists
func (r *ItemRepository) CheckDuplicate(contentHash, feedID string, global bool) (bool, *string, error) {
	var duplicateID sql.NullString
	var query string
	var args []interface{}

	if global {
		// Check for duplicates across all feeds
		query = `SELECT id FROM feed_items WHERE content_hash = $1 LIMIT 1`
		args = []interface{}{contentHash}
	} else {
		// Check for duplicates within the same feed
		query = `SELECT id FROM feed_items WHERE feed_id = $1 AND content_hash = $2 LIMIT 1`
		args = []interface{}{feedID, contentHash}
	}

	err := r.db.QueryRow(query, args...).Scan(&duplicateID)
	if err == sql.ErrNoRows {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to check duplicate: %w", err)
	}

	id := duplicateID.String
	return true, &id, nil
}

// StoreItem stores a normalized item in the database
func (r *ItemRepository) StoreItem(feedID string, item parser.NormalizedItem) error {
	rawDataJSON, err := json.Marshal(item.RawData)
	if err != nil {
		return fmt.Errorf("failed to marshal raw data: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO feed_items (
			feed_id, guid, link, title, description, content,
			published_date, updated_date, author_name, author_email,
			categories, is_filtered, filter_reason, content_hash, raw_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (feed_id, guid) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			updated_date = EXCLUDED.updated_date,
			is_filtered = EXCLUDED.is_filtered,
			filter_reason = EXCLUDED.filter_reason,
			content_hash = EXCLUDED.content_hash,
			raw_data = EXCLUDED.raw_data
	`, feedID, item.GUID, item.Link, item.Title, item.Description, item.Content,
		item.PublishedDate, item.UpdatedDate, item.AuthorName, item.AuthorEmail,
		pq.Array(item.Categories), item.IsFiltered, item.FilterReason,
		item.ContentHash, rawDataJSON)

	if err != nil {
		return fmt.Errorf("failed to store item: %w", err)
	}

	return nil
}

// GetVisibleItems returns non-filtered items for a feed
func (r *ItemRepository) GetVisibleItems(feedID string, limit int) ([]Item, error) {
	rows, err := r.db.Query(`
		SELECT id, feed_id, guid, COALESCE(link, ''), COALESCE(title, ''), 
		       COALESCE(description, ''), COALESCE(content, ''),
		       published_date, updated_date, COALESCE(author_name, ''), 
		       COALESCE(author_email, ''), COALESCE(categories, '{}'),
		       is_filtered, COALESCE(filter_reason, ''),
		       content_hash, created_at
		FROM feed_items
		WHERE feed_id = $1
		  AND is_filtered = false
		ORDER BY COALESCE(published_date, created_at) DESC
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
			&item.Description, &item.Content, &item.PublishedDate, &item.UpdatedDate,
			&item.AuthorName, &item.AuthorEmail, pq.Array(&item.Categories),
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

// GetItemCount returns the total number of items for a feed
func (r *ItemRepository) GetItemCount(feedID string) (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM feed_items WHERE feed_id = $1", feedID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get item count: %w", err)
	}
	return count, nil
}

// GetItemStats returns statistics about items for a feed
func (r *ItemRepository) GetItemStats(feedID string) (total, visible, duplicates, filtered int, err error) {
	err = r.db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN is_filtered = false THEN 1 ELSE 0 END) as visible,
			0 as duplicates,
			SUM(CASE WHEN is_filtered = true THEN 1 ELSE 0 END) as filtered
		FROM feed_items 
		WHERE feed_id = $1
	`, feedID).Scan(&total, &visible, &duplicates, &filtered)

	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get item stats: %w", err)
	}

	return total, visible, duplicates, filtered, nil
}

// DeleteOldItems deletes items older than the specified number of days for a feed
func (r *ItemRepository) DeleteOldItems(feedID string, daysOld int) (int, error) {
	result, err := r.db.Exec(`
		DELETE FROM feed_items 
		WHERE feed_id = $1 
		  AND created_at < NOW() - INTERVAL '%d days'
	`, feedID, daysOld)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old items: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// GetAllItems returns all items for a feed (including filtered ones)
func (r *ItemRepository) GetAllItems(feedID string) ([]Item, error) {
	rows, err := r.db.Query(`
		SELECT id, feed_id, guid, COALESCE(link, ''), COALESCE(title, ''), 
		       COALESCE(description, ''), COALESCE(content, ''),
		       published_date, updated_date, COALESCE(author_name, ''), 
		       COALESCE(author_email, ''), COALESCE(categories, '{}'),
		       is_filtered, COALESCE(filter_reason, ''),
		       content_hash, created_at
		FROM feed_items
		WHERE feed_id = $1
		ORDER BY COALESCE(published_date, created_at) DESC
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
			&item.Description, &item.Content, &item.PublishedDate, &item.UpdatedDate,
			&item.AuthorName, &item.AuthorEmail, pq.Array(&item.Categories),
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

// UpdateItemFilterStatus updates the filter status of an item
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