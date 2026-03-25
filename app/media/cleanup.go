package media

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// CleanupMedia deletes media files in mediaDir that are not in the keepPaths set.
// Returns the number of files deleted.
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

		slog.Info("Deleted orphaned media file", "path", name)
		deleted++
	}

	return deleted, nil
}
