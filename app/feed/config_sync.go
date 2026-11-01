package feed

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lysyi3m/rss-comb/app/database"
)

// DisableRemovedFeeds disables feeds whose config file is gone. A file that exists but
// fails to parse keeps its feed. An empty directory disables nothing, because it more
// likely means a missing mount than the removal of every feed.
func DisableRemovedFeeds(feedsDir string, feedRepo *database.FeedRepository) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(feedsDir, "*.yml"))
	if err != nil {
		return nil, fmt.Errorf("failed to find YAML files: %w", err)
	}
	if len(files) == 0 {
		return nil, nil
	}

	names := make([]string, len(files))
	for i, file := range files {
		names[i] = strings.TrimSuffix(filepath.Base(file), ".yml")
	}
	return feedRepo.DisableFeedsExcept(names)
}

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
