package database

import (
	"database/sql"
	"fmt"
	"time"

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
		       fi.published_at, fi.updated_at, COALESCE(fi.authors, '[]'),
		       COALESCE(fi.categories, '[]'),
		       fi.is_filtered,
		       fi.content_hash, fi.created_at,
		       COALESCE(fi.enclosure_url, ''), COALESCE(fi.enclosure_length, 0), COALESCE(fi.enclosure_type, ''),
		       COALESCE(fi.itunes_duration, 0), COALESCE(fi.itunes_episode, 0), COALESCE(fi.itunes_season, 0), COALESCE(fi.itunes_episode_type, ''), COALESCE(fi.itunes_image, ''),
		       fi.content_extraction_status,
		       fi.media_status, COALESCE(fi.media_path, ''), COALESCE(fi.media_size, 0)
		FROM feed_items fi
		JOIN feeds f ON fi.feed_id = f.id
		WHERE f.name = ?1
		ORDER BY fi.published_at DESC
	`, feedName)
	if err != nil {
		return nil, fmt.Errorf("failed to get all items: %w", err)
	}
	defer rows.Close()

	return r.scanItemRows(rows)
}

// UpsertItem stores source fields. On conflict it keeps processing state and its
// results, so an edited source item does not lose its extracted content or media.
func (r *ItemRepository) UpsertItem(feedName string, item types.Item) (string, error) {
	return upsertItem(r.db, feedName, item)
}

type StoredItem struct {
	ID            string
	ExtractionJob bool
	MediaJob      bool
}

// StoreItem upserts an item and creates the jobs for its pending work in one
// transaction, so a crash cannot leave a hidden item with no job. The stored status
// decides: an edited item may already be processed. Filtered items get no job.
func (r *ItemRepository) StoreItem(feedName, feedID string, item types.Item) (StoredItem, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return StoredItem{}, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	id, err := upsertItem(tx, feedName, item)
	if err != nil {
		return StoredItem{}, err
	}
	stored := StoredItem{ID: id}

	var extractionStatus, mediaStatus sql.NullString
	err = tx.QueryRow(`SELECT content_extraction_status, media_status FROM feed_items WHERE id = ?1`, id).Scan(&extractionStatus, &mediaStatus)
	if err != nil {
		return StoredItem{}, fmt.Errorf("failed to read stored item: %w", err)
	}

	if !item.IsFiltered && extractionStatus.String == "pending" {
		if stored.ExtractionJob, err = createJob(tx, "extract_content", feedID, &id, ExtractionJobMaxRetries); err != nil {
			return StoredItem{}, err
		}
	}
	if !item.IsFiltered && mediaStatus.String == "pending" {
		if stored.MediaJob, err = createJob(tx, "download_media", feedID, &id, MediaJobMaxRetries); err != nil {
			return StoredItem{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return StoredItem{}, fmt.Errorf("failed to commit item: %w", err)
	}
	return stored, nil
}

// RequeueMedia queues downloads for the feed's newest maxItems eligible items that
// have no media and no job: items with no status, such as after a filter change or
// eviction, and pending items that lost their job. includeFailed also retries items
// whose retries ran out. It returns the number of jobs created.
func (r *ItemRepository) RequeueMedia(feedID string, maxItems int, includeFailed bool) (int, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT w.id FROM (
			SELECT id, media_status FROM feed_items
			WHERE feed_id = ?1 AND is_filtered = 0 AND COALESCE(media_status, '') != 'skipped'
			ORDER BY published_at DESC LIMIT ?2
		) w
		WHERE w.media_status IS NULL
		   OR (?3 AND w.media_status = 'failed')
		   OR (w.media_status = 'pending' AND NOT EXISTS (SELECT 1 FROM jobs j WHERE j.item_id = w.id))
	`, feedID, maxItems, includeFailed)
	if err != nil {
		return 0, fmt.Errorf("failed to find items to requeue: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("failed to scan item to requeue: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("error iterating items to requeue: %w", err)
	}
	rows.Close()

	created := 0
	for _, id := range ids {
		if _, err := tx.Exec(`UPDATE feed_items SET media_status = 'pending' WHERE id = ?1`, id); err != nil {
			return 0, fmt.Errorf("failed to requeue item: %w", err)
		}
		ok, err := createJob(tx, "download_media", feedID, &id, MediaJobMaxRetries)
		if err != nil {
			return 0, err
		}
		if ok {
			created++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit requeue: %w", err)
	}
	return created, nil
}

func upsertItem(q querier, feedName string, item types.Item) (string, error) {
	authors := item.Authors
	if authors == nil {
		authors = []string{}
	}

	categories := item.Categories
	if categories == nil {
		categories = []string{}
	}

	var itemID string
	err := q.QueryRow(`
		INSERT INTO feed_items (
			feed_id, guid, link, title, description, content,
			published_at, updated_at, authors,
			categories, is_filtered, content_hash,
			enclosure_url, enclosure_length, enclosure_type,
			itunes_duration, itunes_episode, itunes_season, itunes_episode_type, itunes_image,
			content_extraction_status,
			media_status, media_path, media_size
		) VALUES (
			(SELECT id FROM feeds WHERE name = ?1),
			?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12, ?13, ?14, ?15, ?16, ?17, ?18, ?19, ?20, ?21, ?22, ?23, ?24
		)
		ON CONFLICT (feed_id, guid) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			content = CASE WHEN feed_items.content_extraction_status = 'ready' THEN feed_items.content ELSE EXCLUDED.content END,
			updated_at = EXCLUDED.updated_at,
			authors = EXCLUDED.authors,
			categories = EXCLUDED.categories,
			is_filtered = EXCLUDED.is_filtered,
			content_hash = EXCLUDED.content_hash,
			enclosure_url = EXCLUDED.enclosure_url,
			enclosure_length = EXCLUDED.enclosure_length,
			enclosure_type = EXCLUDED.enclosure_type,
			itunes_duration = CASE WHEN feed_items.media_status IS NOT NULL THEN feed_items.itunes_duration ELSE EXCLUDED.itunes_duration END,
			itunes_episode = EXCLUDED.itunes_episode,
			itunes_season = EXCLUDED.itunes_season,
			itunes_episode_type = EXCLUDED.itunes_episode_type,
			itunes_image = EXCLUDED.itunes_image,
			content_extraction_status = COALESCE(feed_items.content_extraction_status, EXCLUDED.content_extraction_status),
			media_status = COALESCE(feed_items.media_status, EXCLUDED.media_status),
			media_path = CASE WHEN feed_items.media_status IS NOT NULL THEN feed_items.media_path ELSE EXCLUDED.media_path END,
			media_size = CASE WHEN feed_items.media_status IS NOT NULL THEN feed_items.media_size ELSE EXCLUDED.media_size END
		RETURNING id
	`, feedName, item.GUID, item.Link, item.Title, item.Description, item.Content,
		sqliteTime(item.PublishedAt), sqliteTimePtr(item.UpdatedAt), encodeStringSlice(authors),
		encodeStringSlice(categories), item.IsFiltered,
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
		SET is_filtered = ?2
		WHERE id = ?1
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
		WHERE f.name = ?1 AND fi.content_hash = ?2
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
		       COALESCE(fi.enclosure_url, ''), COALESCE(fi.enclosure_length, 0), COALESCE(fi.enclosure_type, ''),
		       COALESCE(fi.itunes_duration, 0), COALESCE(fi.itunes_episode, 0), COALESCE(fi.itunes_season, 0), COALESCE(fi.itunes_episode_type, ''), COALESCE(fi.itunes_image, ''),
		       fi.content_extraction_status,
		       fi.media_status, COALESCE(fi.media_path, ''), COALESCE(fi.media_size, 0)
		FROM feed_items fi
		JOIN feeds f ON fi.feed_id = f.id
		WHERE f.name = ?1
		  AND fi.is_filtered = 0
		  AND (fi.content_extraction_status IS NULL OR fi.content_extraction_status IN ('ready', 'failed'))
		  AND (CASE WHEN f.feed_type = 'youtube' THEN fi.media_status = 'ready'
		            ELSE fi.media_status IS NULL OR fi.media_status = 'ready' END)
		ORDER BY fi.published_at DESC
		LIMIT ?2
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
			JSONStringSlice(&item.Authors), JSONStringSlice(&item.Categories),
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
		       fi.published_at, fi.updated_at, COALESCE(fi.authors, '[]'),
		       COALESCE(fi.categories, '[]'),
		       fi.is_filtered,
		       fi.content_hash, fi.created_at,
		       COALESCE(fi.enclosure_url, ''), COALESCE(fi.enclosure_length, 0), COALESCE(fi.enclosure_type, ''),
		       COALESCE(fi.itunes_duration, 0), COALESCE(fi.itunes_episode, 0), COALESCE(fi.itunes_season, 0), COALESCE(fi.itunes_episode_type, ''), COALESCE(fi.itunes_image, ''),
		       fi.content_extraction_status,
		       fi.media_status, COALESCE(fi.media_path, ''), COALESCE(fi.media_size, 0)
		FROM feed_items fi
		WHERE fi.id = ?1
	`, itemID).Scan(
		&item.ID, &item.GUID, &item.Link, &item.Title,
		&item.Description, &item.Content, &item.PublishedAt, &item.UpdatedAt,
		JSONStringSlice(&item.Authors), JSONStringSlice(&item.Categories),
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

func (r *ItemRepository) UpdateMediaStatus(itemID, status, mediaPath string, mediaSize int64, duration int) error {
	_, err := r.db.Exec(`
		UPDATE feed_items
		SET media_status = ?2, media_path = ?3, media_size = ?4,
			itunes_duration = CASE WHEN ?5 > 0 THEN ?5 ELSE itunes_duration END
		WHERE id = ?1
	`, itemID, status, mediaPath, mediaSize, duration)

	if err != nil {
		return fmt.Errorf("failed to update media status: %w", err)
	}

	return nil
}

type MediaInfo struct {
	MediaPath      string
	MediaSize      int64
	ITunesDuration int
}

func (r *ItemRepository) GetReadyMediaByPath(mediaPath string) (*MediaInfo, error) {
	var info MediaInfo
	err := r.db.QueryRow(`
		SELECT media_path, media_size, COALESCE(itunes_duration, 0) FROM feed_items
		WHERE media_path = ?1 AND media_status = 'ready'
		LIMIT 1
	`, mediaPath).Scan(&info.MediaPath, &info.MediaSize, &info.ITunesDuration)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ready media by path: %w", err)
	}

	return &info, nil
}

// activeMediaQuery selects the ready items whose files media cleanup keeps.
const activeMediaQuery = `
	SELECT sub.id, sub.media_path FROM (
		SELECT fi.id, fi.media_path,
		       ROW_NUMBER() OVER (PARTITION BY fi.feed_id ORDER BY fi.published_at DESC) AS rn,
		       COALESCE(CAST(json_extract(f.settings, '$.max_items') AS INTEGER), 50) AS max_items
		FROM feed_items fi
		JOIN feeds f ON fi.feed_id = f.id
		WHERE f.is_enabled = 1
		  AND fi.media_status = 'ready'
		  AND fi.media_path IS NOT NULL
		  AND fi.is_filtered = 0
	) sub
	WHERE sub.rn <= sub.max_items`

// ReconcileMedia returns the media paths cleanup must keep and resets every other ready
// item, so no row points at a file cleanup deletes. One transaction keeps the two in
// step: an item that turns ready afterwards only adds a file. RequeueMedia downloads
// reset items again if they become eligible.
func (r *ItemRepository) ReconcileMedia() ([]string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	paths, err := activeMediaPaths(tx)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`
		UPDATE feed_items SET media_status = NULL, media_path = '', media_size = 0
		WHERE media_status = 'ready' AND id NOT IN (SELECT id FROM (` + activeMediaQuery + `))
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to clear evicted media: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit media reconciliation: %w", err)
	}
	return paths, nil
}

// GetPendingMediaGUIDs returns the GUIDs of items whose media may be downloading.
func (r *ItemRepository) GetPendingMediaGUIDs() ([]string, error) {
	rows, err := r.db.Query(`SELECT guid FROM feed_items WHERE media_status = 'pending'`)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending media: %w", err)
	}
	defer rows.Close()

	var guids []string
	for rows.Next() {
		var guid string
		if err := rows.Scan(&guid); err != nil {
			return nil, fmt.Errorf("failed to scan pending media: %w", err)
		}
		guids = append(guids, guid)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating pending media: %w", err)
	}
	return guids, nil
}

func (r *ItemRepository) GetAllActiveMediaPaths() ([]string, error) {
	return activeMediaPaths(r.db)
}

func activeMediaPaths(q rowsQuerier) ([]string, error) {
	rows, err := q.Query(`SELECT DISTINCT media_path FROM (` + activeMediaQuery + `)`)
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

func (r *ItemRepository) UpdateItemPublishedAt(itemID string, publishedAt time.Time) error {
	_, err := r.db.Exec(`
		UPDATE feed_items SET published_at = ?2 WHERE id = ?1
	`, itemID, sqliteTime(publishedAt))

	if err != nil {
		return fmt.Errorf("failed to update item published_at: %w", err)
	}

	return nil
}

func (r *ItemRepository) UpdateContentExtractionStatus(itemID, status, content string) error {
	_, err := r.db.Exec(`
		UPDATE feed_items
		SET content_extraction_status = ?2, content = CASE WHEN ?3 = '' THEN content ELSE ?3 END
		WHERE id = ?1
	`, itemID, status, content)

	if err != nil {
		return fmt.Errorf("failed to update content extraction status: %w", err)
	}

	return nil
}
