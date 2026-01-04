package dto

import (
	"fmt"
	"time"
)

// ApplicationConfig is the root configuration structure
type ApplicationConfig struct {
	Application   ApplicationInfo     `mapstructure:"application"`
	Kafka         KafkaConfig         `mapstructure:"kafka"`
	Storage       StorageConfig       `mapstructure:"storage"`
	FileRotation  FileRotationConfig  `mapstructure:"file_rotation"`
	Parquet       ParquetConfig       `mapstructure:"parquet"`
	Avro          AvroConfig          `mapstructure:"avro"`
	Processing    ProcessingConfig    `mapstructure:"processing"`
	Retry         RetryConfig         `mapstructure:"retry"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	Shutdown      ShutdownConfig      `mapstructure:"shutdown"`
}

// ApplicationInfo contains application metadata
type ApplicationInfo struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
}

// KafkaConfig contains Kafka-related configuration
type KafkaConfig struct {
	BootstrapServers []string       `mapstructure:"bootstrap_servers"`
	SecurityProtocol string         `mapstructure:"security_protocol"`
	SASLMechanism    string         `mapstructure:"sasl_mechanism"`
	SASLUsername     string         `mapstructure:"sasl_username"`
	SASLPassword     string         `mapstructure:"sasl_password"`
	Consumer         ConsumerConfig `mapstructure:"consumer"`
	DLQ              DLQConfig      `mapstructure:"dlq"`
}

// ConsumerConfig contains Kafka consumer configuration
type ConsumerConfig struct {
	GroupID             string   `mapstructure:"group_id"`
	Topics              []string `mapstructure:"topics"`
	AutoOffsetReset     string   `mapstructure:"auto_offset_reset"`
	EnableAutoCommit    bool     `mapstructure:"enable_auto_commit"`
	MaxPollRecords      int      `mapstructure:"max_poll_records"`
	MaxPollIntervalMS   int      `mapstructure:"max_poll_interval_ms"`
	SessionTimeoutMS    int      `mapstructure:"session_timeout_ms"`
	HeartbeatIntervalMS int      `mapstructure:"heartbeat_interval_ms"`
}

// DLQConfig contains dead letter queue configuration
type DLQConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	TopicSuffix string `mapstructure:"topic_suffix"`
	MaxRetries  int    `mapstructure:"max_retries"`
}

// StorageConfig contains storage backend configuration
type StorageConfig struct {
	Backend      string      `mapstructure:"backend"`
	Format       string      `mapstructure:"format"`
	Compression  string      `mapstructure:"compression"`
	PathTemplate string      `mapstructure:"path_template"`
	S3           S3Config    `mapstructure:"s3"`
	Azure        AzureConfig `mapstructure:"azure"`
	GCS          GCSConfig   `mapstructure:"gcs"`
	File         FileConfig  `mapstructure:"file"`
}

// S3Config contains AWS S3 configuration
type S3Config struct {
	Bucket       string `mapstructure:"bucket"`
	Region       string `mapstructure:"region"`
	BasePath     string `mapstructure:"base_path"`
	Endpoint     string `mapstructure:"endpoint"`
	UsePathStyle bool   `mapstructure:"use_path_style"`
	SSEEnabled   bool   `mapstructure:"sse_enabled"`
	SSEKMSKeyID  string `mapstructure:"sse_kms_key_id"`
}

// AzureConfig contains Azure Blob Storage configuration
type AzureConfig struct {
	AccountName        string `mapstructure:"account_name"`
	Container          string `mapstructure:"container"`
	UseManagedIdentity bool   `mapstructure:"use_managed_identity"`
}

// GCSConfig contains Google Cloud Storage configuration
type GCSConfig struct {
	Bucket               string `mapstructure:"bucket"`
	ProjectID            string `mapstructure:"project_id"`
	BasePath             string `mapstructure:"base_path"`
	CredentialsFile      string `mapstructure:"credentials_file"`
	CredentialsJSON      string `mapstructure:"credentials_json"`
	UseDefaultCredential bool   `mapstructure:"use_default_credential"`
}

// FileConfig contains local filesystem configuration
type FileConfig struct {
	BasePath string `mapstructure:"base_path"`
}

// FileRotationConfig contains file rotation settings
type FileRotationConfig struct {
	MaxFileSizeMB      int64  `mapstructure:"max_file_size_mb"`
	MaxRecordsPerFile  int    `mapstructure:"max_records_per_file"`
	MaxDurationSeconds int    `mapstructure:"max_duration_seconds"`
	Strategy           string `mapstructure:"strategy"`
}

// ParquetConfig contains Parquet format settings
type ParquetConfig struct {
	Compression      string `mapstructure:"compression"`
	RowGroupSizeMB   int    `mapstructure:"row_group_size_mb"`
	PageSizeKB       int    `mapstructure:"page_size_kb"`
	EnableStatistics bool   `mapstructure:"enable_statistics"`
	EnableDictionary bool   `mapstructure:"enable_dictionary"`
}

// AvroConfig contains Avro format settings
type AvroConfig struct {
	Codec        string `mapstructure:"codec"`
	SyncInterval int    `mapstructure:"sync_interval"`
}

// ProcessingConfig contains processing settings
type ProcessingConfig struct {
	BufferSizeMB           int    `mapstructure:"buffer_size_mb"`
	BufferFlushIntervalSec int    `mapstructure:"buffer_flush_interval_seconds"`
	MaxConcurrentUploads   int    `mapstructure:"max_concurrent_uploads"`
	WorkerPoolSize         int    `mapstructure:"worker_pool_size"`
	CheckpointDir          string `mapstructure:"checkpoint_dir"`
}

// RetryConfig contains retry settings
type RetryConfig struct {
	Enabled                        bool    `mapstructure:"enabled"`
	MaxAttempts                    int     `mapstructure:"max_attempts"`
	InitialBackoffMS               int     `mapstructure:"initial_backoff_ms"`
	MaxBackoffMS                   int     `mapstructure:"max_backoff_ms"`
	BackoffMultiplier              float64 `mapstructure:"backoff_multiplier"`
	Jitter                         bool    `mapstructure:"jitter"`
	CircuitBreakerEnabled          bool    `mapstructure:"circuit_breaker_enabled"`
	CircuitBreakerMaxFailures      int     `mapstructure:"circuit_breaker_max_failures"`
	CircuitBreakerTimeoutSeconds   int     `mapstructure:"circuit_breaker_timeout_seconds"`
	CircuitBreakerMaxRequests      int     `mapstructure:"circuit_breaker_max_requests"`
	CircuitBreakerSuccessThreshold int     `mapstructure:"circuit_breaker_success_threshold"`
}

// ObservabilityConfig contains observability settings
type ObservabilityConfig struct {
	Logging LoggingConfig `mapstructure:"logging"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Tracing TracingConfig `mapstructure:"tracing"`
	Health  HealthConfig  `mapstructure:"health"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// MetricsConfig contains metrics settings
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

// TracingConfig contains tracing settings
type TracingConfig struct {
	Enabled    bool    `mapstructure:"enabled"`
	Exporter   string  `mapstructure:"exporter"`
	Endpoint   string  `mapstructure:"endpoint"`
	SampleRate float64 `mapstructure:"sample_rate"`
}

// HealthConfig contains health check settings
type HealthConfig struct {
	Port          int    `mapstructure:"port"`
	LivenessPath  string `mapstructure:"liveness_path"`
	ReadinessPath string `mapstructure:"readiness_path"`
}

// ShutdownConfig contains shutdown settings
type ShutdownConfig struct {
	GracePeriodSeconds  time.Duration `mapstructure:"grace_period_seconds"`
	ForceTimeoutSeconds time.Duration `mapstructure:"force_timeout_seconds"`
}

// Validate validates the application configuration.
func (c *ApplicationConfig) Validate() error {
	if c.Application.Name == "" {
		return fmt.Errorf("application name is required")
	}
	if len(c.Kafka.BootstrapServers) == 0 {
		return fmt.Errorf("kafka bootstrap servers are required")
	}
	if c.Kafka.Consumer.GroupID == "" {
		return fmt.Errorf("kafka consumer group ID is required")
	}
	if c.Storage.Backend == "" {
		return fmt.Errorf("storage backend is required")
	}
	return nil
}

// Validate validates S3 configuration.
func (c *S3Config) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("s3 bucket is required")
	}
	if c.Region == "" {
		return fmt.Errorf("s3 region is required")
	}
	return nil
}

// Validate validates Azure configuration.
func (c *AzureConfig) Validate() error {
	if c.AccountName == "" {
		return fmt.Errorf("azure account name is required")
	}
	if c.Container == "" {
		return fmt.Errorf("azure container is required")
	}
	return nil
}

// Validate validates file configuration.
func (c *FileConfig) Validate() error {
	if c.BasePath == "" {
		return fmt.Errorf("file base path is required")
	}
	return nil
}
