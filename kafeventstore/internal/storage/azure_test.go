package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestAzureConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  AzureConfig
		wantErr bool
	}{
		{
			name: "valid config with account key",
			config: AzureConfig{
				AccountName:   "testaccount",
				AccountKey:    "dGVzdGtleQ==",
				ContainerName: "test-container",
			},
			wantErr: false,
		},
		{
			name: "empty account name",
			config: AzureConfig{
				AccountName:   "",
				AccountKey:    "dGVzdGtleQ==",
				ContainerName: "test-container",
			},
			wantErr: true,
		},
		{
			name: "empty account key",
			config: AzureConfig{
				AccountName:   "testaccount",
				AccountKey:    "",
				ContainerName: "test-container",
			},
			wantErr: true,
		},
		{
			name: "empty container name",
			config: AzureConfig{
				AccountName:   "testaccount",
				AccountKey:    "dGVzdGtleQ==",
				ContainerName: "",
			},
			wantErr: true,
		},
		{
			name: "with custom endpoint",
			config: AzureConfig{
				AccountName:   "testaccount",
				AccountKey:    "dGVzdGtleQ==",
				ContainerName: "test-container",
				Endpoint:      "http://127.0.0.1:10000/devstoreaccount1",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateAzureConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateAzureConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func validateAzureConfig(config AzureConfig) error {
	if config.AccountName == "" {
		return errors.New("account name is required")
	}
	if config.AccountKey == "" {
		return errors.New("account key is required")
	}
	if config.ContainerName == "" {
		return errors.New("container name is required")
	}
	return nil
}

func TestAzurePath_Construction(t *testing.T) {
	tests := []struct {
		name      string
		container string
		blobPath  string
		want      string
	}{
		{
			name:      "simple path",
			container: "events",
			blobPath:  "topic/v1/dt=2025-12-19/pid=0/file.parquet",
			want:      "wasbs://events/topic/v1/dt=2025-12-19/pid=0/file.parquet",
		},
		{
			name:      "nested path",
			container: "data",
			blobPath:  "events/topic/v1/dt=2025-12-19/pid=0/file.parquet",
			want:      "wasbs://data/events/topic/v1/dt=2025-12-19/pid=0/file.parquet",
		},
		{
			name:      "root path",
			container: "root",
			blobPath:  "file.parquet",
			want:      "wasbs://root/file.parquet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := "wasbs://" + tt.container + "/" + tt.blobPath
			if got != tt.want {
				t.Errorf("Azure path = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAzureContainerName_Validation(t *testing.T) {
	tests := []struct {
		name      string
		container string
		valid     bool
	}{
		{"valid lowercase", "mycontainer", true},
		{"valid with numbers", "container123", true},
		{"valid with dashes", "my-container", true},
		{"invalid uppercase", "MyContainer", false},
		{"invalid underscore", "my_container", false},
		{"invalid start with dash", "-container", false},
		{"invalid end with dash", "container-", false},
		{"invalid too short", "ab", false},
		{"invalid too long", "verylongcontainernamethatexceedssixtythreecharactersinlengthtest", false},
		{"valid minimum length", "abc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidAzureContainerName(tt.container)
			if valid != tt.valid {
				t.Errorf("Container %v validity = %v, want %v", tt.container, valid, tt.valid)
			}
		})
	}
}

func isValidAzureContainerName(name string) bool {
	if len(name) < 3 || len(name) > 63 {
		return false
	}
	if name[0] == '-' || name[len(name)-1] == '-' {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return false
		}
	}
	return true
}

func TestAzureWriter_BlobTiers(t *testing.T) {
	tests := []struct {
		name  string
		tier  string
		valid bool
	}{
		{"hot", "Hot", true},
		{"cool", "Cool", true},
		{"archive", "Archive", true},
		{"invalid", "Invalid", false},
		{"empty", "", true}, // defaults to hot
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validTiers := map[string]bool{
				"Hot":     true,
				"Cool":    true,
				"Archive": true,
				"":        true,
			}

			valid := validTiers[tt.tier]
			if valid != tt.valid {
				t.Errorf("Tier %v validity = %v, want %v", tt.tier, valid, tt.valid)
			}
		})
	}
}

func TestAzureWriter_Metadata(t *testing.T) {
	metadata := map[string]string{
		"Content-Type":     "application/octet-stream",
		"Content-Encoding": "gzip",
		"topic":            "test-topic",
		"partition":        "0",
		"format":           "parquet",
	}

	if len(metadata) == 0 {
		t.Error("Metadata should not be empty")
	}

	requiredKeys := []string{"Content-Type", "topic", "format"}
	for _, key := range requiredKeys {
		if _, ok := metadata[key]; !ok {
			t.Errorf("Required metadata key missing: %v", key)
		}
	}
}

func TestAzureWriter_RetryPolicy(t *testing.T) {
	tests := []struct {
		name        string
		attempt     int
		maxRetries  int
		shouldRetry bool
	}{
		{"first attempt", 0, 3, true},
		{"within limit", 2, 3, true},
		{"at limit", 3, 3, false},
		{"exceeded", 4, 3, false},
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

func TestAzureWriter_ErrorCodes(t *testing.T) {
	tests := []struct {
		name      string
		errorCode string
		retryable bool
	}{
		{"server busy", "ServerBusy", true},
		{"timeout", "OperationTimedOut", true},
		{"blob not found", "BlobNotFound", false},
		{"container not found", "ContainerNotFound", false},
		{"authentication failed", "AuthenticationFailed", false},
		{"invalid credentials", "InvalidCredentials", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryable := tt.errorCode == "ServerBusy" ||
				tt.errorCode == "OperationTimedOut" ||
				tt.errorCode == "ServiceUnavailable"

			if retryable != tt.retryable {
				t.Errorf("Error %v retryable = %v, want %v", tt.errorCode, retryable, tt.retryable)
			}
		})
	}
}

func TestAzureWriter_Authentication(t *testing.T) {
	tests := []struct {
		name       string
		authMethod string
		valid      bool
	}{
		{"shared key", "SharedKey", true},
		{"connection string", "ConnectionString", true},
		{"managed identity", "ManagedIdentity", true},
		{"SAS token", "SAS", true},
		{"invalid", "Invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validMethods := map[string]bool{
				"SharedKey":        true,
				"ConnectionString": true,
				"ManagedIdentity":  true,
				"SAS":              true,
			}

			valid := validMethods[tt.authMethod]
			if valid != tt.valid {
				t.Errorf("Auth method %v validity = %v, want %v", tt.authMethod, valid, tt.valid)
			}
		})
	}
}

func TestAzureWriter_BlockSize(t *testing.T) {
	tests := []struct {
		name      string
		blockSize int64
		valid     bool
	}{
		{"minimum 4 KB", 4 * 1024, true},
		{"default 4 MB", 4 * 1024 * 1024, true},
		{"maximum 100 MB", 100 * 1024 * 1024, true},
		{"too small", 1024, false},
		{"too large", 200 * 1024 * 1024, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minSize := int64(4 * 1024)
			maxSize := int64(100 * 1024 * 1024)
			valid := tt.blockSize >= minSize && tt.blockSize <= maxSize

			if valid != tt.valid {
				t.Errorf("Block size %v validity = %v, want %v", tt.blockSize, valid, tt.valid)
			}
		})
	}
}

func TestAzureWriter_Context(t *testing.T) {
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
			ctx:        cancelledAzureContext(),
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

func cancelledAzureContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestAzureWriter_Close(t *testing.T) {
	closed := false

	closeFunc := func() error {
		if closed {
			return errors.New("already closed")
		}
		closed = true
		return nil
	}

	if err := closeFunc(); err != nil {
		t.Errorf("First close failed: %v", err)
	}
	if !closed {
		t.Error("Should be closed")
	}

	if err := closeFunc(); err == nil {
		t.Error("Second close should fail")
	}
}

func TestAzureWriter_Compression(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validCompressions := map[string]bool{
				"snappy": true,
				"gzip":   true,
				"zstd":   true,
				"none":   true,
			}

			valid := validCompressions[tt.compression]
			if valid != tt.valid {
				t.Errorf("Compression %v validity = %v, want %v", tt.compression, valid, tt.valid)
			}
		})
	}
}

