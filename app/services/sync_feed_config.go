package services

import (
	"context"
	"fmt"

	"github.com/lysyi3m/rss-comb/app/database"
	"github.com/lysyi3m/rss-comb/app/feed"
)

func SyncFeedConfig(
	ctx context.Context,
	feedsDir string,
	feedName string,
	feedRepo *database.FeedRepository,
) (*feed.Config, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	config, hash, err := feed.LoadConfig(feedsDir, feedName)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	err = feedRepo.UpsertFeedConfig(
		config.Name,
		config.URL,
		config.Title,
		config.Enabled,
		config.Settings,
		config.Filters,
		hash,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert config to database: %w", err)
	}

	return config, nil
}
