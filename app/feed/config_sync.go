package feed

import (
	"context"
	"fmt"

	"github.com/lysyi3m/rss-comb/app/database"
)

func ConfigSync(
	ctx context.Context,
	feedsDir string,
	feedName string,
	feedRepo *database.FeedRepository,
) (*Config, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	config, hash, err := LoadConfig(feedsDir, feedName)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	err = feedRepo.UpsertFeedConfig(
		config.Name,
		config.URL,
		config.Title,
		config.Type,
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
