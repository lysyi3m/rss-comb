package media

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func Update(ytdlpCmd string) error {
	parts := strings.Fields(ytdlpCmd)
	if len(parts) == 0 {
		return fmt.Errorf("YT_DLP_CMD is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	args := append(parts[1:], "-U")
	cmd := exec.CommandContext(ctx, parts[0], args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yt-dlp update failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

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

// MediaFileID extracts YouTube video ID from GUID, falls back to SHA-256 hash.
func MediaFileID(guid string) string {
	if after, ok := strings.CutPrefix(guid, "yt:video:"); ok {
		return after
	}
	hash := sha256.Sum256([]byte(guid))
	return fmt.Sprintf("%x", hash[:8])
}

type VideoInfo struct {
	LiveStatus string
	Duration   int
}

// GetVideoInfo returns video metadata (live status, duration) from yt-dlp.
func GetVideoInfo(ctx context.Context, ytdlpCmd, url string) (VideoInfo, error) {
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parts := strings.Fields(ytdlpCmd)
	if len(parts) == 0 {
		return VideoInfo{}, fmt.Errorf("YT_DLP_CMD is empty")
	}

	args := append(parts[1:], "--dump-json", "--no-playlist", url)
	cmd := exec.CommandContext(checkCtx, parts[0], args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return VideoInfo{}, fmt.Errorf("yt-dlp metadata check failed: %w\nOutput: %s", err, string(output))
	}

	var metadata struct {
		LiveStatus string  `json:"live_status"`
		Duration   float64 `json:"duration"`
	}
	if err := json.Unmarshal(output, &metadata); err != nil {
		return VideoInfo{}, fmt.Errorf("failed to parse yt-dlp metadata: %w", err)
	}

	return VideoInfo{
		LiveStatus: metadata.LiveStatus,
		Duration:   int(metadata.Duration),
	}, nil
}

func IsLiveOrUpcoming(info VideoInfo) bool {
	return info.LiveStatus == "is_live" || info.LiveStatus == "is_upcoming"
}

func Download(ctx context.Context, ytdlpCmd, ytdlpArgs, mediaDir, url, fileID string) (string, int64, error) {
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
		"--remote-components", "ejs:github",
		"-o", outputTemplate,
	)

	if ytdlpArgs != "" {
		args = append(args, strings.Fields(ytdlpArgs)...)
	}

	args = append(args, url)

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
