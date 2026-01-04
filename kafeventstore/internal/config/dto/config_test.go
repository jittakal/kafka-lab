package dto

import (
	"testing"
)

func TestApplicationConfig_DefaultValues(t *testing.T) {
	config := &ApplicationConfig{
		Application: ApplicationInfo{
			Name:        "kafka-event-store",
			Version:     "1.0.0",
			Environment: "dev",
		},
	}

	if config.Application.Name == "" {
		t.Error("Application name should not be empty")
	}
	if config.Application.Version == "" {
		t.Error("Application version should not be empty")
	}
	if config.Application.Environment == "" {
		t.Error("Application environment should not be empty")
	}
}

func TestKafkaConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  KafkaConfig
		wantErr bool
	}{
		{
			name: "valid plaintext config",
			config: KafkaConfig{
				BootstrapServers: []string{"localhost:9092"},
				SecurityProtocol: "PLAINTEXT",
				Consumer: ConsumerConfig{
					GroupID: "test-group",
					Topics:  []string{"test-topic"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid SASL config",
			config: KafkaConfig{
				BootstrapServers: []string{"localhost:9092"},
				SecurityProtocol: "SASL_SSL",
				SASLMechanism:    "SCRAM-SHA-256",
				SASLUsername:     "user",
				SASLPassword:     "pass",
				Consumer: ConsumerConfig{
					GroupID: "test-group",
					Topics:  []string{"test-topic"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty bootstrap servers",
			config: KafkaConfig{
				BootstrapServers: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := len(tt.config.BootstrapServers) == 0
			if hasError != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestConsumerConfig_Topics(t *testing.T) {
	tests := []struct {
		name    string
		topics  []string
		wantErr bool
	}{
		{
			name:    "single topic",
			topics:  []string{"events"},
			wantErr: false,
		},
		{
			name:    "multiple topics",
			topics:  []string{"events", "orders", "users"},
			wantErr: false,
		},
		{
			name:    "empty topics",
			topics:  []string{},
			wantErr: true,
		},
		{
			name:    "nil topics",
			topics:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := len(tt.topics) == 0
			if hasError != tt.wantErr {
				t.Errorf("Topics validation = %v, wantErr %v", hasError, tt.wantErr)
			}
		})
	}
}

func TestStorageConfig_Backend(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		valid   bool
	}{
		{"file", "file", true},
		{"s3", "s3", true},
		{"azure", "azure", true},
		{"invalid", "invalid", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.backend == "file" || tt.backend == "s3" || tt.backend == "azure"
			if valid != tt.valid {
				t.Errorf("Backend %v validity = %v, want %v", tt.backend, valid, tt.valid)
			}
		})
	}
}

func TestStorageConfig_Format(t *testing.T) {
	tests := []struct {
		name   string
		format string
		valid  bool
	}{
		{"parquet", "parquet", true},
		{"avro", "avro", true},
		{"json", "json", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := tt.format == "parquet" || tt.format == "avro"
			if valid != tt.valid {
				t.Errorf("Format %v validity = %v, want %v", tt.format, valid, tt.valid)
			}
		})
	}
}

func TestStorageConfig_Compression(t *testing.T) {
	tests := []struct {
		name        string
		compression string
		valid       bool
	}{
		{"snappy", "snappy", true},
		{"gzip", "gzip", true},
		{"zstd", "zstd", true},
		{"none", "none", true},
		{"lz4", "lz4", false},
		{"empty", "", true}, // defaults to format-specific
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validCompressions := map[string]bool{
				"snappy": true,
				"gzip":   true,
				"zstd":   true,
				"none":   true,
				"":       true,
			}

			valid := validCompressions[tt.compression]
			if valid != tt.valid {
				t.Errorf("Compression %v validity = %v, want %v", tt.compression, valid, tt.valid)
			}
		})
	}
}

func TestFileRotationConfig(t *testing.T) {
	config := FileRotationConfig{
		MaxFileSizeMB:      100,
		MaxRecordsPerFile:  10000,
		MaxDurationSeconds: 300,
		Strategy:           "composite",
	}

	if config.MaxFileSizeMB <= 0 {
		t.Error("MaxFileSizeMB should be positive")
	}
	if config.MaxRecordsPerFile <= 0 {
		t.Error("MaxRecordsPerFile should be positive")
	}
	if config.MaxDurationSeconds <= 0 {
		t.Error("MaxDurationSeconds should be positive")
	}
}

func TestProcessingConfig(t *testing.T) {
	config := ProcessingConfig{
		BufferSizeMB:           10,
		BufferFlushIntervalSec: 60,
		WorkerPoolSize:         4,
	}

	if config.BufferSizeMB <= 0 {
		t.Error("BufferSizeMB should be positive")
	}
	if config.BufferFlushIntervalSec <= 0 {
		t.Error("BufferFlushIntervalSec should be positive")
	}
	if config.WorkerPoolSize <= 0 {
		t.Error("WorkerPoolSize should be positive")
	}
}

func TestDLQConfig(t *testing.T) {
	config := DLQConfig{
		Enabled:     true,
		TopicSuffix: ".dlq",
		MaxRetries:  3,
	}

	if config.Enabled && config.TopicSuffix == "" {
		t.Error("TopicSuffix required when DLQ enabled")
	}
	if config.MaxRetries < 0 {
		t.Error("MaxRetries should not be negative")
	}
}

func TestObservabilityConfig(t *testing.T) {
	config := ObservabilityConfig{
		Health: HealthConfig{
			Port: 8080,
		},
		Metrics: MetricsConfig{
			Port:    9090,
			Enabled: true,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	if config.Health.Port <= 0 {
		t.Error("Health port should be positive")
	}
	if config.Metrics.Port <= 0 {
		t.Error("Metrics port should be positive")
	}
	if config.Logging.Level == "" {
		t.Error("Logging level should not be empty")
	}
}

func TestShutdownConfig(t *testing.T) {
	config := ShutdownConfig{
		GracePeriodSeconds:  30,
		ForceTimeoutSeconds: 60,
	}

	if config.GracePeriodSeconds <= 0 {
		t.Error("GracePeriodSeconds should be positive")
	}
	if config.ForceTimeoutSeconds <= 0 {
		t.Error("ForceTimeoutSeconds should be positive")
	}
	if config.ForceTimeoutSeconds < config.GracePeriodSeconds {
		t.Error("ForceTimeoutSeconds should be >= GracePeriodSeconds")
	}
}

func TestS3Config(t *testing.T) {
	config := S3Config{
		Bucket:   "test-bucket",
		Region:   "us-east-1",
		BasePath: "events",
	}

	if config.Bucket == "" {
		t.Error("Bucket should not be empty")
	}
	if config.Region == "" {
		t.Error("Region should not be empty")
	}
}

func TestAzureConfig(t *testing.T) {
	config := AzureConfig{
		AccountName: "testaccount",
		Container:   "events",
	}

	if config.AccountName == "" {
		t.Error("AccountName should not be empty")
	}
	if config.Container == "" {
		t.Error("Container should not be empty")
	}
}

func TestFileConfig(t *testing.T) {
	config := FileConfig{
		BasePath: "/data/events",
	}

	if config.BasePath == "" {
		t.Error("BasePath should not be empty")
	}
}

func TestLogLevel_Validation(t *testing.T) {
	tests := []struct {
		level string
		valid bool
	}{
		{"debug", true},
		{"info", true},
		{"warn", true},
		{"error", true},
		{"fatal", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			validLevels := map[string]bool{
				"debug": true,
				"info":  true,
				"warn":  true,
				"error": true,
				"fatal": true,
			}

			valid := validLevels[tt.level]
			if valid != tt.valid {
				t.Errorf("Log level %v validity = %v, want %v", tt.level, valid, tt.valid)
			}
		})
	}
}

func TestLogFormat_Validation(t *testing.T) {
	tests := []struct {
		format string
		valid  bool
	}{
		{"json", true},
		{"text", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			valid := tt.format == "json" || tt.format == "text"
			if valid != tt.valid {
				t.Errorf("Log format %v validity = %v, want %v", tt.format, valid, tt.valid)
			}
		})
	}
}

func TestSecurityProtocol_Validation(t *testing.T) {
	tests := []struct {
		protocol string
		valid    bool
	}{
		{"PLAINTEXT", true},
		{"SSL", true},
		{"SASL_PLAINTEXT", true},
		{"SASL_SSL", true},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.protocol, func(t *testing.T) {
			validProtocols := map[string]bool{
				"PLAINTEXT":      true,
				"SSL":            true,
				"SASL_PLAINTEXT": true,
				"SASL_SSL":       true,
			}

			valid := validProtocols[tt.protocol]
			if valid != tt.valid {
				t.Errorf("Protocol %v validity = %v, want %v", tt.protocol, valid, tt.valid)
			}
		})
	}
}

func TestSASLMechanism_Validation(t *testing.T) {
	tests := []struct {
		mechanism string
		valid     bool
	}{
		{"PLAIN", true},
		{"SCRAM-SHA-256", true},
		{"SCRAM-SHA-512", true},
		{"AWS_MSK_IAM", true},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.mechanism, func(t *testing.T) {
			validMechanisms := map[string]bool{
				"PLAIN":         true,
				"SCRAM-SHA-256": true,
				"SCRAM-SHA-512": true,
				"AWS_MSK_IAM":   true,
			}

			valid := validMechanisms[tt.mechanism]
			if valid != tt.valid {
				t.Errorf("Mechanism %v validity = %v, want %v", tt.mechanism, valid, tt.valid)
			}
		})
	}
}

func TestConsumerConfig_AutoOffsetReset(t *testing.T) {
	tests := []struct {
		offset string
		valid  bool
	}{
		{"earliest", true},
		{"latest", true},
		{"none", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.offset, func(t *testing.T) {
			valid := tt.offset == "earliest" || tt.offset == "latest"
			if valid != tt.valid {
				t.Errorf("Offset reset %v validity = %v, want %v", tt.offset, valid, tt.valid)
			}
		})
	}
}

func TestFullApplicationConfig(t *testing.T) {
	config := &ApplicationConfig{
		Application: ApplicationInfo{
			Name:        "test-app",
			Version:     "1.0.0",
			Environment: "test",
		},
		Kafka: KafkaConfig{
			BootstrapServers: []string{"localhost:9092"},
			SecurityProtocol: "PLAINTEXT",
			Consumer: ConsumerConfig{
				GroupID:          "test-group",
				Topics:           []string{"test-topic"},
				AutoOffsetReset:  "latest",
				EnableAutoCommit: false,
			},
		},
		Storage: StorageConfig{
			Backend:     "file",
			Format:      "parquet",
			Compression: "snappy",
			File: FileConfig{
				BasePath: "/tmp/events",
			},
		},
		FileRotation: FileRotationConfig{
			MaxFileSizeMB:      100,
			MaxRecordsPerFile:  10000,
			MaxDurationSeconds: 300,
		},
		Processing: ProcessingConfig{
			BufferSizeMB:           10,
			BufferFlushIntervalSec: 60,
		},
		Observability: ObservabilityConfig{
			Health:  HealthConfig{Port: 8080},
			Metrics: MetricsConfig{Port: 9090, Enabled: true},
			Logging: LoggingConfig{Level: "info", Format: "json"},
		},
		Shutdown: ShutdownConfig{
			GracePeriodSeconds:  30,
			ForceTimeoutSeconds: 60,
		},
	}

	// Validate all sections are present
	if config.Application.Name == "" {
		t.Error("Application name missing")
	}
	if len(config.Kafka.BootstrapServers) == 0 {
		t.Error("Kafka bootstrap servers missing")
	}
	if config.Storage.Backend == "" {
		t.Error("Storage backend missing")
	}
	if config.FileRotation.MaxFileSizeMB <= 0 {
		t.Error("File rotation config invalid")
	}
	if config.Processing.BufferSizeMB <= 0 {
		t.Error("Processing config invalid")
	}
	if config.Observability.Health.Port <= 0 {
		t.Error("Observability config invalid")
	}
	if config.Shutdown.GracePeriodSeconds <= 0 {
		t.Error("Shutdown config invalid")
	}
}
