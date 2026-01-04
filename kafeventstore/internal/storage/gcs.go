// Package storage implements Google Cloud Storage writer.
package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/jittakal/kafeventstore/internal/encoder"
	"github.com/jittakal/kafeventstore/pkg/event"
	pkgstorage "github.com/jittakal/kafeventstore/pkg/storage"
)

// Ensure implementation satisfies interface at compile time.
var _ pkgstorage.Writer = (*GCSWriter)(nil)

// GCSConfig contains Google Cloud Storage configuration.
type GCSConfig struct {
	Bucket               string
	ProjectID            string
	CredentialsFile      string
	CredentialsJSON      string
	Endpoint             string
	UseDefaultCredential bool
}

// GCSWriter implements storage.Writer for Google Cloud Storage.
// It supports multiple authentication methods (service account file, JSON, default credentials),
// with automatic blob creation and hierarchical path organization.
type GCSWriter struct {
	client         *storage.Client
	bucket         string
	encoderFactory *encoder.Factory
	logger         *slog.Logger
	metrics        MetricsCollector
	mu             sync.RWMutex
}

// NewGCSWriter creates a new Google Cloud Storage writer.
func NewGCSWriter(
	cfg GCSConfig,
	format event.FileFormat,
	compression string,
	logger *slog.Logger,
	metrics MetricsCollector,
) (*GCSWriter, error) {
	ctx := context.Background()

	// Determine authentication method
	var clientOpts []option.ClientOption
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, option.WithEndpoint(cfg.Endpoint))
	}

	if cfg.UseDefaultCredential {
		// Use default credentials (ADC)
		// This will use GOOGLE_APPLICATION_CREDENTIALS env var or default service account
		logger.Info("using default GCP credentials")
	} else if cfg.CredentialsJSON != "" {
		// Use credentials from JSON string
		clientOpts = append(clientOpts, option.WithCredentialsJSON([]byte(cfg.CredentialsJSON)))
		logger.Info("using GCP credentials from JSON string")
	} else if cfg.CredentialsFile != "" {
		// Use credentials from file
		clientOpts = append(clientOpts, option.WithCredentialsFile(cfg.CredentialsFile))
		logger.Info("using GCP credentials from file", "file", cfg.CredentialsFile)
	} else {
		// Fallback to default credentials
		logger.Info("no explicit credentials provided, using default GCP credentials")
	}

	// Create GCS client
	client, err := storage.NewClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	// Create encoder factory
	encoderFactory := encoder.NewFactory(format, compression)

	// Validate encoder can be created
	if _, err := encoderFactory.CreateEncoder(); err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	logger.Info("GCS writer created",
		"bucket", cfg.Bucket,
		"project_id", cfg.ProjectID,
		"format", format,
		"compression", compression,
	)

	return &GCSWriter{
		client:         client,
		bucket:         cfg.Bucket,
		encoderFactory: encoderFactory,
		logger:         logger,
		metrics:        metrics,
	}, nil
}

// Write writes records to Google Cloud Storage.
func (w *GCSWriter) Write(
	ctx context.Context,
	records []event.Record,
	path string,
	format event.FileFormat,
) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	startTime := time.Now()

	// Create encoder
	enc, err := w.encoderFactory.CreateEncoder()
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("gcs", "encoder_create")
		}
		return 0, fmt.Errorf("failed to create encoder: %w", err)
	}

	// Parse GCS URI to extract object path
	// Path format: gs://bucket/object/path or just object/path
	objectPath := path
	if strings.HasPrefix(path, "gs://") {
		// Remove gs:// prefix and bucket
		pathWithoutProtocol := strings.TrimPrefix(path, "gs://")
		parts := strings.SplitN(pathWithoutProtocol, "/", 2)
		if len(parts) == 2 {
			objectPath = parts[1]
		} else {
			objectPath = ""
		}
	}

	// Generate timestamped filename: events_YYYYMMDD_HHMMSS_NNN.{ext}
	now := time.Now()
	timestamp := now.Format("20060102_150405")
	filename := fmt.Sprintf("events_%s_%03d%s", timestamp, now.Nanosecond()/1000000, enc.FileExtension())
	objectPath = objectPath + filename
	objectPath = strings.TrimPrefix(objectPath, "/")

	// Encode to temporary file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("gcs-upload-%d%s", now.UnixNano(), enc.FileExtension()))

	stats, err := enc.Encode(tempFile, records)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("gcs", "encode")
		}
		return 0, fmt.Errorf("failed to encode records: %w", err)
	}
	defer os.Remove(tempFile)

	// Open the file for upload
	file, err := os.Open(tempFile)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("gcs", "file_open")
		}
		return 0, fmt.Errorf("failed to open encoded file: %w", err)
	}
	defer file.Close()

	// Create GCS object writer
	obj := w.client.Bucket(w.bucket).Object(objectPath)
	gcsWriter := obj.NewWriter(ctx)

	// Set content type based on format
	switch format {
	case event.FormatParquet:
		gcsWriter.ContentType = "application/octet-stream"
	case event.FormatAvro:
		gcsWriter.ContentType = "application/avro"
	default:
		gcsWriter.ContentType = "application/octet-stream"
	}

	// Copy file to GCS
	bytesWritten, err := io.Copy(gcsWriter, file)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("gcs", "upload")
		}
		gcsWriter.Close()
		return 0, fmt.Errorf("failed to write to GCS: %w", err)
	}

	// Close the writer to finalize the upload
	if err := gcsWriter.Close(); err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("gcs", "close")
		}
		return 0, fmt.Errorf("failed to close GCS writer: %w", err)
	}

	duration := time.Since(startTime)

	w.logger.Info("wrote records to GCS",
		"bucket", w.bucket,
		"object", objectPath,
		"record_count", stats.RecordCount,
		"file_size", stats.SizeBytes,
		"bytes_written", bytesWritten,
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

// Close closes the GCS writer.
func (w *GCSWriter) Close() error {
	w.logger.Info("closing GCS writer")
	if w.client != nil {
		return w.client.Close()
	}
	return nil
}
