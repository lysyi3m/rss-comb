package database

import (
	"database/sql"
	"fmt"
	"time"
)

var _ FeedRepository = (*FeedRepositoryImpl)(nil)

type FeedRepositoryImpl struct {
	db *DB
}

func NewFeedRepository(db *DB) FeedRepository {
	return &FeedRepositoryImpl{db: db}
}

func (r *FeedRepositoryImpl) GetFeed(feedName string) (*Feed, error) {
	var feed Feed
	err := r.db.QueryRow(`
		SELECT id, name, feed_url, COALESCE(link, ''), title, COALESCE(description, ''), COALESCE(image_url, ''), COALESCE(language, ''),
		       last_fetched_at, next_fetch_at, feed_published_at, feed_updated_at, created_at, updated_at
		FROM feeds
		WHERE name = $1
	`, feedName).Scan(
		&feed.ID, &feed.Name, &feed.FeedURL, &feed.Link, &feed.Title, &feed.Description, &feed.ImageURL, &feed.Language,
		&feed.LastFetchedAt, &feed.NextFetchAt, &feed.FeedPublishedAt, &feed.FeedUpdatedAt,
		&feed.CreatedAt, &feed.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed by name: %w", err)
	}

	return &feed, nil
}

func (r *FeedRepositoryImpl) GetFeedCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM feeds").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get feed count: %w", err)
	}
	return count, nil
}

func (r *FeedRepositoryImpl) UpsertFeed(feedName, feedURL string) error {
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

func (r *FeedRepositoryImpl) GetFeedContentHash(feedName string) (*string, error) {
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

func (r *FeedRepositoryImpl) UpdateFeedMetadataWithHash(feedName string, title string, link string, description string, imageURL string, language string, feedPublishedAt *time.Time, feedUpdatedAt *time.Time, contentHash string, nextFetchAt time.Time) error {
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
