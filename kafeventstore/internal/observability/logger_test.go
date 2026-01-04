package observability

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name   string
		config LoggingConfig
	}{
		{
			name: "json format",
			config: LoggingConfig{
				Level:  "info",
				Format: "json",
			},
		},
		{
			name: "text format",
			config: LoggingConfig{
				Level:  "debug",
				Format: "text",
			},
		},
		{
			name: "default format",
			config: LoggingConfig{
				Level:  "warn",
				Format: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.config)
			if logger == nil {
				t.Fatal("NewLogger returned nil")
			}
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	var buf bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("test message", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Log output should contain 'test message', got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Log output should contain 'key=value', got: %s", output)
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level string
	}{
		{"debug"},
		{"info"},
		{"warn"},
		{"warning"},
		{"error"},
		{"invalid"}, // Should default to info
		{""},        // Should default to info
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			config := LoggingConfig{
				Level:  tt.level,
				Format: "json",
			}
			logger := NewLogger(config)
			if logger == nil {
				t.Errorf("NewLogger with level %q returned nil", tt.level)
			}
		})
	}
}

func TestLoggerJSONFormat(t *testing.T) {
	config := LoggingConfig{
		Level:  "info",
		Format: "json",
	}

	logger := NewLogger(config)

	// We can't directly test output without mocking, but we can verify logger is created
	if logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestLoggerTextFormat(t *testing.T) {
	config := LoggingConfig{
		Level:  "debug",
		Format: "text",
	}

	logger := NewLogger(config)

	if logger == nil {
		t.Error("expected non-nil logger")
	}
}

func TestLoggerLevelParsing(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug level", "debug"},
		{"info level", "info"},
		{"warn level", "warn"},
		{"warning level", "warning"},
		{"error level", "error"},
		{"invalid defaults to info", "invalid"},
		{"empty defaults to info", ""},
		{"uppercase", "DEBUG"},
		{"mixed case", "Info"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := LoggingConfig{
				Level:  tt.level,
				Format: "json",
			}
			logger := NewLogger(config)
			if logger == nil {
				t.Errorf("NewLogger with level %q should not return nil", tt.level)
			}
		})
	}
}

func TestLoggerWithAttributes(t *testing.T) {
	var buf bytes.Buffer

	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	logger := slog.New(handler)

	logger = logger.With("app", "test-app", "version", "1.0")
	logger.Info("startup", "port", 8080)

	output := buf.String()
	if !strings.Contains(output, "app=test-app") {
		t.Errorf("Should contain app attribute, got: %s", output)
	}
	if !strings.Contains(output, "version=1.0") {
		t.Errorf("Should contain version attribute, got: %s", output)
	}
	if !strings.Contains(output, "startup") {
		t.Errorf("Should contain message, got: %s", output)
	}
}
