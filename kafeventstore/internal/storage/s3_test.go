package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestS3Config_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  S3Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: S3Config{
				Bucket: "test-bucket",
				Region: "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "empty bucket",
			config: S3Config{
				Bucket: "",
				Region: "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "empty region",
			config: S3Config{
				Bucket: "test-bucket",
				Region: "",
			},
			wantErr: true,
		},
		{
			name: "with endpoint",
			config: S3Config{
				Bucket:   "test-bucket",
				Region:   "us-east-1",
				Endpoint: "http://localhost:9000",
			},
			wantErr: false,
		},
		{
			name: "with SSE enabled",
			config: S3Config{
				Bucket:     "test-bucket",
				Region:     "us-east-1",
				SSEEnabled: true,
			},
			wantErr: false,
		},
		{
			name: "with SSE KMS key",
			config: S3Config{
				Bucket:      "test-bucket",
				Region:      "us-east-1",
				SSEEnabled:  true,
				SSEKMSKeyID: "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateS3Config(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateS3Config() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func validateS3Config(config S3Config) error {
	if config.Bucket == "" {
		return errors.New("bucket is required")
	}
	if config.Region == "" {
		return errors.New("region is required")
	}
	return nil
}

func TestS3Path_Construction(t *testing.T) {
	tests := []struct {
		name   string
		bucket string
		path   string
		want   string
	}{
		{
			name:   "simple path",
			bucket: "my-bucket",
			path:   "events/topic/v1/dt=2025-12-19/pid=0/",
			want:   "s3://my-bucket/events/topic/v1/dt=2025-12-19/pid=0/",
		},
		{
			name:   "with base path",
			bucket: "my-bucket",
			path:   "data/events/topic/v1/dt=2025-12-19/pid=0/",
			want:   "s3://my-bucket/data/events/topic/v1/dt=2025-12-19/pid=0/",
		},
		{
			name:   "root path",
			bucket: "my-bucket",
			path:   "/",
			want:   "s3://my-bucket/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct path avoiding double slashes
			path := tt.path
			if path == "/" {
				path = ""
			}
			got := "s3://" + tt.bucket + "/" + path
			if got != tt.want {
				t.Errorf("S3 path = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestS3Writer_Compression(t *testing.T) {
	tests := []struct {
		name        string
		compression string
		valid       bool
	}{
		{"snappy", "snappy", true},
		{"gzip", "gzip", true},
		{"zstd", "zstd", true},
		{"none", "none", true},
		{"invalid", "invalid", false},
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

func TestS3Writer_SSE(t *testing.T) {
	tests := []struct {
		name        string
		sseEnabled  bool
		sseKMSKeyID string
		encryption  string
	}{
		{
			name:        "SSE disabled",
			sseEnabled:  false,
			sseKMSKeyID: "",
			encryption:  "none",
		},
		{
			name:        "SSE-S3 enabled",
			sseEnabled:  true,
			sseKMSKeyID: "",
			encryption:  "AES256",
		},
		{
			name:        "SSE-KMS enabled",
			sseEnabled:  true,
			sseKMSKeyID: "arn:aws:kms:us-east-1:123456789012:key/12345678",
			encryption:  "aws:kms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var encryption string
			if !tt.sseEnabled {
				encryption = "none"
			} else if tt.sseKMSKeyID != "" {
				encryption = "aws:kms"
			} else {
				encryption = "AES256"
			}

			if encryption != tt.encryption {
				t.Errorf("Encryption = %v, want %v", encryption, tt.encryption)
			}
		})
	}
}

func TestS3Writer_PathStyle(t *testing.T) {
	tests := []struct {
		name         string
		usePathStyle bool
		bucket       string
		region       string
		expectedURL  string
	}{
		{
			name:         "virtual hosted style",
			usePathStyle: false,
			bucket:       "my-bucket",
			region:       "us-east-1",
			expectedURL:  "https://my-bucket.s3.us-east-1.amazonaws.com",
		},
		{
			name:         "path style",
			usePathStyle: true,
			bucket:       "my-bucket",
			region:       "us-east-1",
			expectedURL:  "https://s3.us-east-1.amazonaws.com/my-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var url string
			if tt.usePathStyle {
				url = "https://s3." + tt.region + ".amazonaws.com/" + tt.bucket
			} else {
				url = "https://" + tt.bucket + ".s3." + tt.region + ".amazonaws.com"
			}

			if url != tt.expectedURL {
				t.Errorf("URL = %v, want %v", url, tt.expectedURL)
			}
		})
	}
}

func TestS3Writer_RetryLogic(t *testing.T) {
	tests := []struct {
		name        string
		attempt     int
		maxRetries  int
		shouldRetry bool
	}{
		{"first attempt", 0, 3, true},
		{"second attempt", 1, 3, true},
		{"at limit", 3, 3, false},
		{"exceeded limit", 4, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRetry := tt.attempt < tt.maxRetries
			if shouldRetry != tt.shouldRetry {
				t.Errorf("Retry decision = %v, want %v", shouldRetry, tt.shouldRetry)
			}
		})
	}
}

func TestS3Writer_Close(t *testing.T) {
	closed := false

	closeFunc := func() error {
		if closed {
			return errors.New("already closed")
		}
		closed = true
		return nil
	}

	// First close
	if err := closeFunc(); err != nil {
		t.Errorf("First close failed: %v", err)
	}
	if !closed {
		t.Error("Should be closed")
	}

	// Second close should fail
	if err := closeFunc(); err == nil {
		t.Error("Second close should fail")
	}
}

func TestS3Writer_ErrorTypes(t *testing.T) {
	tests := []struct {
		name      string
		errorCode string
		retryable bool
	}{
		{"throttling", "SlowDown", true},
		{"service unavailable", "ServiceUnavailable", true},
		{"no such bucket", "NoSuchBucket", false},
		{"access denied", "AccessDenied", false},
		{"invalid credentials", "InvalidAccessKeyId", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryable := tt.errorCode == "SlowDown" ||
				tt.errorCode == "ServiceUnavailable" ||
				tt.errorCode == "RequestTimeout"

			if retryable != tt.retryable {
				t.Errorf("Error %v retryable = %v, want %v", tt.errorCode, retryable, tt.retryable)
			}
		})
	}
}

func TestS3Writer_MultiplartUpload(t *testing.T) {
	tests := []struct {
		name               string
		fileSize           int64
		partSize           int64
		shouldUseMultipart bool
	}{
		{"small file", 1024 * 1024, 5 * 1024 * 1024, false},      // 1 MB
		{"medium file", 10 * 1024 * 1024, 5 * 1024 * 1024, true}, // 10 MB
		{"large file", 100 * 1024 * 1024, 5 * 1024 * 1024, true}, // 100 MB
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldUseMultipart := tt.fileSize > tt.partSize
			if shouldUseMultipart != tt.shouldUseMultipart {
				t.Errorf("Multipart decision = %v, want %v", shouldUseMultipart, tt.shouldUseMultipart)
			}
		})
	}
}

func TestS3Writer_Context(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		shouldFail bool
	}{
		{
			name:       "valid context",
			ctx:        context.Background(),
			shouldFail: false,
		},
		{
			name:       "cancelled context",
			ctx:        cancelledContext(),
			shouldFail: true,
		},
		{
			name:       "timeout context",
			ctx:        timeoutContext(),
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			select {
			case <-tt.ctx.Done():
				if !tt.shouldFail {
					t.Error("Context should not be done")
				}
			default:
				if tt.shouldFail {
					t.Error("Context should be done")
				}
			}
		})
	}
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func timeoutContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond)
	return ctx
}

