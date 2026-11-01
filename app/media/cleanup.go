package media

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func CleanupMedia(mediaDir string, keepPaths []string) (int, error) {
	keepSet := make(map[string]struct{}, len(keepPaths))
	for _, p := range keepPaths {
		keepSet[p] = struct{}{}
	}

	entries, err := os.ReadDir(mediaDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read media directory: %w", err)
	}

	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".mp3") {
			continue
		}
		if _, keep := keepSet[name]; keep {
			continue
		}

		fullPath := filepath.Join(mediaDir, name)
		if err := os.Remove(fullPath); err != nil {
			slog.Warn("Failed to delete orphaned media file", "path", fullPath, "error", err)
			continue
		}

		deleted++
	}

	cleanupIncoming(filepath.Join(mediaDir, incomingDir), keepSet)

	return deleted, nil
}

// cleanupIncoming removes download directories a crash left behind for items that no
// longer need them. keepSet holds the file of every item still downloading.
func cleanupIncoming(dir string, keepSet map[string]struct{}) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if _, keep := keepSet[entry.Name()+".mp3"]; keep {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		if err := os.RemoveAll(fullPath); err != nil {
			slog.Warn("Failed to delete abandoned download", "path", fullPath, "error", err)
		}
	}
}
