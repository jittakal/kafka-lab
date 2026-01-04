// Package storage implements storage writer implementations.
package storage

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jittakal/kafeventstore/internal/encoder"
	"github.com/jittakal/kafeventstore/pkg/event"
	"github.com/jittakal/kafeventstore/pkg/storage"
)

// Ensure implementation satisfies interface at compile time.
var _ storage.Writer = (*FileWriter)(nil)

// MetricsCollector defines metrics operations for storage.
type MetricsCollector interface {
	IncFilesWritten(topic string, partition int32, format string, status string)
	ObserveFileSize(topic string, partition int32, format string, size float64)
	ObserveStorageWriteDuration(topic string, partition int32, duration float64)
	IncStorageErrors(backend string, operation string)
}

// FileConfig contains local filesystem configuration.
type FileConfig struct {
	BasePath string
}

// FileWriter implements storage.Writer for local filesystem storage.
// It provides thread-safe file writing with support for multiple formats (Avro, Parquet)
// and compression options. Files are organized in a hierarchical directory structure.
type FileWriter struct {
	basePath       string
	encoderFactory *encoder.Factory
	logger         *slog.Logger
	metrics        MetricsCollector
	mu             sync.RWMutex
	fileSequence   int    // Sequence counter for files created in the same second
	lastTimestamp  string // Last timestamp used for filename generation
}

// NewFileWriter creates a new filesystem storage writer.
func NewFileWriter(
	config FileConfig,
	format event.FileFormat,
	compression string,
	logger *slog.Logger,
	metrics MetricsCollector,
) (*FileWriter, error) {
	// Ensure base path exists
	if err := os.MkdirAll(config.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	// Create encoder factory
	encoderFactory := encoder.NewFactory(format, compression)

	// Validate encoder can be created
	if _, err := encoderFactory.CreateEncoder(); err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	logger.Info("filesystem writer created",
		"base_path", config.BasePath,
		"format", format,
		"compression", compression,
	)

	return &FileWriter{
		basePath:       config.BasePath,
		encoderFactory: encoderFactory,
		logger:         logger,
		metrics:        metrics,
	}, nil
}

// Write writes records to the filesystem.
func (w *FileWriter) Write(
	ctx context.Context,
	records []event.Record,
	path string,
	format event.FileFormat,
) (int64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(records) == 0 {
		return 0, fmt.Errorf("no records to write")
	}

	startTime := time.Now()

	// Create encoder
	fileEncoder, err := w.encoderFactory.CreateEncoder()
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("file", "encoder_create")
		}
		return 0, fmt.Errorf("failed to create encoder: %w", err)
	}

	// Strip file:// protocol prefix if present
	cleanPath := path
	if strings.HasPrefix(path, "file://") {
		cleanPath = strings.TrimPrefix(path, "file://")
	}

	// Generate timestamped filename: events_YYYYMMDD_HHMMSS_NNN.{ext}
	now := time.Now()
	timestamp := now.Format("20060102_150405")

	// Increment sequence if same timestamp as last file
	if timestamp == w.lastTimestamp {
		w.fileSequence++
	} else {
		w.fileSequence = 1
		w.lastTimestamp = timestamp
	}

	filename := fmt.Sprintf("events_%s_%03d%s", timestamp, w.fileSequence, fileEncoder.FileExtension())

	// Convert relative path to absolute and add timestamped filename
	dir := filepath.Join(w.basePath, cleanPath)
	fullPath := filepath.Join(dir, filename)

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("file", "mkdir")
		}
		return 0, fmt.Errorf("failed to create directory: %w", err)
	}

	// Encode and write records
	stats, err := fileEncoder.Encode(fullPath, records)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("file", "encode")
		}
		return 0, fmt.Errorf("failed to encode records: %w", err)
	}

	duration := time.Since(startTime)

	w.logger.Info("wrote records to file",
		"path", fullPath,
		"record_count", stats.RecordCount,
		"file_size", stats.SizeBytes,
		"format", format,
		"total_duration_ms", duration.Milliseconds(),
	)

	// Update metrics
	if w.metrics != nil && len(records) > 0 {
		topic := records[0].Kafka.Topic
		partition := records[0].Kafka.Partition

		w.metrics.IncFilesWritten(topic, partition, string(format), "success")
		w.metrics.ObserveFileSize(topic, partition, string(format), float64(stats.SizeBytes))
		w.metrics.ObserveStorageWriteDuration(topic, partition, duration.Seconds())
	}

	return stats.SizeBytes, nil
}

// Close closes the writer.
func (w *FileWriter) Close() error {
	w.logger.Info("closing filesystem writer")
	return nil
}
