package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

type FeedRepository struct {
	db *DB
}

func NewFeedRepository(db *DB) *FeedRepository {
	return &FeedRepository{db: db}
}

func (r *FeedRepository) GetFeed(feedName string) (*Feed, error) {
	var feed Feed
	err := r.db.QueryRow(`
		SELECT id, name, feed_url, COALESCE(link, ''), COALESCE(title, ''), COALESCE(description, ''), COALESCE(image_url, ''), COALESCE(language, ''),
		       last_fetched_at, next_fetch_at, feed_published_at, feed_updated_at, created_at, updated_at,
		       is_enabled, settings, filters, config_hash
		FROM feeds
		WHERE name = $1
	`, feedName).Scan(
		&feed.ID, &feed.Name, &feed.FeedURL, &feed.Link, &feed.Title, &feed.Description, &feed.ImageURL, &feed.Language,
		&feed.LastFetchedAt, &feed.NextFetchAt, &feed.FeedPublishedAt, &feed.FeedUpdatedAt,
		&feed.CreatedAt, &feed.UpdatedAt,
		&feed.IsEnabled, &feed.Settings, &feed.Filters, &feed.ConfigHash,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed by name: %w", err)
	}

	return &feed, nil
}

func (r *FeedRepository) GetFeedCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM feeds").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get feed count: %w", err)
	}
	return count, nil
}

func (r *FeedRepository) UpsertFeed(feedName, feedURL string) error {
	_, err := r.db.Exec(`
		INSERT INTO feeds (name, feed_url)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET
			feed_url = EXCLUDED.feed_url,
			next_fetch_at = CASE
				WHEN feeds.feed_url != EXCLUDED.feed_url
				THEN NULL
				ELSE feeds.next_fetch_at
			END,
			updated_at = NOW()
	`, feedName, feedURL)

	if err != nil {
		return fmt.Errorf("failed to upsert feed config: %w", err)
	}

	return nil
}

func (r *FeedRepository) GetFeedContentHash(feedName string) (*string, error) {
	var contentHash *string
	err := r.db.QueryRow("SELECT content_hash FROM feeds WHERE name = $1", feedName).Scan(&contentHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get feed content hash: %w", err)
	}
	return contentHash, nil
}

func (r *FeedRepository) UpdateFeedMetadataWithHash(feedName string, title string, link string, description string, imageURL string, language string, feedPublishedAt *time.Time, feedUpdatedAt *time.Time, contentHash string, nextFetchAt time.Time) error {
	_, err := r.db.Exec(`
		UPDATE feeds
		SET title = $2, link = $3, description = $4, image_url = $5, language = $6, feed_published_at = $7, feed_updated_at = $8,
		    content_hash = $9, next_fetch_at = $10, last_fetched_at = NOW(), updated_at = NOW()
		WHERE name = $1
	`, feedName, title, link, description, imageURL, language, feedPublishedAt, feedUpdatedAt, contentHash, nextFetchAt)

	if err != nil {
		return fmt.Errorf("failed to update feed metadata with hash: %w", err)
	}

	return nil
}

func (r *FeedRepository) UpsertFeedConfig(feedName string, feedURL string, isEnabled bool, settings interface{}, filters interface{}, configHash string) error {
	var existingHash *string
	err := r.db.QueryRow("SELECT config_hash FROM feeds WHERE name = $1", feedName).Scan(&existingHash)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing config hash: %w", err)
	}

	if existingHash != nil && *existingHash == configHash {
		return nil
	}

	if existingHash == nil {
		slog.Info("New feed configuration", "feed", feedName)
	} else {
		slog.Info("Feed configuration updated", "feed", feedName)
	}

	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	filtersJSON, err := json.Marshal(filters)
	if err != nil {
		return fmt.Errorf("failed to marshal filters: %w", err)
	}

	_, err = r.db.Exec(`
		INSERT INTO feeds (name, feed_url, is_enabled, settings, filters, config_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name) DO UPDATE SET
			feed_url = EXCLUDED.feed_url,
			is_enabled = EXCLUDED.is_enabled,
			settings = EXCLUDED.settings,
			filters = EXCLUDED.filters,
			config_hash = EXCLUDED.config_hash,
			next_fetch_at = CASE
				WHEN feeds.feed_url != EXCLUDED.feed_url OR feeds.config_hash != EXCLUDED.config_hash
				THEN NULL
				ELSE feeds.next_fetch_at
			END,
			updated_at = NOW()
	`, feedName, feedURL, isEnabled, settingsJSON, filtersJSON, configHash)

	if err != nil {
		return fmt.Errorf("failed to upsert feed config: %w", err)
	}

	return nil
}

type FeedScheduleInfo struct {
	Name        string
	NextFetchAt *time.Time
}

func (r *FeedRepository) GetDueFeeds() ([]FeedScheduleInfo, error) {
	rows, err := r.db.Query(`
		SELECT name, next_fetch_at
		FROM feeds
		WHERE is_enabled = true
		  AND next_fetch_at <= NOW()
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get due feeds: %w", err)
	}
	defer rows.Close()

	var feeds []FeedScheduleInfo
	for rows.Next() {
		var feed FeedScheduleInfo
		err := rows.Scan(&feed.Name, &feed.NextFetchAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed schedule info: %w", err)
		}
		feeds = append(feeds, feed)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feeds: %w", err)
	}

	return feeds, nil
}
