package storage

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
)

// mockMetricsCollector implements MetricsCollector for testing
type mockMetricsCollector struct {
	filesWritten       int
	fileSizes          []float64
	storageDurations   []float64
	storageErrors      int
	lastFileStatus     string
	lastTopic          string
	lastPartition      int32
	lastFormat         string
	lastErrorBackend   string
	lastErrorOperation string
}

func (m *mockMetricsCollector) IncFilesWritten(topic string, partition int32, format string, status string) {
	m.filesWritten++
	m.lastTopic = topic
	m.lastPartition = partition
	m.lastFormat = format
	m.lastFileStatus = status
}

func (m *mockMetricsCollector) ObserveFileSize(topic string, partition int32, format string, size float64) {
	m.fileSizes = append(m.fileSizes, size)
}

func (m *mockMetricsCollector) ObserveStorageWriteDuration(topic string, partition int32, duration float64) {
	m.storageDurations = append(m.storageDurations, duration)
}

func (m *mockMetricsCollector) IncStorageErrors(backend string, operation string) {
	m.storageErrors++
	m.lastErrorBackend = backend
	m.lastErrorOperation = operation
}

func TestNewFileWriter(t *testing.T) {
	tests := []struct {
		name        string
		config      FileConfig
		format      event.FileFormat
		compression string
		wantErr     bool
	}{
		{
			name: "valid parquet config",
			config: FileConfig{
				BasePath: filepath.Join(os.TempDir(), "test-file-writer-parquet"),
			},
			format:      event.FormatParquet,
			compression: "snappy",
			wantErr:     false,
		},
		{
			name: "valid avro config",
			config: FileConfig{
				BasePath: filepath.Join(os.TempDir(), "test-file-writer-avro"),
			},
			format:      event.FormatAvro,
			compression: "snappy",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.RemoveAll(tt.config.BasePath)

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			metrics := &mockMetricsCollector{}

			writer, err := NewFileWriter(tt.config, tt.format, tt.compression, logger, metrics)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if writer == nil {
					t.Fatal("expected non-nil writer")
				}
				if writer.basePath != tt.config.BasePath {
					t.Errorf("basePath = %v, want %v", writer.basePath, tt.config.BasePath)
				}
			}
		})
	}
}

func TestFileWriter_Write(t *testing.T) {
	basePath := filepath.Join(os.TempDir(), "test-file-writer-write")
	defer os.RemoveAll(basePath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	metrics := &mockMetricsCollector{}

	writer, err := NewFileWriter(
		FileConfig{BasePath: basePath},
		event.FormatParquet,
		"snappy",
		logger,
		metrics,
	)
	if err != nil {
		t.Fatalf("NewFileWriter() failed: %v", err)
	}

	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				SpecVersion: "1.0",
				Type:        "test.event",
				Source:      "test-source",
				ID:          "test-id-1",
				Time:        &now,
				Data:        []byte(`{"message": "test1"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    100,
				Timestamp: now,
			},
			Offset:      100,
			ProcessedAt: now,
		},
		{
			Event: &event.CloudEvent{
				SpecVersion: "1.0",
				Type:        "test.event",
				Source:      "test-source",
				ID:          "test-id-2",
				Time:        &now,
				Data:        []byte(`{"message": "test2"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    101,
				Timestamp: now,
			},
			Offset:      101,
			ProcessedAt: now,
		},
	}

	tests := []struct {
		name    string
		records []event.Record
		path    string
		format  event.FileFormat
		wantErr bool
	}{
		{
			name:    "successful write",
			records: records,
			path:    "test-topic/v1/dt=2024-01-01/partition=0/file",
			format:  event.FormatParquet,
			wantErr: false,
		},
		{
			name:    "empty records",
			records: []event.Record{},
			path:    "test-topic/empty",
			format:  event.FormatParquet,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			size, err := writer.Write(ctx, tt.records, tt.path, tt.format)

			if (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if size <= 0 {
					t.Errorf("Write() size = %v, want > 0", size)
				}

				// Verify directory exists (file has timestamped name)
				dirPath := filepath.Join(basePath, tt.path)
				if _, err := os.Stat(dirPath); os.IsNotExist(err) {
					t.Errorf("expected directory at %s", dirPath)
				}

				// Verify at least one file exists in the directory
				entries, err := os.ReadDir(dirPath)
				if err != nil || len(entries) == 0 {
					t.Errorf("expected files in directory %s", dirPath)
				}

				// Verify metrics were updated
				if metrics.filesWritten != 1 {
					t.Errorf("filesWritten = %d, want 1", metrics.filesWritten)
				}
				if metrics.lastFileStatus != "success" {
					t.Errorf("lastFileStatus = %s, want success", metrics.lastFileStatus)
				}
				if len(metrics.fileSizes) != 1 {
					t.Errorf("len(fileSizes) = %d, want 1", len(metrics.fileSizes))
				}
			}
		})
	}
}

func TestFileWriter_Close(t *testing.T) {
	basePath := filepath.Join(os.TempDir(), "test-file-writer-close")
	defer os.RemoveAll(basePath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	metrics := &mockMetricsCollector{}

	writer, err := NewFileWriter(
		FileConfig{BasePath: basePath},
		event.FormatParquet,
		"snappy",
		logger,
		metrics,
	)
	if err != nil {
		t.Fatalf("NewFileWriter() failed: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}
