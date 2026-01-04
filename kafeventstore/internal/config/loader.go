package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jittakal/kafeventstore/internal/config/dto"
	"github.com/spf13/viper"
)

// Loader handles configuration loading and validation
type Loader struct {
	v *viper.Viper
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	return &Loader{v: v}
}

// Load loads configuration from file and environment variables
func (l *Loader) Load(path string) (*dto.ApplicationConfig, error) {
	// Set defaults
	l.setDefaults()

	// Load from file if provided
	if path != "" {
		l.v.SetConfigFile(path)
		if err := l.v.ReadInConfig(); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	// Expand environment variables in config values
	// Only expand if the value contains ${...} pattern
	for _, key := range l.v.AllKeys() {
		value := l.v.GetString(key)
		if strings.Contains(value, "${") {
			l.v.Set(key, os.ExpandEnv(value))
		}
	}

	// Unmarshal configuration
	var config dto.ApplicationConfig
	if err := l.v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := l.Validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func (l *Loader) setDefaults() {
	// Application defaults
	l.v.SetDefault("application.name", "kafka-event-store")
	l.v.SetDefault("application.version", "1.0.0")
	l.v.SetDefault("application.environment", "development")

	// Kafka defaults
	l.v.SetDefault("kafka.security_protocol", "SASL_SSL")
	l.v.SetDefault("kafka.sasl_mechanism", "PLAIN")
	l.v.SetDefault("kafka.consumer.auto_offset_reset", "earliest")
	l.v.SetDefault("kafka.consumer.enable_auto_commit", false)
	l.v.SetDefault("kafka.consumer.max_poll_records", 1000)
	l.v.SetDefault("kafka.consumer.max_poll_interval_ms", 300000)
	l.v.SetDefault("kafka.consumer.session_timeout_ms", 30000)
	l.v.SetDefault("kafka.consumer.heartbeat_interval_ms", 10000)
	l.v.SetDefault("kafka.dlq.enabled", true)
	l.v.SetDefault("kafka.dlq.topic_suffix", "-dlq")
	l.v.SetDefault("kafka.dlq.max_retries", 3)

	// Storage defaults
	l.v.SetDefault("storage.backend", "file")
	l.v.SetDefault("storage.format", "parquet")
	l.v.SetDefault("storage.s3.use_path_style", false)
	l.v.SetDefault("storage.s3.sse_enabled", true)

	// File rotation defaults
	l.v.SetDefault("file_rotation.max_file_size_mb", 128)
	l.v.SetDefault("file_rotation.max_records_per_file", 100000)
	l.v.SetDefault("file_rotation.max_duration_seconds", 300)
	l.v.SetDefault("file_rotation.strategy", "any")

	// Parquet defaults
	l.v.SetDefault("parquet.compression", "snappy")
	l.v.SetDefault("parquet.row_group_size_mb", 100)
	l.v.SetDefault("parquet.page_size_kb", 1024)
	l.v.SetDefault("parquet.enable_statistics", true)
	l.v.SetDefault("parquet.enable_dictionary", true)

	// Avro defaults
	l.v.SetDefault("avro.codec", "snappy")
	l.v.SetDefault("avro.sync_interval", 16000)

	// Processing defaults
	l.v.SetDefault("processing.buffer_size_mb", 64)
	l.v.SetDefault("processing.buffer_flush_interval_seconds", 60)
	l.v.SetDefault("processing.max_concurrent_uploads", 5)
	l.v.SetDefault("processing.worker_pool_size", 10)
	l.v.SetDefault("processing.checkpoint_dir", "") // Empty by default - rely on Kafka offset management

	// Retry defaults
	l.v.SetDefault("retry.max_attempts", 5)
	l.v.SetDefault("retry.initial_backoff_ms", 100)
	l.v.SetDefault("retry.max_backoff_ms", 30000)
	l.v.SetDefault("retry.backoff_multiplier", 2.0)
	l.v.SetDefault("retry.enable_jitter", true)

	// Observability defaults
	l.v.SetDefault("observability.logging.level", "info")
	l.v.SetDefault("observability.logging.format", "json")
	l.v.SetDefault("observability.logging.output", "stdout")
	l.v.SetDefault("observability.metrics.enabled", true)
	l.v.SetDefault("observability.metrics.port", 9090)
	l.v.SetDefault("observability.metrics.path", "/metrics")
	l.v.SetDefault("observability.tracing.enabled", false)
	l.v.SetDefault("observability.tracing.exporter", "otlp")
	l.v.SetDefault("observability.tracing.sample_rate", 0.1)
	l.v.SetDefault("observability.health.port", 8080)
	l.v.SetDefault("observability.health.liveness_path", "/health/live")
	l.v.SetDefault("observability.health.readiness_path", "/health/ready")

	// Shutdown defaults
	l.v.SetDefault("shutdown.grace_period_seconds", 30)
	l.v.SetDefault("shutdown.force_timeout_seconds", 60)
}

// Validate validates the configuration
func (l *Loader) Validate(config *dto.ApplicationConfig) error {
	// Kafka validation
	if len(config.Kafka.BootstrapServers) == 0 {
		return errors.New("kafka.bootstrap_servers is required")
	}
	if len(config.Kafka.Consumer.Topics) == 0 {
		return errors.New("kafka.consumer.topics is required")
	}
	if config.Kafka.Consumer.GroupID == "" {
		return errors.New("kafka.consumer.group_id is required")
	}

	// Storage validation
	switch config.Storage.Backend {
	case "s3":
		if config.Storage.S3.Bucket == "" {
			return errors.New("storage.s3.bucket is required for S3 backend")
		}
		if config.Storage.S3.Region == "" {
			return errors.New("storage.s3.region is required for S3 backend")
		}
	case "azure":
		if config.Storage.Azure.AccountName == "" {
			return errors.New("storage.azure.account_name is required for Azure backend")
		}
		if config.Storage.Azure.Container == "" {
			return errors.New("storage.azure.container is required for Azure backend")
		}
	case "gcs":
		if config.Storage.GCS.Bucket == "" {
			return errors.New("storage.gcs.bucket is required for GCS backend")
		}
	case "file":
		if config.Storage.File.BasePath == "" {
			return errors.New("storage.file.base_path is required for file backend")
		}
	default:
		return fmt.Errorf("unsupported storage backend: %s", config.Storage.Backend)
	}

	// Format validation
	if config.Storage.Format != "parquet" && config.Storage.Format != "avro" {
		return fmt.Errorf("unsupported storage format: %s", config.Storage.Format)
	}

	// File rotation validation
	if config.FileRotation.Strategy != "any" && config.FileRotation.Strategy != "all" {
		return fmt.Errorf("unsupported rotation strategy: %s", config.FileRotation.Strategy)
	}

	// Port validation
	if config.Observability.Metrics.Port < 1 || config.Observability.Metrics.Port > 65535 {
		return fmt.Errorf("invalid metrics port: %d", config.Observability.Metrics.Port)
	}
	if config.Observability.Health.Port < 1 || config.Observability.Health.Port > 65535 {
		return fmt.Errorf("invalid health port: %d", config.Observability.Health.Port)
	}

	return nil
}
