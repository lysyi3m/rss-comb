package database

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/lysyi3m/rss-comb/app/types"
)

type ItemRepository struct {
	db *DB
}

func NewItemRepository(db *DB) *ItemRepository {
	return &ItemRepository{db: db}
}

func (r *ItemRepository) GetAllItems(feedName string) ([]Item, error) {
	rows, err := r.db.Query(`
		SELECT fi.id, fi.guid, COALESCE(fi.link, ''), COALESCE(fi.title, ''), 
		       COALESCE(fi.description, ''), COALESCE(fi.content, ''),
		       fi.published_at, fi.updated_at, COALESCE(fi.authors, '{}'), 
		       COALESCE(fi.categories, '{}'),
		       fi.is_filtered,
		       fi.content_hash, fi.created_at,
		       COALESCE(fi.enclosure_url, ''), COALESCE(fi.enclosure_length, 0), COALESCE(fi.enclosure_type, '')
		FROM feed_items fi
		JOIN feeds f ON fi.feed_id = f.id
		WHERE f.name = $1
		ORDER BY fi.published_at DESC
	`, feedName)
	if err != nil {
		return nil, fmt.Errorf("failed to get all items: %w", err)
	}
	defer rows.Close()

	return r.scanItemRows(rows)
}

func (r *ItemRepository) UpsertItem(feedName string, item types.Item) error {
	authors := item.Authors
	if authors == nil {
		authors = []string{}
	}

	categories := item.Categories
	if categories == nil {
		categories = []string{}
	}

	_, err := r.db.Exec(`
		INSERT INTO feed_items (
			feed_id, guid, link, title, description, content,
			published_at, updated_at, authors,
			categories, is_filtered, content_hash,
			enclosure_url, enclosure_length, enclosure_type
		) VALUES (
			(SELECT id FROM feeds WHERE name = $1),
			$2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (feed_id, guid) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			updated_at = EXCLUDED.updated_at,
			authors = EXCLUDED.authors,
			categories = EXCLUDED.categories,
			is_filtered = EXCLUDED.is_filtered,
			content_hash = EXCLUDED.content_hash,
			enclosure_url = EXCLUDED.enclosure_url,
			enclosure_length = EXCLUDED.enclosure_length,
			enclosure_type = EXCLUDED.enclosure_type
	`, feedName, item.GUID, item.Link, item.Title, item.Description, item.Content,
		item.PublishedAt, item.UpdatedAt, pq.Array(authors),
		pq.Array(categories), item.IsFiltered,
		item.ContentHash, item.EnclosureURL, item.EnclosureLength, item.EnclosureType)

	if err != nil {
		return fmt.Errorf("failed to upsert item: %w", err)
	}

	return nil
}

func (r *ItemRepository) UpdateItemFilterStatus(itemID string, isFiltered bool) error {
	_, err := r.db.Exec(`
		UPDATE feed_items 
		SET is_filtered = $2
		WHERE id = $1
	`, itemID, isFiltered)

	if err != nil {
		return fmt.Errorf("failed to update item filter status: %w", err)
	}

	return nil
}

func (r *ItemRepository) CheckDuplicate(feedName, contentHash string) (bool, *string, error) {
	var duplicateID sql.NullString

	query := `
		SELECT fi.id 
		FROM feed_items fi 
		JOIN feeds f ON fi.feed_id = f.id 
		WHERE f.name = $1 AND fi.content_hash = $2 
		LIMIT 1`
	err := r.db.QueryRow(query, feedName, contentHash).Scan(&duplicateID)
	if err == sql.ErrNoRows {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("failed to check duplicate: %w", err)
	}

	id := duplicateID.String
	return true, &id, nil
}

func (r *ItemRepository) GetVisibleItems(feedName string, limit int) ([]Item, error) {
	rows, err := r.db.Query(`
		SELECT fi.id, fi.guid, COALESCE(fi.link, ''), COALESCE(fi.title, ''),
		       COALESCE(fi.description, ''), COALESCE(fi.content, ''),
		       fi.published_at, fi.updated_at, fi.authors, fi.categories, fi.is_filtered,
		       fi.content_hash, fi.created_at,
		       COALESCE(fi.enclosure_url, ''), fi.enclosure_length, COALESCE(fi.enclosure_type, '')
		FROM feed_items fi
		JOIN feeds f ON fi.feed_id = f.id
		WHERE f.name = $1
		  AND fi.is_filtered = false
		ORDER BY fi.published_at DESC
		LIMIT $2
	`, feedName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get visible items: %w", err)
	}
	defer rows.Close()

	return r.scanItemRows(rows)
}

func (r *ItemRepository) scanItemRows(rows *sql.Rows) ([]Item, error) {
	var items []Item
	for rows.Next() {
		var item Item
		err := rows.Scan(
			&item.ID, &item.GUID, &item.Link, &item.Title,
			&item.Description, &item.Content, &item.PublishedAt, &item.UpdatedAt,
			pq.Array(&item.Authors), pq.Array(&item.Categories),
			&item.IsFiltered,
			&item.ContentHash, &item.CreatedAt,
			&item.EnclosureURL, &item.EnclosureLength, &item.EnclosureType,
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
