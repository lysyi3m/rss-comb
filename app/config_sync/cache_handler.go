package config_sync

import (
	"log"
	"sync"

	"github.com/lysyi3m/rss-comb/app/config"
)

// ConfigCacheHandler provides shared functionality for components that need
// to maintain an in-memory cache of feed configurations
type ConfigCacheHandler struct {
	configs       map[string]*config.FeedConfig
	configsMutex  sync.RWMutex
	componentName string
}

// NewConfigCacheHandler creates a new config cache handler
func NewConfigCacheHandler(componentName string, initialConfigs map[string]*config.FeedConfig) *ConfigCacheHandler {
	// Create a copy of the initial configs to avoid sharing the same map
	configsCopy := make(map[string]*config.FeedConfig, len(initialConfigs))
	for k, v := range initialConfigs {
		configsCopy[k] = v
	}

	return &ConfigCacheHandler{
		configs:       configsCopy,
		componentName: componentName,
	}
}

// OnConfigUpdate implements the ConfigUpdateHandler interface
func (h *ConfigCacheHandler) OnConfigUpdate(filePath string, cfg *config.FeedConfig, isDelete bool) error {
	h.configsMutex.Lock()
	defer h.configsMutex.Unlock()

	if isDelete {
		delete(h.configs, filePath)
		log.Printf("%s removed configuration: %s (ID: %s)", h.componentName, filePath, cfg.Feed.ID)
	} else {
		h.configs[filePath] = cfg
		log.Printf("%s updated configuration: %s (ID: %s)", h.componentName, filePath, cfg.Feed.ID)
	}

	return nil
}

// GetConfig safely retrieves a configuration by file path
func (h *ConfigCacheHandler) GetConfig(configFile string) (*config.FeedConfig, bool) {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()
	cfg, ok := h.configs[configFile]
	return cfg, ok
}

// GetAllConfigs safely retrieves all configurations
func (h *ConfigCacheHandler) GetAllConfigs() map[string]*config.FeedConfig {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()

	// Return a copy to avoid external modifications
	configsCopy := make(map[string]*config.FeedConfig, len(h.configs))
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
func (h *ConfigCacheHandler) GetConfigByFeedID(feedID string) (*config.FeedConfig, bool) {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()
	
	for _, cfg := range h.configs {
		if cfg.Feed.ID == feedID {
			return cfg, true
		}
	}
	return nil, false
}

// GetConfigAndFileByFeedID finds a configuration and its file path by feed ID
// Returns the configuration, file path, and whether it was found
func (h *ConfigCacheHandler) GetConfigAndFileByFeedID(feedID string) (*config.FeedConfig, string, bool) {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()
	
	for file, cfg := range h.configs {
		if cfg.Feed.ID == feedID {
			return cfg, file, true
		}
	}
	return nil, "", false
}