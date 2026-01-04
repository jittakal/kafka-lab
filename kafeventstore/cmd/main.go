package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jittakal/kafeventstore/internal/config"
	"github.com/jittakal/kafeventstore/internal/config/dto"
	"github.com/jittakal/kafeventstore/internal/kafka"
	"github.com/jittakal/kafeventstore/internal/observability"
	"github.com/jittakal/kafeventstore/internal/server"
	"github.com/jittakal/kafeventstore/internal/storage"
	"github.com/jittakal/kafeventstore/pkg/event"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("application error: %v", err)
	}
}

func run() error {
	// Parse command-line flags
	configPath := flag.String("config", "", "path to configuration file")
	flag.Parse()

	// Load configuration
	// Priority: CLI flag > CONFIG_PATH env var > default path
	var cfgPath string
	if *configPath != "" {
		cfgPath = *configPath
	} else if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		cfgPath = envPath
	} else {
		cfgPath = "config/application.yaml"
	}

	loader := config.NewLoader()
	cfg, err := loader.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize observability
	logger := observability.NewLogger(observability.LoggingConfig{
		Level:  cfg.Observability.Logging.Level,
		Format: cfg.Observability.Logging.Format,
	})
	logger.Info("starting kafka event store",
		"version", cfg.Application.Version,
		"environment", cfg.Application.Environment,
	)

	registry := prometheus.NewRegistry()
	metrics := observability.NewMetrics(registry)

	// Track cleanup functions
	var cleanupFuncs []func() error
	addCleanup := func(name string, fn func() error) {
		cleanupFuncs = append(cleanupFuncs, fn)
		logger.Debug("registered cleanup", "component", name)
	}

	// Initialize event validator
	validator := newCloudEventsValidator()

	// Initialize partition router
	protocol := getStorageProtocol(cfg.Storage.Backend)
	bucket := getStorageBucket(cfg)
	basePath := getStorageBasePath(cfg)
	router := storage.NewRouter(protocol, bucket, basePath, "v1")

	// Initialize rotation policy
	policy := storage.NewPolicy(storage.PolicyConfig{
		MaxFileSizeMB:      cfg.FileRotation.MaxFileSizeMB,
		MaxRecordsPerFile:  cfg.FileRotation.MaxRecordsPerFile,
		MaxDurationSeconds: cfg.FileRotation.MaxDurationSeconds,
		Strategy:           cfg.FileRotation.Strategy,
	})

	// Initialize infrastructure
	consumerConfig := kafka.ConsumerConfig{
		BootstrapServers:    cfg.Kafka.BootstrapServers,
		GroupID:             cfg.Kafka.Consumer.GroupID,
		SecurityProtocol:    cfg.Kafka.SecurityProtocol,
		SASLMechanism:       cfg.Kafka.SASLMechanism,
		SASLUsername:        cfg.Kafka.SASLUsername,
		SASLPassword:        cfg.Kafka.SASLPassword,
		AutoOffsetReset:     cfg.Kafka.Consumer.AutoOffsetReset,
		EnableAutoCommit:    cfg.Kafka.Consumer.EnableAutoCommit,
		MaxPollIntervalMS:   cfg.Kafka.Consumer.MaxPollIntervalMS,
		SessionTimeoutMS:    cfg.Kafka.Consumer.SessionTimeoutMS,
		HeartbeatIntervalMS: cfg.Kafka.Consumer.HeartbeatIntervalMS,
	}
	consumer, err := kafka.NewSaramaConsumer(consumerConfig, logger, metrics)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}
	addCleanup("kafka-consumer", consumer.Close)

	dlqConfig := kafka.DLQConfig{
		Enabled:     cfg.Kafka.DLQ.Enabled,
		TopicSuffix: cfg.Kafka.DLQ.TopicSuffix,
		MaxRetries:  cfg.Kafka.DLQ.MaxRetries,
	}
	dlqPublisher, err := kafka.NewDLQPublisher(cfg.Kafka.BootstrapServers, consumerConfig, dlqConfig, logger, cfg.Application.Name)
	if err != nil {
		return fmt.Errorf("failed to create DLQ publisher: %w", err)
	}
	addCleanup("dlq-publisher", dlqPublisher.Close)

	// Get file format
	format := event.FormatParquet
	if cfg.Storage.Format == "avro" {
		format = event.FormatAvro
	}

	// Get compression (default to format-specific default if not specified)
	compression := cfg.Storage.Compression
	if compression == "" {
		if format == event.FormatParquet {
			compression = "snappy"
		} else {
			compression = "gzip"
		}
	}

	// Create storage writer based on backend
	var writer interface {
		Write(ctx context.Context, records []event.Record, path string, format event.FileFormat) (int64, error)
		Close() error
	}
	switch cfg.Storage.Backend {
	case "file":
		fileConfig := storage.FileConfig{
			BasePath: cfg.Storage.File.BasePath,
		}
		writer, err = storage.NewFileWriter(fileConfig, format, compression, logger, metrics)
		if err != nil {
			return fmt.Errorf("failed to create filesystem writer: %w", err)
		}
	case "s3":
		s3Config := storage.S3Config{
			Bucket:       cfg.Storage.S3.Bucket,
			Region:       cfg.Storage.S3.Region,
			Endpoint:     cfg.Storage.S3.Endpoint,
			UsePathStyle: cfg.Storage.S3.UsePathStyle,
			SSEEnabled:   cfg.Storage.S3.SSEEnabled,
			SSEKMSKeyID:  cfg.Storage.S3.SSEKMSKeyID,
		}
		writer, err = storage.NewS3Writer(s3Config, format, compression, logger, metrics)
		if err != nil {
			return fmt.Errorf("failed to create S3 writer: %w", err)
		}
	case "azure":
		azureConfig := storage.AzureConfig{
			AccountName:   cfg.Storage.Azure.AccountName,
			AccountKey:    os.Getenv("AZURE_STORAGE_ACCOUNT_KEY"),
			ContainerName: cfg.Storage.Azure.Container,
			Endpoint:      "",
		}
		writer, err = storage.NewAzureWriter(azureConfig, format, compression, logger, metrics)
		if err != nil {
			return fmt.Errorf("failed to create Azure Blob writer: %w", err)
		}
	case "gcs":
		gcsConfig := storage.GCSConfig{
			Bucket:               cfg.Storage.GCS.Bucket,
			ProjectID:            cfg.Storage.GCS.ProjectID,
			CredentialsFile:      cfg.Storage.GCS.CredentialsFile,
			CredentialsJSON:      os.Getenv("GCP_CREDENTIALS_JSON"),
			UseDefaultCredential: cfg.Storage.GCS.UseDefaultCredential,
		}
		writer, err = storage.NewGCSWriter(gcsConfig, format, compression, logger, metrics)
		if err != nil {
			return fmt.Errorf("failed to create GCS writer: %w", err)
		}
	default:
		return fmt.Errorf("unsupported storage backend: %s (supported: file, s3, azure, gcs)", cfg.Storage.Backend)
	}
	addCleanup("storage-writer", writer.Close)

	// Initialize buffer manager
	bufferSizeBytes := int64(cfg.Processing.BufferSizeMB * 1024 * 1024)
	bufferMgr := newSimpleBufferManager(bufferSizeBytes, cfg.FileRotation.MaxRecordsPerFile)

	// Create simple health checker
	healthChecker := &simpleHealthChecker{isHealthy: true}

	// Start HTTP server
	httpServer := server.NewServer(
		cfg.Observability.Health.Port,
		cfg.Observability.Metrics.Port,
		healthChecker,
		registry,
		logger,
	)

	if err := httpServer.Start(); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	addCleanup("http-server", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpServer.Shutdown(ctx)
	})

	logger.Info("application started successfully")

	// Subscribe to topics
	if err := consumer.Subscribe(context.Background(), cfg.Kafka.Consumer.Topics); err != nil {
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consuming
	eventChan, errorChan, err := consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	// Start consume loop in background
	consumeErrChan := make(chan error, 1)
	go func() {
		consumeErrChan <- processEvents(ctx, eventChan, errorChan, validator, writer, router, policy, dlqPublisher, bufferMgr, format, logger, metrics)
	}()

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		logger.Info("received termination signal")
	case err := <-consumeErrChan:
		if err != nil {
			logger.Error("consume error", "error", err)
			return err
		}
	}

	// Graceful shutdown
	logger.Info("initiating graceful shutdown")
	cancel()

	// Allow time for in-flight operations to complete
	shutdownTimeout := time.Duration(cfg.Shutdown.GracePeriodSeconds) * time.Second
	time.Sleep(shutdownTimeout)

	logger.Info("application stopped successfully")
	return nil
}

