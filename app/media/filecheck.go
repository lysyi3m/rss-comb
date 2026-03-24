package media

import (
	"os"
	"path/filepath"
)

// FileExists checks if a media file exists and returns its size.
func FileExists(mediaDir, mediaPath string) (int64, bool) {
	info, err := os.Stat(filepath.Join(mediaDir, mediaPath))
	if err != nil {
		return 0, false
	}
	return info.Size(), true
}