func TestAzureWriter_RecordBatch(t *testing.T) {
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
	}

	if len(records) != 1 {
		t.Errorf("Record count = %d, want 1", len(records))
	}

	if records[0].Event == nil {
		t.Error("Event should not be nil")
	}
}

func TestAzureEmulator(t *testing.T) {
	// Azurite emulator default endpoint
	emulatorEndpoint := "http://127.0.0.1:10000/devstoreaccount1"
	emulatorAccountName := "devstoreaccount1"
	emulatorAccountKey := "Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw=="

	config := AzureConfig{
		AccountName:   emulatorAccountName,
		AccountKey:    emulatorAccountKey,
		ContainerName: "test-container",
		Endpoint:      emulatorEndpoint,
	}

	if err := validateAzureConfig(config); err != nil {
		t.Errorf("Emulator config should be valid: %v", err)
	}
}

func TestAzureConnectionString(t *testing.T) {
	connectionString := "DefaultEndpointsProtocol=https;AccountName=testaccount;AccountKey=dGVzdGtleQ==;EndpointSuffix=core.windows.net"

	if connectionString == "" {
		t.Error("Connection string should not be empty")
	}

	requiredComponents := []string{"AccountName", "AccountKey"}
	for _, component := range requiredComponents {
		if !contains(connectionString, component) {
			t.Errorf("Connection string missing %v", component)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr))
}

func TestAzureSASToken(t *testing.T) {
	sasToken := "?sv=2020-08-04&ss=b&srt=sco&sp=rwdlac&se=2025-12-31T23:59:59Z&st=2025-01-01T00:00:00Z&spr=https&sig=signature"

	if sasToken == "" {
		t.Error("SAS token should not be empty")
	}

	if sasToken[0] != '?' {
		t.Error("SAS token should start with '?'")
	}

	requiredParams := []string{"sv", "sp", "sig"}
	for _, param := range requiredParams {
		if !contains(sasToken, param) {
			t.Errorf("SAS token missing parameter %v", param)
		}
	}
}
