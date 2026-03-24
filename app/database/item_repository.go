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
		       COALESCE(fi.enclosure_url, ''), COALESCE(fi.enclosure_length, 0), COALESCE(fi.enclosure_type, ''),
		       COALESCE(fi.itunes_duration, 0), COALESCE(fi.itunes_episode, 0), COALESCE(fi.itunes_season, 0), COALESCE(fi.itunes_episode_type, ''), COALESCE(fi.itunes_image, ''),
		       fi.content_extraction_status,
		       fi.media_status, COALESCE(fi.media_path, ''), COALESCE(fi.media_size, 0)
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

func (r *ItemRepository) UpsertItem(feedName string, item types.Item) (string, error) {
	authors := item.Authors
	if authors == nil {
		authors = []string{}
	}

	categories := item.Categories
	if categories == nil {
		categories = []string{}
	}

	var itemID string
	err := r.db.QueryRow(`
		INSERT INTO feed_items (
			feed_id, guid, link, title, description, content,
			published_at, updated_at, authors,
			categories, is_filtered, content_hash,
			enclosure_url, enclosure_length, enclosure_type,
			itunes_duration, itunes_episode, itunes_season, itunes_episode_type, itunes_image,
			content_extraction_status,
			media_status, media_path, media_size
		) VALUES (
			(SELECT id FROM feeds WHERE name = $1),
			$2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24
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
			enclosure_type = EXCLUDED.enclosure_type,
			itunes_duration = EXCLUDED.itunes_duration,
			itunes_episode = EXCLUDED.itunes_episode,
			itunes_season = EXCLUDED.itunes_season,
			itunes_episode_type = EXCLUDED.itunes_episode_type,
			itunes_image = EXCLUDED.itunes_image,
			content_extraction_status = EXCLUDED.content_extraction_status,
			media_status = EXCLUDED.media_status,
			media_path = EXCLUDED.media_path,
			media_size = EXCLUDED.media_size
		RETURNING id
	`, feedName, item.GUID, item.Link, item.Title, item.Description, item.Content,
		item.PublishedAt, item.UpdatedAt, pq.Array(authors),
		pq.Array(categories), item.IsFiltered,
		item.ContentHash, item.EnclosureURL, item.EnclosureLength, item.EnclosureType,
		item.ITunesDuration, item.ITunesEpisode, item.ITunesSeason, item.ITunesEpisodeType, item.ITunesImage,
		item.ContentExtractionStatus,
		item.MediaStatus, item.MediaPath, item.MediaSize).Scan(&itemID)

	if err != nil {
		return "", fmt.Errorf("failed to upsert item: %w", err)
	}

	return itemID, nil
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
		       COALESCE(fi.enclosure_url, ''), fi.enclosure_length, COALESCE(fi.enclosure_type, ''),
		       COALESCE(fi.itunes_duration, 0), COALESCE(fi.itunes_episode, 0), COALESCE(fi.itunes_season, 0), COALESCE(fi.itunes_episode_type, ''), COALESCE(fi.itunes_image, ''),
		       fi.content_extraction_status,
		       fi.media_status, COALESCE(fi.media_path, ''), COALESCE(fi.media_size, 0)
		FROM feed_items fi
		JOIN feeds f ON fi.feed_id = f.id
		WHERE f.name = $1
		  AND fi.is_filtered = false
		  AND (fi.content_extraction_status IS NULL OR fi.content_extraction_status IN ('ready', 'failed'))
		  AND (fi.media_status IS NULL OR fi.media_status = 'ready')
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
			&item.ITunesDuration, &item.ITunesEpisode, &item.ITunesSeason, &item.ITunesEpisodeType, &item.ITunesImage,
			&item.ContentExtractionStatus,
			&item.MediaStatus, &item.MediaPath, &item.MediaSize,
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

func (r *ItemRepository) GetItemByID(itemID string) (*Item, error) {
	var item Item
	err := r.db.QueryRow(`
		SELECT fi.id, fi.guid, COALESCE(fi.link, ''), COALESCE(fi.title, ''),
		       COALESCE(fi.description, ''), COALESCE(fi.content, ''),
		       fi.published_at, fi.updated_at, COALESCE(fi.authors, '{}'),
		       COALESCE(fi.categories, '{}'),
		       fi.is_filtered,
		       fi.content_hash, fi.created_at,
		       COALESCE(fi.enclosure_url, ''), COALESCE(fi.enclosure_length, 0), COALESCE(fi.enclosure_type, ''),
		       COALESCE(fi.itunes_duration, 0), COALESCE(fi.itunes_episode, 0), COALESCE(fi.itunes_season, 0), COALESCE(fi.itunes_episode_type, ''), COALESCE(fi.itunes_image, ''),
		       fi.content_extraction_status,
		       fi.media_status, COALESCE(fi.media_path, ''), COALESCE(fi.media_size, 0)
		FROM feed_items fi
		WHERE fi.id = $1
	`, itemID).Scan(
		&item.ID, &item.GUID, &item.Link, &item.Title,
		&item.Description, &item.Content, &item.PublishedAt, &item.UpdatedAt,
		pq.Array(&item.Authors), pq.Array(&item.Categories),
		&item.IsFiltered,
		&item.ContentHash, &item.CreatedAt,
		&item.EnclosureURL, &item.EnclosureLength, &item.EnclosureType,
		&item.ITunesDuration, &item.ITunesEpisode, &item.ITunesSeason, &item.ITunesEpisodeType, &item.ITunesImage,
		&item.ContentExtractionStatus,
		&item.MediaStatus, &item.MediaPath, &item.MediaSize,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get item by ID: %w", err)
	}

	return &item, nil
}

func (r *ItemRepository) UpdateMediaStatus(itemID, status, mediaPath string, mediaSize int64) error {
	_, err := r.db.Exec(`
		UPDATE feed_items
		SET media_status = $2, media_path = $3, media_size = $4
		WHERE id = $1
	`, itemID, status, mediaPath, mediaSize)

	if err != nil {
		return fmt.Errorf("failed to update media status: %w", err)
	}

	return nil
}

type MediaInfo struct {
	MediaPath string
	MediaSize int64
}

func (r *ItemRepository) GetReadyMediaByPath(mediaPath string) (*MediaInfo, error) {
	var info MediaInfo
	err := r.db.QueryRow(`
		SELECT media_path, media_size FROM feed_items
		WHERE media_path = $1 AND media_status = 'ready'
		LIMIT 1
	`, mediaPath).Scan(&info.MediaPath, &info.MediaSize)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ready media by path: %w", err)
	}

	return &info, nil
}

func (r *ItemRepository) GetAllActiveMediaPaths() ([]string, error) {
	rows, err := r.db.Query(`
		SELECT DISTINCT fi.media_path
		FROM feed_items fi
		JOIN feeds f ON fi.feed_id = f.id
		WHERE f.is_enabled = true
		  AND fi.media_status = 'ready'
		  AND fi.media_path IS NOT NULL
		  AND fi.is_filtered = false
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get active media paths: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("failed to scan media path: %w", err)
		}
		paths = append(paths, path)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media paths: %w", err)
	}

	return paths, nil
}

func (r *ItemRepository) UpdateContentExtractionStatus(itemID, status, content string) error {
	_, err := r.db.Exec(`
		UPDATE feed_items
		SET content_extraction_status = $2, content = CASE WHEN $3 = '' THEN content ELSE $3 END
		WHERE id = $1
	`, itemID, status, content)

	if err != nil {
		return fmt.Errorf("failed to update content extraction status: %w", err)
	}

	return nil
}
