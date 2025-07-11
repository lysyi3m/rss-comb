package config_watcher

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lysyi3m/rss-comb/app/config"
	"github.com/lysyi3m/rss-comb/app/config_loader"
)

// ConfigWatcher watches the feeds directory for configuration file changes
type ConfigWatcher struct {
	feedsDir       string
	loader         *config_loader.Loader
	configs        map[string]*config.FeedConfig
	configsMutex   sync.RWMutex
	watcher        *fsnotify.Watcher
	debouncer      map[string]*time.Timer
	debounceMutex  sync.Mutex
	debounceDelay  time.Duration
	updateHandlers []ConfigUpdateHandler
}

// NewConfigWatcher creates a new configuration file watcher
func NewConfigWatcher(loader *config_loader.Loader, feedsDir string) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Load initial configurations
	configs, err := loader.LoadAll()
	if err != nil {
		watcher.Close()
		return nil, err
	}

	cw := &ConfigWatcher{
		feedsDir:       feedsDir,
		loader:         loader,
		configs:        configs,
		watcher:        watcher,
		debouncer:      make(map[string]*time.Timer),
		debounceDelay:  500 * time.Millisecond, // Wait 500ms after last change
		updateHandlers: make([]ConfigUpdateHandler, 0),
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
func (cw *ConfigWatcher) GetConfigs() map[string]*config.FeedConfig {
	cw.configsMutex.RLock()
	defer cw.configsMutex.RUnlock()

	// Create a copy of the configs map
	configsCopy := make(map[string]*config.FeedConfig, len(cw.configs))
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
		cw.reloadConfiguration(event.Name, event.Op)

		// Remove timer from map
		cw.debounceMutex.Lock()
		delete(cw.debouncer, event.Name)
		cw.debounceMutex.Unlock()
	})
}

// reloadConfiguration reloads a configuration file and notifies handlers
func (cw *ConfigWatcher) reloadConfiguration(filePath string, op fsnotify.Op) {
	relPath, _ := filepath.Rel(cw.feedsDir, filePath)

	// Handle file deletion and rename operations
	if op&fsnotify.Remove == fsnotify.Remove || op&fsnotify.Rename == fsnotify.Rename {
		cw.handleConfigDeletion(filePath, relPath)

		// For rename operations, also check if a new file was created at a different path
		// This is handled by the fsnotify create event, so we don't need to do anything special here
		return
	}

	// Handle file creation/modification with robust error handling
	log.Printf("Reloading configuration: %s", relPath)

	// Validate that the file exists before attempting to load
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Configuration file no longer exists, treating as deletion: %s", relPath)
			cw.handleConfigDeletion(filePath, relPath)
			return
		}
		log.Printf("Error accessing configuration file %s: %v", relPath, err)
		return
	}

	// Load the specific configuration file with comprehensive error handling
	newConfig, err := cw.loader.Load(filePath)
	if err != nil {
		log.Printf("Error reloading configuration %s: %v", relPath, err)
		log.Printf("Configuration file %s will be skipped until the error is resolved", relPath)
		return
	}

	// Update the configs map atomically
	cw.configsMutex.Lock()
	oldConfig, existed := cw.configs[filePath]
	cw.configs[filePath] = newConfig
	cw.configsMutex.Unlock()

	// Log what changed
	cw.logConfigChange(filePath, oldConfig, newConfig, existed)

	// Notify all registered handlers about the config update
	handlerErrors := make([]error, 0)
	for _, handler := range cw.updateHandlers {
		if err := handler.OnConfigUpdate(filePath, newConfig, false); err != nil {
			handlerErrors = append(handlerErrors, err)
			log.Printf("Error notifying config update handler: %v", err)
		}
	}

	// Report overall success or failure
	if len(handlerErrors) > 0 {
		log.Printf("Configuration reload completed with %d handler errors: %s", len(handlerErrors), relPath)
	} else {
		log.Printf("Configuration reload completed successfully: %s", relPath)
	}
}

// handleConfigDeletion handles configuration file deletion
func (cw *ConfigWatcher) handleConfigDeletion(filePath, relPath string) {
	log.Printf("Configuration file deleted: %s", relPath)

	// Remove from configs map
	cw.configsMutex.Lock()
	deletedConfig, existed := cw.configs[filePath]
	if existed {
		delete(cw.configs, filePath)
	}
	cw.configsMutex.Unlock()

	if !existed {
		log.Printf("Deleted configuration was not loaded: %s", relPath)
		return
	}

	log.Printf("Removed configuration: %s (ID: %s)", relPath, deletedConfig.Feed.ID)

	// Notify all registered handlers about the deletion
	handlerErrors := make([]error, 0)
	for _, handler := range cw.updateHandlers {
		if err := handler.OnConfigUpdate(filePath, deletedConfig, true); err != nil {
			handlerErrors = append(handlerErrors, err)
			log.Printf("Error notifying config deletion handler: %v", err)
		}
	}

	// Report overall success or failure
	if len(handlerErrors) > 0 {
		log.Printf("Configuration deletion handling completed with %d handler errors: %s", len(handlerErrors), relPath)
	} else {
		log.Printf("Configuration deletion handling completed successfully: %s", relPath)
	}
}

// logConfigChange logs what changed in a configuration
func (cw *ConfigWatcher) logConfigChange(filePath string, oldConfig, newConfig *config.FeedConfig, existed bool) {
	relPath, _ := filepath.Rel(cw.feedsDir, filePath)

	if !existed {
		log.Printf("Added configuration: %s (ID: %s)", relPath, newConfig.Feed.ID)
		return
	}

	// Check what changed
	if oldConfig.Feed.URL != newConfig.Feed.URL {
		log.Printf("Updated feed URL in %s: %s -> %s", relPath, oldConfig.Feed.URL, newConfig.Feed.URL)
	}
	if oldConfig.Settings.Enabled != newConfig.Settings.Enabled {
		status := "disabled"
		if newConfig.Settings.Enabled {
			status = "enabled"
		}
		log.Printf("Updated feed status in %s: %s", relPath, status)
	}
	if len(oldConfig.Filters) != len(newConfig.Filters) {
		log.Printf("Updated filters in %s: %d -> %d filters", relPath, len(oldConfig.Filters), len(newConfig.Filters))
	}
	if oldConfig.Settings.RefreshInterval != newConfig.Settings.RefreshInterval {
		log.Printf("Updated refresh interval in %s: %ds -> %ds", relPath, oldConfig.Settings.RefreshInterval, newConfig.Settings.RefreshInterval)
	}
	if oldConfig.Settings.MaxItems != newConfig.Settings.MaxItems {
		log.Printf("Updated max items in %s: %d -> %d", relPath, oldConfig.Settings.MaxItems, newConfig.Settings.MaxItems)
	}
}