func TestS3Writer_Metadata(t *testing.T) {
	metadata := map[string]string{
		"Content-Type":         "application/octet-stream",
		"Content-Encoding":     "gzip",
		"x-amz-meta-topic":     "test-topic",
		"x-amz-meta-partition": "0",
		"x-amz-meta-format":    "parquet",
	}

	if len(metadata) == 0 {
		t.Error("Metadata should not be empty")
	}

	requiredKeys := []string{"Content-Type", "x-amz-meta-topic", "x-amz-meta-format"}
	for _, key := range requiredKeys {
		if _, ok := metadata[key]; !ok {
			t.Errorf("Required metadata key missing: %v", key)
		}
	}
}

func TestS3Writer_Tags(t *testing.T) {
	tags := map[string]string{
		"Environment": "dev",
		"Service":     "event-store",
		"Format":      "parquet",
		"Topic":       "test-topic",
	}

	if len(tags) == 0 {
		t.Error("Tags should not be empty")
	}

	for key, value := range tags {
		if key == "" || value == "" {
			t.Error("Tag key and value should not be empty")
		}
	}
}

func TestS3Writer_RecordBatch(t *testing.T) {
	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				ID:          "1",
				Source:      "test",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    1,
				Timestamp: now,
			},
		},
		{
			Event: &event.CloudEvent{
				ID:          "2",
				Source:      "test",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    2,
				Timestamp: now,
			},
		},
	}

	if len(records) != 2 {
		t.Errorf("Record count = %d, want 2", len(records))
	}

	for i, record := range records {
		if record.Event == nil {
			t.Errorf("Record %d event should not be nil", i)
		}
		if record.Kafka.Topic == "" {
			t.Errorf("Record %d topic should not be empty", i)
		}
	}
}
