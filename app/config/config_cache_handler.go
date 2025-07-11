package config

import (
	"log"
	"sync"
)

// ConfigCacheHandler provides shared functionality for components that need
// to maintain an in-memory cache of feed configurations
type ConfigCacheHandler struct {
	configs      map[string]*FeedConfig
	configsMutex sync.RWMutex
	componentName string
}

// NewConfigCacheHandler creates a new config cache handler
func NewConfigCacheHandler(componentName string, initialConfigs map[string]*FeedConfig) *ConfigCacheHandler {
	// Create a copy of the initial configs to avoid sharing the same map
	configsCopy := make(map[string]*FeedConfig, len(initialConfigs))
	for k, v := range initialConfigs {
		configsCopy[k] = v
	}
	
	return &ConfigCacheHandler{
		configs:       configsCopy,
		componentName: componentName,
	}
}

// OnConfigUpdate implements the ConfigUpdateHandler interface
func (h *ConfigCacheHandler) OnConfigUpdate(filePath string, config *FeedConfig, isDelete bool) error {
	h.configsMutex.Lock()
	defer h.configsMutex.Unlock()
	
	if isDelete {
		delete(h.configs, filePath)
		log.Printf("%s removed configuration: %s (ID: %s)", h.componentName, filePath, config.Feed.ID)
	} else {
		h.configs[filePath] = config
		log.Printf("%s updated configuration: %s (ID: %s)", h.componentName, filePath, config.Feed.ID)
	}
	
	return nil
}

// GetConfig safely retrieves a configuration by file path
func (h *ConfigCacheHandler) GetConfig(configFile string) (*FeedConfig, bool) {
	h.configsMutex.RLock()
	defer h.configsMutex.RUnlock()
	config, ok := h.configs[configFile]
	return config, ok
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