func processEvents(
	ctx context.Context,
	eventChan <-chan *event.ConsumedEvent,
	errorChan <-chan error,
	validator *simpleValidator,
	writer interface {
		Write(ctx context.Context, records []event.Record, path string, format event.FileFormat) (int64, error)
		Close() error
	},
	router *storage.DefaultRouter,
	policy *storage.CompositePolicy,
	dlq *kafka.DLQPublisher,
	bufferMgr *simpleBufferManager,
	format event.FileFormat,
	logger *slog.Logger,
	metrics *observability.Metrics,
) error {
	fileStats := make(map[event.PartitionID]event.FileStats)

	for {
		select {
		case <-ctx.Done():
			logger.Info("context cancelled, stopping processing")
			return nil
		case err := <-errorChan:
			if err != nil {
				logger.Error("consumer error", "error", err)
			}
		case consumedEvent, ok := <-eventChan:
			if !ok {
				logger.Info("event channel closed")
				return nil
			}

			partitionID := event.PartitionID{
				Topic:     consumedEvent.Metadata.Topic,
				Partition: consumedEvent.Metadata.Partition,
			}

			// Validate event
			if err := validator.Validate(consumedEvent.Event); err != nil {
				logger.Warn("invalid cloud event",
					"topic", partitionID.Topic,
					"partition", partitionID.Partition,
					"offset", consumedEvent.Metadata.Offset,
					"error", err,
				)

				// Send to DLQ
				if dlq != nil {
					_ = dlq.Publish(ctx, consumedEvent.Event, consumedEvent.Metadata, "validation_failed")
				}

				// Commit the offset to skip bad message
				if consumedEvent.CommitFunc != nil {
					_ = consumedEvent.CommitFunc()
				}
				continue
			}

			// Create storage record
			record := event.Record{
				Event:       consumedEvent.Event,
				Kafka:       consumedEvent.Metadata,
				Offset:      consumedEvent.Metadata.Offset,
				ProcessedAt: time.Now(),
			}

			// Add to buffer
			bufferMgr.append(partitionID, record)

			// Update file stats
			stats := fileStats[partitionID]
			stats.RecordCount++
			if len(consumedEvent.Event.Data) > 0 {
				stats.SizeBytes += int64(len(consumedEvent.Event.Data))
			}
			fileStats[partitionID] = stats

			// Check if we should flush
			if bufferMgr.shouldFlush(partitionID, policy, stats) {
				records := bufferMgr.getRecords(partitionID)
				if len(records) > 0 {
					// Get event time and spec_version from first record
					// All records in batch should have similar timestamps (within rotation window)
					eventTime := records[0].GetEventTimeUnix()
					specVersion := ""
					if records[0].Event != nil {
						specVersion = records[0].Event.SpecVersion
					}

					// Get storage path using event time (not processing time)
					path := router.Route(partitionID, eventTime, specVersion)

					// Write to storage
					bytesWritten, err := writer.Write(ctx, records, path, format)
					if err != nil {
						logger.Error("failed to write to storage",
							"topic", partitionID.Topic,
							"partition", partitionID.Partition,
							"error", err,
						)

						// Send to DLQ
						if dlq != nil {
							for _, rec := range records {
								_ = dlq.Publish(ctx, rec.Event, rec.Kafka, "storage_failed")
							}
						}
					} else {
						logger.Info("wrote batch to storage",
							"topic", partitionID.Topic,
							"partition", partitionID.Partition,
							"records", len(records),
							"bytes", bytesWritten,
							"path", path,
						)
					}

					// Clear buffer and reset stats
					bufferMgr.clear(partitionID)
					delete(fileStats, partitionID)
				}
			}

			// Commit offset
			if consumedEvent.CommitFunc != nil {
				if err := consumedEvent.CommitFunc(); err != nil {
					logger.Error("failed to commit offset",
						"topic", partitionID.Topic,
						"partition", partitionID.Partition,
						"offset", consumedEvent.Metadata.Offset,
						"error", err,
					)
				}
			}
		}
	}
}

