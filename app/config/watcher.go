package config

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigWatcher watches the feeds directory for configuration file changes
type ConfigWatcher struct {
	feedsDir     string
	loader       *Loader
	configs      map[string]*FeedConfig
	configsMutex sync.RWMutex
	watcher      *fsnotify.Watcher
	debouncer    map[string]*time.Timer
	debounceMutex sync.Mutex
	debounceDelay time.Duration
	updateHandlers []ConfigUpdateHandler
}

// ConfigUpdateHandler defines the interface for components that need to be notified of config changes
type ConfigUpdateHandler interface {
	OnConfigUpdate(configs map[string]*FeedConfig) error
}

// NewConfigWatcher creates a new configuration file watcher
func NewConfigWatcher(feedsDir string) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	loader := NewLoader(feedsDir)
	
	// Load initial configurations
	configs, err := loader.LoadAll()
	if err != nil {
		watcher.Close()
		return nil, err
	}

	cw := &ConfigWatcher{
		feedsDir:       feedsDir,
		loader:        loader,
		configs:       configs,
		watcher:       watcher,
		debouncer:     make(map[string]*time.Timer),
		debounceDelay: 500 * time.Millisecond, // Wait 500ms after last change
	}

	// Add the feeds directory to the watcher
	err = watcher.Add(feedsDir)
	if err != nil {
		watcher.Close()
		return nil, err
	}

	log.Printf("ConfigWatcher initialized, monitoring %s", feedsDir)
	return cw, nil
}

// AddUpdateHandler registers a handler to be called when configurations are updated
func (cw *ConfigWatcher) AddUpdateHandler(handler ConfigUpdateHandler) {
	cw.updateHandlers = append(cw.updateHandlers, handler)
}

// GetConfigs returns a thread-safe copy of the current configurations
func (cw *ConfigWatcher) GetConfigs() map[string]*FeedConfig {
	cw.configsMutex.RLock()
	defer cw.configsMutex.RUnlock()
	
	// Create a copy of the configs map
	configsCopy := make(map[string]*FeedConfig, len(cw.configs))
	for k, v := range cw.configs {
		configsCopy[k] = v
	}
	return configsCopy
}

// Start begins watching for file system changes
func (cw *ConfigWatcher) Start(ctx context.Context) error {
	log.Printf("Starting configuration watcher for directory: %s", cw.feedsDir)
	
	for {
		select {
		case <-ctx.Done():
			log.Printf("ConfigWatcher stopping...")
			return ctx.Err()
			
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return nil
			}
			cw.handleFileEvent(event)
			
		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("ConfigWatcher error: %v", err)
		}
	}
}

// Stop closes the file watcher
func (cw *ConfigWatcher) Stop() error {
	log.Printf("Stopping configuration watcher...")
	
	// Cancel any pending debounce timers
	cw.debounceMutex.Lock()
	for _, timer := range cw.debouncer {
		timer.Stop()
	}
	cw.debouncer = make(map[string]*time.Timer)
	cw.debounceMutex.Unlock()
	
	return cw.watcher.Close()
}

// handleFileEvent processes file system events with debouncing
func (cw *ConfigWatcher) handleFileEvent(event fsnotify.Event) {
	// Only process .yml files
	if !strings.HasSuffix(event.Name, ".yml") {
		return
	}
	
	// Get relative path for logging
	relPath, _ := filepath.Rel(cw.feedsDir, event.Name)
	
	log.Printf("Config file event: %s -> %s", event.Op.String(), relPath)
	
	// Debounce rapid changes to the same file
	cw.debounceMutex.Lock()
	defer cw.debounceMutex.Unlock()
	
	// Cancel existing timer for this file
	if timer, exists := cw.debouncer[event.Name]; exists {
		timer.Stop()
	}
	
	// Set new timer for this file
	cw.debouncer[event.Name] = time.AfterFunc(cw.debounceDelay, func() {
		cw.reloadConfigurations()
		
		// Remove timer from map
		cw.debounceMutex.Lock()
		delete(cw.debouncer, event.Name)
		cw.debounceMutex.Unlock()
	})
}

// reloadConfigurations reloads all configuration files and notifies handlers
func (cw *ConfigWatcher) reloadConfigurations() {
	log.Printf("Reloading configurations...")
	
	// Load fresh configurations from disk
	newConfigs, err := cw.loader.LoadAll()
	if err != nil {
		log.Printf("Error reloading configurations: %v", err)
		return
	}
	
	// Update the configs map atomically
	cw.configsMutex.Lock()
	oldConfigs := cw.configs
	cw.configs = newConfigs
	cw.configsMutex.Unlock()
	
	// Log changes
	cw.logConfigChanges(oldConfigs, newConfigs)
	
	// Notify all registered handlers
	for _, handler := range cw.updateHandlers {
		if err := handler.OnConfigUpdate(newConfigs); err != nil {
			log.Printf("Error notifying config update handler: %v", err)
		}
	}
	
	log.Printf("Configuration reload completed. %d configurations loaded", len(newConfigs))
}

// logConfigChanges logs what configurations have changed
func (cw *ConfigWatcher) logConfigChanges(oldConfigs, newConfigs map[string]*FeedConfig) {
	// Find added configs
	for file, config := range newConfigs {
		if _, exists := oldConfigs[file]; !exists {
			log.Printf("Added configuration: %s (ID: %s)", file, config.Feed.ID)
		}
	}
	
	// Find removed configs
	for file, config := range oldConfigs {
		if _, exists := newConfigs[file]; !exists {
			log.Printf("Removed configuration: %s (ID: %s)", file, config.Feed.ID)
		}
	}
	
	// Find modified configs (simplified check - could be more sophisticated)
	for file, newConfig := range newConfigs {
		if oldConfig, exists := oldConfigs[file]; exists {
			// Check if URL changed (most common change)
			if oldConfig.Feed.URL != newConfig.Feed.URL {
				log.Printf("Updated feed URL in %s: %s -> %s", file, oldConfig.Feed.URL, newConfig.Feed.URL)
			}
			// Check if enabled status changed
			if oldConfig.Settings.Enabled != newConfig.Settings.Enabled {
				status := "disabled"
				if newConfig.Settings.Enabled {
					status = "enabled"
				}
				log.Printf("Updated feed status in %s: %s", file, status)
			}
			// Check if filters changed
			if len(oldConfig.Filters) != len(newConfig.Filters) {
				log.Printf("Updated filters in %s: %d -> %d filters", file, len(oldConfig.Filters), len(newConfig.Filters))
			}
		}
	}
}