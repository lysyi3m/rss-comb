package media

import (
	"os"
	"path/filepath"
)

func FileExists(mediaDir, mediaPath string) (int64, bool) {
	info, err := os.Stat(filepath.Join(mediaDir, mediaPath))
	if err != nil {
		return 0, false
	}
	return info.Size(), true
}
