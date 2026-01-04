// Package storage implements S3 storage writer.
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

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/jittakal/kafeventstore/internal/encoder"
	"github.com/jittakal/kafeventstore/pkg/event"
	"github.com/jittakal/kafeventstore/pkg/storage"
)

// Ensure implementation satisfies interface at compile time.
var _ storage.Writer = (*S3Writer)(nil)

// S3Config contains AWS S3 configuration.
type S3Config struct {
	Bucket       string
	Region       string
	Endpoint     string
	UsePathStyle bool
	SSEEnabled   bool
	SSEKMSKeyID  string
}

// S3Writer implements storage.Writer for AWS S3 storage.
// It provides multipart upload support, server-side encryption (SSE),
// and automatic retry handling for S3 operations.
type S3Writer struct {
	client         *s3.Client
	uploader       *manager.Uploader
	bucket         string
	region         string
	sseEnabled     bool
	sseKMSKeyID    string
	encoderFactory *encoder.Factory
	logger         *slog.Logger
	metrics        MetricsCollector
	mu             sync.RWMutex
}

// NewS3Writer creates a new S3 storage writer.
func NewS3Writer(
	cfg S3Config,
	format event.FileFormat,
	compression string,
	logger *slog.Logger,
	metrics MetricsCollector,
) (*S3Writer, error) {
	// Load AWS config
	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})

	// Create uploader with multipart upload support
	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024 // 10MB parts
		u.Concurrency = 5             // 5 concurrent uploads
	})

	// Create encoder factory
	encoderFactory := encoder.NewFactory(format, compression)

	// Validate encoder can be created
	if _, err := encoderFactory.CreateEncoder(); err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	logger.Info("S3 writer created",
		"bucket", cfg.Bucket,
		"region", cfg.Region,
		"format", format,
		"compression", compression,
		"sse_enabled", cfg.SSEEnabled,
	)

	return &S3Writer{
		client:         s3Client,
		uploader:       uploader,
		bucket:         cfg.Bucket,
		region:         cfg.Region,
		sseEnabled:     cfg.SSEEnabled,
		sseKMSKeyID:    cfg.SSEKMSKeyID,
		encoderFactory: encoderFactory,
		logger:         logger,
		metrics:        metrics,
	}, nil
}

// Write writes records to S3.
func (w *S3Writer) Write(
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
			w.metrics.IncStorageErrors("s3", "encoder_create")
		}
		return 0, fmt.Errorf("failed to create encoder: %w", err)
	}

	// Parse S3 URI to extract key
	// Path format: s3://bucket/key/path or just key/path
	s3Key := path
	if strings.HasPrefix(path, "s3://") {
		// Remove s3:// prefix
		pathWithoutProtocol := strings.TrimPrefix(path, "s3://")
		// Remove bucket name (first segment)
		parts := strings.SplitN(pathWithoutProtocol, "/", 2)
		if len(parts) == 2 {
			s3Key = parts[1]
		} else {
			s3Key = ""
		}
	}

	// Generate timestamped filename: events_YYYYMMDD_HHMMSS_NNN.{ext}
	now := time.Now()
	timestamp := now.Format("20060102_150405")
	filename := fmt.Sprintf("events_%s_%03d%s", timestamp, now.Nanosecond()/1000000, fileEncoder.FileExtension())
	s3Key = s3Key + filename
	s3Key = strings.TrimPrefix(s3Key, "/")

	// Encode to temporary file
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("s3-upload-%d%s", now.UnixNano(), fileEncoder.FileExtension()))

	stats, err := fileEncoder.Encode(tempFile, records)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("s3", "encode")
		}
		return 0, fmt.Errorf("failed to encode records: %w", err)
	}
	defer os.Remove(tempFile)

	// Open the file for upload
	file, err := os.Open(tempFile)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("s3", "file_open")
		}
		return 0, fmt.Errorf("failed to open encoded file: %w", err)
	}
	defer file.Close()

	// Prepare upload input
	uploadInput := &s3.PutObjectInput{
		Bucket: aws.String(w.bucket),
		Key:    aws.String(s3Key),
		Body:   file,
	}

	// Add SSE if enabled
	if w.sseEnabled {
		if w.sseKMSKeyID != "" {
			uploadInput.ServerSideEncryption = types.ServerSideEncryptionAwsKms
			uploadInput.SSEKMSKeyId = aws.String(w.sseKMSKeyID)
		} else {
			uploadInput.ServerSideEncryption = types.ServerSideEncryptionAes256
		}
	}

	// Upload to S3
	result, err := w.uploader.Upload(ctx, uploadInput)
	if err != nil {
		if w.metrics != nil {
			w.metrics.IncStorageErrors("s3", "upload")
		}
		return 0, fmt.Errorf("failed to upload to S3: %w", err)
	}

	duration := time.Since(startTime)

	w.logger.Info("wrote records to S3",
		"bucket", w.bucket,
		"key", s3Key,
		"record_count", stats.RecordCount,
		"file_size", stats.SizeBytes,
		"format", format,
		"location", result.Location,
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

// Close closes the S3 writer.
func (w *S3Writer) Close() error {
	w.logger.Info("closing S3 writer")
	return nil
}