func getStorageProtocol(backend string) string {
	switch backend {
	case "s3":
		return "s3"
	case "azure":
		return "wasbs"
	case "gcs":
		return "gs"
	case "file":
		return "file"
	default:
		return "file"
	}
}

func getStorageBucket(cfg *dto.ApplicationConfig) string {
	switch cfg.Storage.Backend {
	case "s3":
		return cfg.Storage.S3.Bucket
	case "azure":
		return cfg.Storage.Azure.Container
	case "gcs":
		return cfg.Storage.GCS.Bucket
	case "file":
		return "" // File backend uses basePath only, no bucket
	default:
		return "events"
	}
}

func getStorageBasePath(cfg *dto.ApplicationConfig) string {
	switch cfg.Storage.Backend {
	case "s3":
		return cfg.Storage.S3.BasePath
	case "gcs":
		return cfg.Storage.GCS.BasePath
	case "azure":
		return ""
	case "file":
		return "" // File backend: basePath is handled by FileWriter, router only needs topic/date/partition structure
	default:
		return ""
	}
}

// simpleHealthChecker implements server.HealthChecker interface
type simpleHealthChecker struct {
	isHealthy bool
}

func (h *simpleHealthChecker) Liveness() bool {
	return h.isHealthy
}

func (h *simpleHealthChecker) Readiness(ctx context.Context) bool {
	return h.isHealthy
}

