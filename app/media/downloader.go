package media

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Validate checks that the configured yt-dlp command is available and working.
func Validate(ytdlpCmd string) error {
	parts := strings.Fields(ytdlpCmd)
	if len(parts) == 0 {
		return fmt.Errorf("YT_DLP_CMD is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := append(parts[1:], "--version")
	cmd := exec.CommandContext(ctx, parts[0], args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yt-dlp validation failed (cmd: %s): %w\nOutput: %s", ytdlpCmd, err, string(output))
	}

	return nil
}

// MediaFileID extracts a stable, filesystem-safe identifier from an item's GUID.
func MediaFileID(guid string) string {
	if after, ok := strings.CutPrefix(guid, "yt:video:"); ok {
		return after
	}
	hash := sha256.Sum256([]byte(guid))
	return fmt.Sprintf("%x", hash[:8])
}

// Download runs yt-dlp to extract audio from the given URL.
// Returns the relative filename and file size on success.
func Download(ctx context.Context, ytdlpCmd, mediaDir, url, fileID string) (string, int64, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	parts := strings.Fields(ytdlpCmd)
	if len(parts) == 0 {
		return "", 0, fmt.Errorf("YT_DLP_CMD is empty")
	}

	outputTemplate := fileID + ".%(ext)s"
	args := append(parts[1:],
		"--extract-audio", "--audio-format", "mp3", "--audio-quality", "64k",
		"--postprocessor-args", "-ac 1",
		"--no-playlist", "--no-progress",
		"-o", outputTemplate,
		url,
	)

	cmd := exec.CommandContext(downloadCtx, parts[0], args...)
	cmd.Dir = mediaDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("yt-dlp failed: %w\nOutput: %s", err, string(output))
	}

	mediaPath := fileID + ".mp3"
	fullPath := filepath.Join(mediaDir, mediaPath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", 0, fmt.Errorf("downloaded file not found at %s: %w", fullPath, err)
	}

	return mediaPath, info.Size(), nil
}
