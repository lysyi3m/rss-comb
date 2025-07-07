package database

import (
	"database/sql"
	"fmt"
	"time"
)

// FeedRepository handles database operations for feeds
type FeedRepository struct {
	db *DB
}

// NewFeedRepository creates a new feed repository
func NewFeedRepository(db *DB) *FeedRepository {
	return &FeedRepository{db: db}
}

// UpsertFeed inserts or updates a feed configuration
func (r *FeedRepository) UpsertFeed(configFile, feedID, feedURL, feedName string) (string, error) {
	// First try to get existing feed by feed_id
	existingFeed, err := r.GetFeedByID(feedID)
	if err != nil {
		return "", fmt.Errorf("failed to check existing feed: %w", err)
	}

	var dbID string
	if existingFeed != nil {
		// Update existing feed
		err = r.db.QueryRow(`
			UPDATE feeds 
			SET config_file = $2, feed_url = $3, feed_name = $4, updated_at = NOW()
			WHERE feed_id = $1
			RETURNING id
		`, feedID, configFile, feedURL, feedName).Scan(&dbID)
	} else {
		// Insert new feed
		err = r.db.QueryRow(`
			INSERT INTO feeds (config_file, feed_id, feed_url, feed_name)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, configFile, feedID, feedURL, feedName).Scan(&dbID)
	}

	if err != nil {
		return "", fmt.Errorf("failed to upsert feed: %w", err)
	}

	return dbID, nil
}

// UpsertFeedWithChangeDetection inserts or updates a feed configuration with change detection
func (r *FeedRepository) UpsertFeedWithChangeDetection(configFile, feedID, feedURL, feedName string) (string, bool, error) {
	// First try to get existing feed by feed_id
	existingFeed, err := r.GetFeedByID(feedID)
	if err != nil {
		return "", false, fmt.Errorf("failed to check existing feed: %w", err)
	}

	var dbID string
	var urlChanged bool
	if existingFeed != nil {
		// Check if URL has changed
		if existingFeed.URL != feedURL {
			urlChanged = true
		}
		
		// Update existing feed
		err = r.db.QueryRow(`
			UPDATE feeds 
			SET config_file = $2, feed_url = $3, feed_name = $4, updated_at = NOW()
			WHERE feed_id = $1
			RETURNING id
		`, feedID, configFile, feedURL, feedName).Scan(&dbID)
	} else {
		// Insert new feed
		err = r.db.QueryRow(`
			INSERT INTO feeds (config_file, feed_id, feed_url, feed_name)
			VALUES ($1, $2, $3, $4)
			RETURNING id
		`, configFile, feedID, feedURL, feedName).Scan(&dbID)
	}

	if err != nil {
		return "", false, fmt.Errorf("failed to upsert feed: %w", err)
	}

	return dbID, urlChanged, nil
}

// UpdateFeedMetadata updates feed metadata after successful parsing
func (r *FeedRepository) UpdateFeedMetadata(feedID string, iconURL string, language string) error {
	_, err := r.db.Exec(`
		UPDATE feeds
		SET feed_icon_url = $2, language = $3, last_success = NOW(), updated_at = NOW()
		WHERE id = $1
	`, feedID, iconURL, language)

	if err != nil {
		return fmt.Errorf("failed to update feed metadata: %w", err)
	}

	return nil
}

// UpdateNextFetch updates the next fetch time for a feed
func (r *FeedRepository) UpdateNextFetch(feedID string, nextFetch time.Time) error {
	_, err := r.db.Exec(`
		UPDATE feeds
		SET next_fetch = $2, last_fetched = NOW(), updated_at = NOW()
		WHERE id = $1
	`, feedID, nextFetch)

	if err != nil {
		return fmt.Errorf("failed to update next fetch time: %w", err)
	}

	return nil
}

// GetFeedsDueForRefresh returns feeds that need to be refreshed
func (r *FeedRepository) GetFeedsDueForRefresh() ([]Feed, error) {
	rows, err := r.db.Query(`
		SELECT id, feed_id, config_file, feed_url, feed_name, COALESCE(feed_icon_url, ''), COALESCE(language, ''),
		       last_fetched, last_success, next_fetch, enabled, created_at, updated_at
		FROM feeds
		WHERE enabled = true
		  AND (next_fetch IS NULL OR next_fetch <= NOW())
		ORDER BY COALESCE(next_fetch, '1970-01-01'::timestamp)
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
			&feed.ID, &feed.FeedID, &feed.ConfigFile, &feed.URL, &feed.Name, &feed.IconURL, &feed.Language,
			&feed.LastFetched, &feed.LastSuccess, &feed.NextFetch, &feed.Enabled,
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

// GetFeedByConfigFile retrieves a feed by its configuration file path
func (r *FeedRepository) GetFeedByConfigFile(configFile string) (*Feed, error) {
	var feed Feed
	err := r.db.QueryRow(`
		SELECT id, feed_id, config_file, feed_url, feed_name, COALESCE(feed_icon_url, ''), COALESCE(language, ''),
		       last_fetched, last_success, next_fetch, enabled, created_at, updated_at
		FROM feeds
		WHERE config_file = $1
	`, configFile).Scan(
		&feed.ID, &feed.FeedID, &feed.ConfigFile, &feed.URL, &feed.Name, &feed.IconURL, &feed.Language,
		&feed.LastFetched, &feed.LastSuccess, &feed.NextFetch, &feed.Enabled,
		&feed.CreatedAt, &feed.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed by config file: %w", err)
	}

	return &feed, nil
}

// GetFeedByURL retrieves a feed by its URL
func (r *FeedRepository) GetFeedByURL(feedURL string) (*Feed, error) {
	var feed Feed
	err := r.db.QueryRow(`
		SELECT id, feed_id, config_file, feed_url, feed_name, COALESCE(feed_icon_url, ''), COALESCE(language, ''),
		       last_fetched, last_success, next_fetch, enabled, created_at, updated_at
		FROM feeds
		WHERE feed_url = $1
	`, feedURL).Scan(
		&feed.ID, &feed.FeedID, &feed.ConfigFile, &feed.URL, &feed.Name, &feed.IconURL, &feed.Language,
		&feed.LastFetched, &feed.LastSuccess, &feed.NextFetch, &feed.Enabled,
		&feed.CreatedAt, &feed.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed by URL: %w", err)
	}

	return &feed, nil
}

// GetFeedByID retrieves a feed by its configuration feed ID
func (r *FeedRepository) GetFeedByID(feedID string) (*Feed, error) {
	var feed Feed
	err := r.db.QueryRow(`
		SELECT id, feed_id, config_file, feed_url, feed_name, COALESCE(feed_icon_url, ''), COALESCE(language, ''),
		       last_fetched, last_success, next_fetch, enabled, created_at, updated_at
		FROM feeds
		WHERE feed_id = $1
	`, feedID).Scan(
		&feed.ID, &feed.FeedID, &feed.ConfigFile, &feed.URL, &feed.Name, &feed.IconURL, &feed.Language,
		&feed.LastFetched, &feed.LastSuccess, &feed.NextFetch, &feed.Enabled,
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

// SetFeedActive sets the active status of a feed
func (r *FeedRepository) SetFeedActive(feedID string, active bool) error {
	_, err := r.db.Exec(`
		UPDATE feeds
		SET enabled = $2, updated_at = NOW()
		WHERE id = $1
	`, feedID, active)

	if err != nil {
		return fmt.Errorf("failed to set feed active status: %w", err)
	}

	return nil
}

// GetFeedCount returns the total number of feeds
func (r *FeedRepository) GetFeedCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM feeds").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get feed count: %w", err)
	}
	return count, nil
}

// GetActiveFeedCount returns the count of enabled feeds
func (r *FeedRepository) GetActiveFeedCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM feeds WHERE enabled = true").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get active feed count: %w", err)
	}
	return count, nil
}