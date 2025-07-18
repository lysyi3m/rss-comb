package database

import (
	"database/sql"
	"fmt"
	"time"
)

// Compile-time interface compliance check
var _ FeedRepository = (*FeedRepositoryImpl)(nil)

// FeedRepositoryImpl handles database operations for feeds
type FeedRepositoryImpl struct {
	db *DB
}

// NewFeedRepository creates a new feed repository
func NewFeedRepository(db *DB) FeedRepository {
	return &FeedRepositoryImpl{db: db}
}

// UpsertFeedWithChangeDetection inserts or updates a feed configuration with change detection
func (r *FeedRepositoryImpl) UpsertFeedWithChangeDetection(configFile, feedID, feedURL, feedTitle string) (string, bool, error) {
	// First try to get existing feed by feed_id
	existingFeed, err := r.GetFeedByID(feedID)
	if err != nil {
		return "", false, fmt.Errorf("failed to check existing feed: %w", err)
	}

	var dbID string
	var urlChanged bool
	if existingFeed != nil {
		// Check if URL has changed
		if existingFeed.FeedURL != feedURL {
			urlChanged = true
		}
		
		// Update existing feed
		err = r.db.QueryRow(`
			UPDATE feeds 
			SET config_file = $2, feed_url = $3, title = $4, updated_at = NOW()
			WHERE feed_id = $1
			RETURNING id
		`, feedID, configFile, feedURL, feedTitle).Scan(&dbID)
	} else {
		// Insert new feed
		err = r.db.QueryRow(`
			INSERT INTO feeds (config_file, feed_id, feed_url, title)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, configFile, feedID, feedURL, feedTitle).Scan(&dbID)
	}

	if err != nil {
		return "", false, fmt.Errorf("failed to upsert feed: %w", err)
	}

	return dbID, urlChanged, nil
}

// UpdateFeedMetadata updates feed metadata including published timestamp after successful parsing
func (r *FeedRepositoryImpl) UpdateFeedMetadata(feedID string, link string, imageURL string, language string, feedPublishedAt *time.Time) error {
	_, err := r.db.Exec(`
		UPDATE feeds
		SET link = $2, image_url = $3, language = $4, feed_published_at = $5, updated_at = NOW()
		WHERE id = $1
	`, feedID, link, imageURL, language, feedPublishedAt)

	if err != nil {
		return fmt.Errorf("failed to update feed metadata: %w", err)
	}

	return nil
}

// UpdateNextFetch updates the next fetch time for a feed
func (r *FeedRepositoryImpl) UpdateNextFetch(feedID string, nextFetch time.Time) error {
	_, err := r.db.Exec(`
		UPDATE feeds
		SET next_fetch_at = $2, last_fetched_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, feedID, nextFetch)

	if err != nil {
		return fmt.Errorf("failed to update next fetch time: %w", err)
	}

	return nil
}

// GetFeedsDueForRefresh returns feeds that need to be refreshed
func (r *FeedRepositoryImpl) GetFeedsDueForRefresh() ([]Feed, error) {
	rows, err := r.db.Query(`
		SELECT id, feed_id, config_file, feed_url, COALESCE(link, ''), title, COALESCE(image_url, ''), COALESCE(language, ''),
		       last_fetched_at, next_fetch_at, feed_published_at, is_enabled, created_at, updated_at
		FROM feeds
		WHERE is_enabled = true
		  AND (next_fetch_at IS NULL OR next_fetch_at <= NOW())
		ORDER BY COALESCE(next_fetch_at, '1970-01-01'::timestamp)
		LIMIT 50
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get feeds due for refresh: %w", err)
	}
	defer rows.Close()

	var feeds []Feed
	for rows.Next() {
		var feed Feed
		err := rows.Scan(
			&feed.ID, &feed.FeedID, &feed.ConfigFile, &feed.FeedURL, &feed.Link, &feed.Title, &feed.ImageURL, &feed.Language,
			&feed.LastFetchedAt, &feed.NextFetchAt, &feed.FeedPublishedAt, &feed.IsEnabled,
			&feed.CreatedAt, &feed.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed row: %w", err)
		}
		feeds = append(feeds, feed)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feed rows: %w", err)
	}

	return feeds, nil
}

// GetFeedByID retrieves a feed by its configuration feed ID
func (r *FeedRepositoryImpl) GetFeedByID(feedID string) (*Feed, error) {
	var feed Feed
	err := r.db.QueryRow(`
		SELECT id, feed_id, config_file, feed_url, COALESCE(link, ''), title, COALESCE(image_url, ''), COALESCE(language, ''),
		       last_fetched_at, next_fetch_at, feed_published_at, is_enabled, created_at, updated_at
		FROM feeds
		WHERE feed_id = $1
	`, feedID).Scan(
		&feed.ID, &feed.FeedID, &feed.ConfigFile, &feed.FeedURL, &feed.Link, &feed.Title, &feed.ImageURL, &feed.Language,
		&feed.LastFetchedAt, &feed.NextFetchAt, &feed.FeedPublishedAt, &feed.IsEnabled,
		&feed.CreatedAt, &feed.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed by ID: %w", err)
	}

	return &feed, nil
}

// SetFeedEnabled sets the enabled status of a feed
func (r *FeedRepositoryImpl) SetFeedEnabled(feedID string, enabled bool) error {
	_, err := r.db.Exec(`
		UPDATE feeds
		SET is_enabled = $2, updated_at = NOW()
		WHERE id = $1
	`, feedID, enabled)

	if err != nil {
		return fmt.Errorf("failed to set feed enabled status: %w", err)
	}

	return nil
}

// GetFeedCount returns the total number of feeds
func (r *FeedRepositoryImpl) GetFeedCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM feeds").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get feed count: %w", err)
	}
	return count, nil
}

// GetEnabledFeedCount returns the count of enabled feeds
func (r *FeedRepositoryImpl) GetEnabledFeedCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM feeds WHERE is_enabled = true").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get enabled feed count: %w", err)
	}
	return count, nil
}
