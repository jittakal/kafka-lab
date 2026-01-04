package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jittakal/kafeventstore/internal/config/dto"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("expected non-nil loader")
	}
	if loader.v == nil {
		t.Fatal("expected non-nil viper instance")
	}
}

func TestLoader_LoadWithValidConfig(t *testing.T) {
	// Create a temporary config file
	tempDir := os.TempDir()
	configFile := filepath.Join(tempDir, "test-config.yaml")
	defer os.Remove(configFile)

	configContent := `
application:
  name: test-app
  version: 1.0.0

kafka:
  bootstrap_servers:
    - localhost:9092
  consumer:
    group_id: test-group
    topics:
      - test-topic

storage:
  backend: file
  format: parquet
  file:
    base_path: /tmp/test
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	loader := NewLoader()
	config, err := loader.Load(configFile)

	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if config == nil {
		t.Fatal("expected non-nil config")
	}

	// Verify loaded values
	if config.Application.Name != "test-app" {
		t.Errorf("Application.Name = %s, want test-app", config.Application.Name)
	}
	if config.Kafka.Consumer.GroupID != "test-group" {
		t.Errorf("Kafka.Consumer.GroupID = %s, want test-group", config.Kafka.Consumer.GroupID)
	}
	if len(config.Kafka.Consumer.Topics) != 1 || config.Kafka.Consumer.Topics[0] != "test-topic" {
		t.Errorf("Kafka.Consumer.Topics = %v, want [test-topic]", config.Kafka.Consumer.Topics)
	}
}

func TestLoader_LoadWithMissingFile(t *testing.T) {
	loader := NewLoader()

	// Loading with non-existent file should succeed (will use defaults + env vars)
	config, err := loader.Load("/nonexistent/config.yaml")
	if err == nil {
		// This might succeed with default values, so we need to check validation
		if config != nil {
			// Config loaded but may fail validation if required fields are missing
			t.Log("Config loaded with defaults, validation may fail for required fields")
		}
	}
}

func TestLoader_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *dto.ApplicationConfig
		wantErr bool
	}{
		{
			name: "valid file backend config",
			config: &dto.ApplicationConfig{
				Kafka: dto.KafkaConfig{
					BootstrapServers: []string{"localhost:9092"},
					Consumer: dto.ConsumerConfig{
						GroupID: "test-group",
						Topics:  []string{"test-topic"},
					},
				},
				Storage: dto.StorageConfig{
					Backend: "file",
					Format:  "parquet",
					File: dto.FileConfig{
						BasePath: "/tmp/test",
					},
				},
				FileRotation: dto.FileRotationConfig{
					Strategy: "any",
				},
				Observability: dto.ObservabilityConfig{
					Metrics: dto.MetricsConfig{
						Port: 9090,
					},
					Health: dto.HealthConfig{
						Port: 8080,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing bootstrap servers",
			config: &dto.ApplicationConfig{
				Kafka: dto.KafkaConfig{
					BootstrapServers: []string{},
					Consumer: dto.ConsumerConfig{
						GroupID: "test-group",
						Topics:  []string{"test-topic"},
					},
				},
				Storage: dto.StorageConfig{
					Backend: "file",
					Format:  "parquet",
					File: dto.FileConfig{
						BasePath: "/tmp/test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing consumer topics",
			config: &dto.ApplicationConfig{
				Kafka: dto.KafkaConfig{
					BootstrapServers: []string{"localhost:9092"},
					Consumer: dto.ConsumerConfig{
						GroupID: "test-group",
						Topics:  []string{},
					},
				},
				Storage: dto.StorageConfig{
					Backend: "file",
					Format:  "parquet",
					File: dto.FileConfig{
						BasePath: "/tmp/test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing consumer group id",
			config: &dto.ApplicationConfig{
				Kafka: dto.KafkaConfig{
					BootstrapServers: []string{"localhost:9092"},
					Consumer: dto.ConsumerConfig{
						GroupID: "",
						Topics:  []string{"test-topic"},
					},
				},
				Storage: dto.StorageConfig{
					Backend: "file",
					Format:  "parquet",
					File: dto.FileConfig{
						BasePath: "/tmp/test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "s3 backend missing bucket",
			config: &dto.ApplicationConfig{
				Kafka: dto.KafkaConfig{
					BootstrapServers: []string{"localhost:9092"},
					Consumer: dto.ConsumerConfig{
						GroupID: "test-group",
						Topics:  []string{"test-topic"},
					},
				},
				Storage: dto.StorageConfig{
					Backend: "s3",
					Format:  "parquet",
					S3: dto.S3Config{
						Region: "us-east-1",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "azure backend missing account name",
			config: &dto.ApplicationConfig{
				Kafka: dto.KafkaConfig{
					BootstrapServers: []string{"localhost:9092"},
					Consumer: dto.ConsumerConfig{
						GroupID: "test-group",
						Topics:  []string{"test-topic"},
					},
				},
				Storage: dto.StorageConfig{
					Backend: "azure",
					Format:  "parquet",
					Azure: dto.AzureConfig{
						Container: "test-container",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "unsupported storage backend",
			config: &dto.ApplicationConfig{
				Kafka: dto.KafkaConfig{
					BootstrapServers: []string{"localhost:9092"},
					Consumer: dto.ConsumerConfig{
						GroupID: "test-group",
						Topics:  []string{"test-topic"},
					},
				},
				Storage: dto.StorageConfig{
					Backend: "unsupported",
					Format:  "parquet",
				},
			},
			wantErr: true,
		},
		{
			name: "unsupported storage format",
			config: &dto.ApplicationConfig{
				Kafka: dto.KafkaConfig{
					BootstrapServers: []string{"localhost:9092"},
					Consumer: dto.ConsumerConfig{
						GroupID: "test-group",
						Topics:  []string{"test-topic"},
					},
				},
				Storage: dto.StorageConfig{
					Backend: "file",
					Format:  "unsupported",
					File: dto.FileConfig{
						BasePath: "/tmp/test",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid metrics port",
			config: &dto.ApplicationConfig{
				Kafka: dto.KafkaConfig{
					BootstrapServers: []string{"localhost:9092"},
					Consumer: dto.ConsumerConfig{
						GroupID: "test-group",
						Topics:  []string{"test-topic"},
					},
				},
				Storage: dto.StorageConfig{
					Backend: "file",
					Format:  "parquet",
					File: dto.FileConfig{
						BasePath: "/tmp/test",
					},
				},
				FileRotation: dto.FileRotationConfig{
					Strategy: "any",
				},
				Observability: dto.ObservabilityConfig{
					Metrics: dto.MetricsConfig{
						Port: 70000, // Invalid port
					},
					Health: dto.HealthConfig{
						Port: 8080,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader()
			err := loader.Validate(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoader_setDefaults(t *testing.T) {
	loader := NewLoader()
	loader.setDefaults()

	// Verify some key defaults are set
	if loader.v.GetString("application.name") != "kafka-event-store" {
		t.Error("default application.name not set correctly")
	}
	if loader.v.GetString("storage.backend") != "file" {
		t.Error("default storage.backend not set correctly")
	}
	if loader.v.GetString("storage.format") != "parquet" {
		t.Error("default storage.format not set correctly")
	}
}
