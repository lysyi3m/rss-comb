package config_watcher

import "github.com/lysyi3m/rss-comb/app/config"

// ConfigUpdateHandler defines the interface for components that need to be notified of config changes
type ConfigUpdateHandler interface {
	OnConfigUpdate(filePath string, config *config.FeedConfig, isDelete bool) error
}
