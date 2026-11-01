package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// incomingDir, inside the media directory, holds downloads in progress.
const incomingDir = ".incoming"

// DownloadTimeout bounds a single yt-dlp download.
const DownloadTimeout = 30 * time.Minute

// subprocessWaitDelay bounds the wait after a kill, for a descendant that left the
// process group but still holds the output pipes.
const subprocessWaitDelay = 5 * time.Second

// newCommand runs the command in its own process group and cancels the whole group.
// Otherwise a child such as ffmpeg outlives yt-dlp and keeps writing the output
// file while the job retries.
func newCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = subprocessWaitDelay
	return cmd
}

func Update(ytdlpCmd string) error {
	parts := strings.Fields(ytdlpCmd)
	if len(parts) == 0 {
		return fmt.Errorf("YT_DLP_CMD is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// --update-to nightly rather than -U: stable lags too far behind YouTube's
	// extraction changes (see the Dockerfile). Also migrates a stable-channel
	// binary onto nightly; no-op once there.
	args := append(parts[1:], "--update-to", "nightly")
	cmd := newCommand(ctx, parts[0], args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yt-dlp update failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// Validate returns the yt-dlp version, so a stale version stays visible in the
// startup log even when the update failed.
func Validate(ytdlpCmd string) (string, error) {
	parts := strings.Fields(ytdlpCmd)
	if len(parts) == 0 {
		return "", fmt.Errorf("YT_DLP_CMD is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	args := append(parts[1:], "--version")
	cmd := newCommand(ctx, parts[0], args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("yt-dlp validation failed (cmd: %s): %w\nOutput: %s", ytdlpCmd, err, string(output))
	}

	return strings.TrimSpace(string(output)), nil
}

// safeFileID matches a YouTube video ID. The GUID comes from a remote feed and the
// result names files and directories, so it must not be able to carry a path.
var safeFileID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// MediaFileID extracts YouTube video ID from GUID, falls back to SHA-256 hash.
func MediaFileID(guid string) string {
	if after, ok := strings.CutPrefix(guid, "yt:video:"); ok && safeFileID.MatchString(after) {
		return after
	}
	hash := sha256.Sum256([]byte(guid))
	return fmt.Sprintf("%x", hash[:8])
}

type VideoInfo struct {
	LiveStatus       string
	Duration         int
	ReleaseTimestamp  int64
	UploadTimestamp   int64
}

// GetVideoInfo returns video metadata (live status, duration, release time, upload timestamp) from yt-dlp.
func GetVideoInfo(ctx context.Context, ytdlpCmd, ytdlpArgs, url string) (VideoInfo, error) {
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parts := strings.Fields(ytdlpCmd)
	if len(parts) == 0 {
		return VideoInfo{}, fmt.Errorf("YT_DLP_CMD is empty")
	}

	// The fixed flags come last so that download options in YT_DLP_ARGS, such as
	// --no-simulate, cannot turn this metadata call into a download.
	args := append(parts[1:], strings.Fields(ytdlpArgs)...)
	args = append(args, "--simulate", "--dump-json", "--no-playlist", "--ignore-no-formats-error", url)
	cmd := newCommand(checkCtx, parts[0], args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return VideoInfo{}, fmt.Errorf("yt-dlp metadata check failed: %w\nOutput: %s", err, stderr.String())
	}

	type videoMetadata struct {
		ID               string  `json:"id"`
		LiveStatus       string  `json:"live_status"`
		Duration         float64 `json:"duration"`
		ReleaseTimestamp *int64  `json:"release_timestamp"`
		Timestamp        *int64  `json:"timestamp"`
	}
	// YT_DLP_ARGS can add other stdout lines, such as --print output, so look for
	// the line that is a video record.
	var metadata videoMetadata
	for _, line := range bytes.Split(stdout.Bytes(), []byte("\n")) {
		var candidate videoMetadata
		if json.Unmarshal(line, &candidate) == nil && candidate.ID != "" {
			metadata = candidate
		}
	}
	if metadata.ID == "" {
		return VideoInfo{}, fmt.Errorf("no video metadata in yt-dlp output")
	}

	info := VideoInfo{
		LiveStatus: metadata.LiveStatus,
		Duration:   int(metadata.Duration),
	}
	if metadata.ReleaseTimestamp != nil {
		info.ReleaseTimestamp = *metadata.ReleaseTimestamp
	}
	if metadata.Timestamp != nil {
		info.UploadTimestamp = *metadata.Timestamp
	}

	return info, nil
}

func Download(ctx context.Context, ytdlpCmd, ytdlpArgs, mediaDir, url, fileID string) (string, int64, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, DownloadTimeout)
	defer cancel()

	parts := strings.Fields(ytdlpCmd)
	if len(parts) == 0 {
		return "", 0, fmt.Errorf("YT_DLP_CMD is empty")
	}

	// Work in a private directory and publish by rename, so an interrupted download
	// or conversion never leaves a partial file under the final name.
	workDir := filepath.Join(incomingDir, fileID)
	fullWorkDir := filepath.Join(mediaDir, workDir)
	if err := os.RemoveAll(fullWorkDir); err != nil {
		return "", 0, fmt.Errorf("failed to clear download directory: %w", err)
	}
	defer os.RemoveAll(fullWorkDir)

	// The template is relative to mediaDir, which is also the dev container's cwd.
	outputTemplate := filepath.Join(workDir, fileID+".%(ext)s")
	args := append(parts[1:],
		"--format", "bestaudio[protocol!=m3u8_native][protocol!=m3u8]",
		"--extract-audio", "--audio-format", "mp3", "--audio-quality", "64k",
		"--postprocessor-args", "ffmpeg:-ac 1",
		"--no-playlist", "--no-progress",
		"--remote-components", "ejs:github",
		"-o", outputTemplate,
	)

	if ytdlpArgs != "" {
		args = append(args, strings.Fields(ytdlpArgs)...)
	}

	args = append(args, url)

	cmd := newCommand(downloadCtx, parts[0], args...)
	cmd.Dir = mediaDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("yt-dlp failed: %w\nOutput: %s", err, string(output))
	}

	mediaPath := fileID + ".mp3"
	fullPath := filepath.Join(mediaDir, mediaPath)
	if err := os.Rename(filepath.Join(fullWorkDir, mediaPath), fullPath); err != nil {
		return "", 0, fmt.Errorf("failed to publish downloaded file: %w", err)
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", 0, fmt.Errorf("downloaded file not found at %s: %w", fullPath, err)
	}

	return mediaPath, info.Size(), nil
}

// GetAudioDuration returns the duration in seconds of an audio file using ffprobe.
func GetAudioDuration(mediaDir, filename string) (int, error) {
	fullPath := filepath.Join(mediaDir, filename)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := newCommand(ctx, "ffprobe",
		"-i", fullPath,
		"-show_entries", "format=duration",
		"-v", "quiet",
		"-of", "csv=p=0",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w\nOutput: %s", err, stderr.String())
	}

	durationStr := strings.TrimSpace(stdout.String())
	if durationStr == "" || durationStr == "N/A" {
		return 0, fmt.Errorf("ffprobe returned no duration")
	}

	var duration float64
	if _, err := fmt.Sscanf(durationStr, "%f", &duration); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe duration %q: %w", durationStr, err)
	}

	return int(duration), nil
}
