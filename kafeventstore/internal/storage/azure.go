// Package storage implements Azure Blob storage writer.
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

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/jittakal/kafeventstore/internal/encoder"
	"github.com/jittakal/kafeventstore/pkg/event"
	"github.com/jittakal/kafeventstore/pkg/storage"
)

// Ensure implementation satisfies interface at compile time.
var _ storage.Writer = (*AzureWriter)(nil)

// AzureConfig contains Azure Blob Storage configuration.
type AzureConfig struct {
	AccountName   string
	AccountKey    string
	ContainerName string
	Endpoint      string
}

// AzureWriter implements storage.Writer for Azure Blob Storage.
// It supports both managed identity and access key authentication,
// with automatic blob creation and hierarchical path organization.
type AzureWriter struct {
	client         *azblob.Client
	containerName  string
	encoderFactory *encoder.Factory
	logger         *slog.Logger
	metrics        MetricsCollector
	mu             sync.RWMutex
}

// NewAzureWriter creates a new Azure Blob storage writer.
func NewAzureWriter(
	cfg AzureConfig,
	format event.FileFormat,
	compression string,
	logger *slog.Logger,
	metrics MetricsCollector,
) (*AzureWriter, error) {
	// Build connection string
	var connectionString string
	if cfg.Endpoint != "" {
		connectionString = fmt.Sprintf("DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;BlobEndpoint=%s",
			cfg.AccountName, cfg.AccountKey, cfg.Endpoint)
	} else {
		connectionString = fmt.Sprintf("DefaultEndpointsProtocol=https;AccountName=%s;AccountKey=%s;EndpointSuffix=core.windows.net",
			cfg.AccountName, cfg.AccountKey)
	}

	// Create Azure client
	client, err := azblob.NewClientFromConnectionString(connectionString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure client: %w", err)
	}

	// Create encoder factory
	encoderFactory := encoder.NewFactory(format, compression)

	// Validate encoder can be created
	if _, err := encoderFactory.CreateEncoder(); err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	logger.Info("Azure writer created",
		"container", cfg.ContainerName,
		"account", cfg.AccountName,
		"format", format,
		"compression", compression,
	)

	return &AzureWriter{
		client:         client,
		containerName:  cfg.ContainerName,
		encoderFactory: encoderFactory,
		logger:         logger,
		metrics:        metrics,
	}, nil
}

// Write writes records to Azure Blob Storage.
func (w *AzureWriter) Write(ctx context.Context, records []event.Record, path string, format event.FileFormat) (int64, error) {
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
			w.metrics.IncStorageErrors("azure", "encoder_create")
		}
		return 0, fmt.Errorf("failed to create encoder: %w", err)
	}

	// Parse Azure URI to extract blob path
	blobPath := path
	if strings.HasPrefix(path, "wasbs://") {
		// Remove wasbs:// prefix and container
		pathWithoutProtocol := strings.TrimPrefix(path, "wasbs://")
		parts := strings.SplitN(pathWithoutProtocol, "/", 2)
		if len(parts) == 2 {
			blobPath = parts[1]
		} else {
			blobPath = ""
		}
	}

	// Generate timestamped filename: events_YYYYMMDD_HHMMSS_NNN.{ext}
	now := time.Now()
	timestamp := now.Format("20060102_150405")
	filename := fmt.Sprintf("events_%s_%03d%s", timestamp, now.Nanosecond()/1000000, enc.FileExtension())
	blobPath = blobPath + filename
	blobPath = strings.TrimPrefix(blobPath, "/")

	// Encode to temporary file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("azure-upload-%d%s", now.UnixNano(), enc.FileExtension()))

	stats, err := enc.Encode(tempFile, records)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("azure", "encode")
		}
		return 0, fmt.Errorf("failed to encode records: %w", err)
	}
	defer os.Remove(tempFile)

	// Open the file for upload
	file, err := os.Open(tempFile)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("azure", "file_open")
		}
		return 0, fmt.Errorf("failed to open encoded file: %w", err)
	}
	defer file.Close()

	// Upload to Azure Blob
	_, err = w.client.UploadFile(ctx, w.containerName, blobPath, file, nil)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("azure", "upload")
		}
		return 0, fmt.Errorf("failed to upload to Azure Blob: %w", err)
	}

	duration := time.Since(startTime)

	w.logger.Info("wrote records to Azure Blob",
		"container", w.containerName,
		"blob", blobPath,
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

// Close closes the Azure writer.
func (w *AzureWriter) Close() error {
	w.logger.Info("Azure writer closed")
	return nil
}
