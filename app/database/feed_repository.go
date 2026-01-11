package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/lysyi3m/rss-comb/app/types"
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
		       is_enabled, settings, filters, config_hash,
		       COALESCE(itunes_author, ''), COALESCE(itunes_image, ''), COALESCE(itunes_explicit, ''), COALESCE(itunes_owner_name, ''), COALESCE(itunes_owner_email, '')
		FROM feeds
		WHERE name = $1
	`, feedName).Scan(
		&feed.ID, &feed.Name, &feed.FeedURL, &feed.Link, &feed.Title, &feed.Description, &feed.ImageURL, &feed.Language,
		&feed.LastFetchedAt, &feed.NextFetchAt, &feed.FeedPublishedAt, &feed.FeedUpdatedAt,
		&feed.CreatedAt, &feed.UpdatedAt,
		&feed.IsEnabled, &feed.Settings, &feed.Filters, &feed.ConfigHash,
		&feed.ITunesAuthor, &feed.ITunesImage, &feed.ITunesExplicit, &feed.ITunesOwnerName, &feed.ITunesOwnerEmail,
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

func (r *FeedRepository) UpdateFeedMetadata(feedName string, metadata *types.Metadata, contentHash string, nextFetchAt time.Time) error {
	_, err := r.db.Exec(`
		UPDATE feeds
		SET title = $2, link = $3, description = $4, image_url = $5, language = $6, feed_published_at = $7, feed_updated_at = $8,
		    content_hash = $9, next_fetch_at = $10, last_fetched_at = NOW(), updated_at = NOW(),
		    itunes_author = $11, itunes_image = $12, itunes_explicit = $13, itunes_owner_name = $14, itunes_owner_email = $15
		WHERE name = $1
	`, feedName, metadata.Title, metadata.Link, metadata.Description, metadata.ImageURL, metadata.Language, metadata.FeedPublishedAt, metadata.FeedUpdatedAt, contentHash, nextFetchAt,
		metadata.ITunesAuthor, metadata.ITunesImage, metadata.ITunesExplicit, metadata.ITunesOwnerName, metadata.ITunesOwnerEmail)

	if err != nil {
		return fmt.Errorf("failed to update feed metadata: %w", err)
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
		  AND (next_fetch_at IS NULL OR next_fetch_at <= NOW())
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
