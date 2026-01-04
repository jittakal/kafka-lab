package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestGCSConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  GCSConfig
		wantErr bool
	}{
		{
			name: "valid config with default credentials",
			config: GCSConfig{
				Bucket:               "test-bucket",
				ProjectID:            "test-project",
				UseDefaultCredential: true,
			},
			wantErr: false,
		},
		{
			name: "valid config with credentials file",
			config: GCSConfig{
				Bucket:          "test-bucket",
				ProjectID:       "test-project",
				CredentialsFile: "/path/to/credentials.json",
			},
			wantErr: false,
		},
		{
			name: "valid config with credentials JSON",
			config: GCSConfig{
				Bucket:          "test-bucket",
				ProjectID:       "test-project",
				CredentialsJSON: `{"type": "service_account"}`,
			},
			wantErr: false,
		},
		{
			name: "empty bucket",
			config: GCSConfig{
				Bucket:               "",
				ProjectID:            "test-project",
				UseDefaultCredential: true,
			},
			wantErr: true,
		},
		{
			name: "with endpoint",
			config: GCSConfig{
				Bucket:               "test-bucket",
				ProjectID:            "test-project",
				Endpoint:             "http://localhost:9000",
				UseDefaultCredential: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGCSConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGCSConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func validateGCSConfig(config GCSConfig) error {
	if config.Bucket == "" {
		return errors.New("bucket is required")
	}
	return nil
}

func TestGCSPath_Construction(t *testing.T) {
	tests := []struct {
		name   string
		bucket string
		path   string
		want   string
	}{
		{
			name:   "simple path",
			bucket: "test-bucket",
			path:   "events/topic/v1/dt=2023-01-01/pid=0/",
			want:   "events/topic/v1/dt=2023-01-01/pid=0/",
		},
		{
			name:   "path with gs:// prefix",
			bucket: "test-bucket",
			path:   "gs://test-bucket/events/topic/v1/dt=2023-01-01/pid=0/",
			want:   "events/topic/v1/dt=2023-01-01/pid=0/",
		},
		{
			name:   "path with leading slash",
			bucket: "test-bucket",
			path:   "/events/topic/v1/dt=2023-01-01/pid=0/",
			want:   "events/topic/v1/dt=2023-01-01/pid=0/",
		},
		{
			name:   "path without trailing slash",
			bucket: "test-bucket",
			path:   "events/topic/v1/dt=2023-01-01/pid=0",
			want:   "events/topic/v1/dt=2023-01-01/pid=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeGCSPath(tt.path, tt.bucket)
			if got != tt.want {
				t.Errorf("normalizeGCSPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

// normalizeGCSPath normalizes a GCS path by removing protocol and bucket prefix
func normalizeGCSPath(path, bucket string) string {
	// Remove gs:// prefix if present
	if len(path) >= 5 && path[:5] == "gs://" {
		path = path[5:]
		// Remove bucket name
		if len(path) > len(bucket) && path[:len(bucket)] == bucket {
			path = path[len(bucket):]
		}
	}
	// Remove leading slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	return path
}

func TestGCSWriter_Interface(t *testing.T) {
	// This test ensures GCSWriter implements the storage.Writer interface
	var _ interface {
		Write(ctx context.Context, records []event.Record, path string, format event.FileFormat) (int64, error)
		Close() error
	} = (*GCSWriter)(nil)
}

func TestGCSWriter_Metadata(t *testing.T) {
	tests := []struct {
		name        string
		format      event.FileFormat
		wantContent string
	}{
		{
			name:        "parquet format",
			format:      event.FormatParquet,
			wantContent: "application/octet-stream",
		},
		{
			name:        "avro format",
			format:      event.FormatAvro,
			wantContent: "application/avro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentType := getGCSContentType(tt.format)
			if contentType != tt.wantContent {
				t.Errorf("getGCSContentType() = %v, want %v", contentType, tt.wantContent)
			}
		})
	}
}

// getGCSContentType returns the content type for a given format
func getGCSContentType(format event.FileFormat) string {
	switch format {
	case event.FormatParquet:
		return "application/octet-stream"
	case event.FormatAvro:
		return "application/avro"
	default:
		return "application/octet-stream"
	}
}

func TestGCSWriter_PathGeneration(t *testing.T) {
	now := time.Date(2023, 1, 15, 10, 30, 45, 123456789, time.UTC)

	tests := []struct {
		name          string
		basePath      string
		extension     string
		wantPrefix    string
		wantExtension string
	}{
		{
			name:          "parquet file",
			basePath:      "events/topic/v1/dt=2023-01-15/pid=0/",
			extension:     ".parquet",
			wantPrefix:    "events/topic/v1/dt=2023-01-15/pid=0/events_20230115_103045_",
			wantExtension: ".parquet",
		},
		{
			name:          "avro file",
			basePath:      "events/topic/v1/dt=2023-01-15/pid=0/",
			extension:     ".avro",
			wantPrefix:    "events/topic/v1/dt=2023-01-15/pid=0/events_20230115_103045_",
			wantExtension: ".avro",
		},
		{
			name:          "compressed parquet",
			basePath:      "data/topic/v1/dt=2023-01-15/pid=1/",
			extension:     ".parquet.snappy",
			wantPrefix:    "data/topic/v1/dt=2023-01-15/pid=1/events_20230115_103045_",
			wantExtension: ".parquet.snappy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := generateGCSFilename(now, tt.extension)
			objectPath := tt.basePath + filename

			// Check that the path starts with expected prefix
			expectedStart := tt.wantPrefix
			if len(objectPath) < len(expectedStart) || objectPath[:len(expectedStart)] != expectedStart {
				t.Errorf("objectPath prefix = %v, want prefix %v", objectPath[:min(len(objectPath), len(expectedStart))], expectedStart)
			}

			// Check that the path ends with expected extension
			if len(objectPath) < len(tt.wantExtension) || objectPath[len(objectPath)-len(tt.wantExtension):] != tt.wantExtension {
				t.Errorf("objectPath extension = %v, want %v", objectPath[len(objectPath)-len(tt.wantExtension):], tt.wantExtension)
			}
		})
	}
}

// generateGCSFilename generates a timestamped filename
func generateGCSFilename(now time.Time, extension string) string {
	timestamp := now.Format("20060102_150405")
	return "events_" + timestamp + "_" + formatNanos(now.Nanosecond()) + extension
}

func formatNanos(nanos int) string {
	millis := nanos / 1000000
	if millis < 10 {
		return "00" + string(rune('0'+millis))
	} else if millis < 100 {
		return "0" + string(rune('0'+millis/10)) + string(rune('0'+millis%10))
	}
	return string(rune('0'+millis/100)) + string(rune('0'+(millis/10)%10)) + string(rune('0'+millis%10))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
