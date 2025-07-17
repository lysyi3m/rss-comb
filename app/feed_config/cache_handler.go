package feed_config

import (
	"log/slog"
	"sync"
)

// ConfigCacheHandler provides shared functionality for components that need
// to maintain an in-memory cache of feed configurations
type ConfigCacheHandler struct {
	configs      map[string]*FeedConfig
	configsMutex sync.RWMutex
}

// NewConfigCacheHandler creates a new config cache handler
func NewConfigCacheHandler(initialConfigs map[string]*FeedConfig) *ConfigCacheHandler {
	// Create a copy of the initial configs to avoid sharing the same map
	configsCopy := make(map[string]*FeedConfig, len(initialConfigs))
	for k, v := range initialConfigs {
		configsCopy[k] = v
	}

	return &ConfigCacheHandler{
		configs: configsCopy,
	}
}

// OnConfigUpdate updates the configuration cache
func (h *ConfigCacheHandler) OnConfigUpdate(filePath string, feedConfig *FeedConfig, isDelete bool) error {
	h.configsMutex.Lock()
	defer h.configsMutex.Unlock()

	if isDelete {
		delete(h.configs, filePath)
		slog.Info("Configuration removed", "file", filePath, "feed_id", feedConfig.Feed.ID)
	} else {
		h.configs[filePath] = feedConfig
	}

	return nil
}

// GetConfig safely retrieves a configuration by file path
func (h *ConfigCacheHandler) GetConfig(configFile string) (*FeedConfig, bool) {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()
	feedConfig, ok := h.configs[configFile]
	return feedConfig, ok
}

// GetAllConfigs safely retrieves all configurations
func (h *ConfigCacheHandler) GetAllConfigs() map[string]*FeedConfig {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()

	// Return a copy to avoid external modifications
	configsCopy := make(map[string]*FeedConfig, len(h.configs))
	for k, v := range h.configs {
		configsCopy[k] = v
	}
	return configsCopy
}

// GetConfigCount returns the number of loaded configurations
func (h *ConfigCacheHandler) GetConfigCount() int {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()
	return len(h.configs)
}

// GetConfigByFeedID finds a configuration by its feed ID
// Returns the configuration and whether it was found
func (h *ConfigCacheHandler) GetConfigByFeedID(feedID string) (*FeedConfig, bool) {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()
	
	for _, feedConfig := range h.configs {
		if feedConfig.Feed.ID == feedID {
			return feedConfig, true
		}
	}
	return nil, false
}

// GetConfigAndFileByFeedID finds a configuration and its file path by feed ID
// Returns the configuration, file path, and whether it was found
func (h *ConfigCacheHandler) GetConfigAndFileByFeedID(feedID string) (*FeedConfig, string, bool) {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()
	
	for file, feedConfig := range h.configs {
		if feedConfig.Feed.ID == feedID {
			return feedConfig, file, true
		}
	}
	return nil, "", false
}
