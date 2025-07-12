package logger

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

// Initialize sets up the global logger with appropriate configuration
func Initialize(debug bool) {
	var level slog.Level
	if debug {
		level = slog.LevelDebug
	} else {
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Simplify time format for better readability
			if a.Key == slog.TimeKey {
				return slog.String("time", a.Value.Time().Format("2006-01-02 15:04:05"))
			}
			return a
		},
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

// Feed-specific logging helpers for common patterns
func FeedProcessed(title string, newItems, duplicates, filtered int, duration string) {
	Logger.Info("Feed processed",
		"feed", title,
		"new", newItems,
		"duplicates", duplicates,
		"filtered", filtered,
		"duration", duration)
}

func FeedError(title, operation string, err error) {
	Logger.Error("Feed operation failed",
		"feed", title,
		"operation", operation,
		"error", err)
}

func TaskCompleted(taskType, feedID string, duration string) {
	Logger.Debug("Task completed",
		"type", taskType,
		"feed_id", feedID,
		"duration", duration)
}

func ConfigChange(action, file, feedID string) {
	Logger.Info("Configuration changed",
		"action", action,
		"file", file,
		"feed_id", feedID)
}