func (h *simpleHealthChecker) IsHealthy() bool {
	return h.isHealthy
}

func (h *simpleHealthChecker) GetStatus() map[string]string {
	return map[string]string{
		"status": "healthy",
	}
}

// simpleValidator validates CloudEvents
type simpleValidator struct{}

func newCloudEventsValidator() *simpleValidator {
	return &simpleValidator{}
}

func (v *simpleValidator) Validate(evt *event.CloudEvent) error {
	if evt == nil {
		return fmt.Errorf("event is nil")
	}
	if evt.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if evt.Source == "" {
		return fmt.Errorf("event source is required")
	}
	if evt.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if evt.SpecVersion == "" {
		return fmt.Errorf("event spec version is required")
	}
	return nil
}

// simpleBufferManager manages per-partition buffers
type simpleBufferManager struct {
	buffers             map[event.PartitionID]*partitionBuffer
	maxBufferSize       int64
	maxRecordsPerBuffer int
}

type partitionBuffer struct {
	records []event.Record
	size    int64
}

func newSimpleBufferManager(maxSize int64, maxRecords int) *simpleBufferManager {
	return &simpleBufferManager{
		buffers:             make(map[event.PartitionID]*partitionBuffer),
		maxBufferSize:       maxSize,
		maxRecordsPerBuffer: maxRecords,
	}
}

func (m *simpleBufferManager) append(partitionID event.PartitionID, record event.Record) {
	if _, exists := m.buffers[partitionID]; !exists {
		m.buffers[partitionID] = &partitionBuffer{
			records: make([]event.Record, 0, m.maxRecordsPerBuffer),
		}
	}
	buf := m.buffers[partitionID]
	buf.records = append(buf.records, record)
	buf.size += int64(len(record.Event.Data))
}

func (m *simpleBufferManager) shouldFlush(partitionID event.PartitionID, policy *storage.CompositePolicy, stats event.FileStats) bool {
	buf, exists := m.buffers[partitionID]
	if !exists {
		return false
	}
	return policy.ShouldRotate(stats) || len(buf.records) >= m.maxRecordsPerBuffer
}

func (m *simpleBufferManager) getRecords(partitionID event.PartitionID) []event.Record {
	buf, exists := m.buffers[partitionID]
	if !exists {
		return nil
	}
	return buf.records
}

func (m *simpleBufferManager) clear(partitionID event.PartitionID) {
	delete(m.buffers, partitionID)
}
