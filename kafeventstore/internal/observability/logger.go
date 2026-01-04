package observability

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

// LoggingConfig contains logging configuration.
type LoggingConfig struct {
	Level  string
	Format string
	Output string
}

// NewLogger creates a new structured logger based on configuration.
func NewLogger(config LoggingConfig) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(config.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var output io.Writer
	switch strings.ToLower(config.Output) {
	case "stderr":
		output = os.Stderr
	case "stdout":
		output = os.Stdout
	default:
		output = os.Stdout
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch strings.ToLower(config.Format) {
	case "json":
		handler = slog.NewJSONHandler(output, opts)
	case "text":
		handler = slog.NewTextHandler(output, opts)
	default:
		handler = slog.NewJSONHandler(output, opts)
	}

	return slog.New(handler)
}
