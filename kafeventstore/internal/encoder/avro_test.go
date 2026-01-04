package encoder

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestNewAvroEncoder(t *testing.T) {
	tests := []struct {
		name        string
		compression string
		wantErr     bool
	}{
		{"gzip compression", "gzip", false},
		{"uncompressed", "uncompressed", false},
		{"deflate compression", "deflate", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder, err := NewAvroEncoder(tt.compression)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAvroEncoder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && encoder == nil {
				t.Error("expected non-nil encoder")
			}
			if !tt.wantErr && encoder.compression != tt.compression {
				t.Errorf("compression = %v, want %v", encoder.compression, tt.compression)
			}
		})
	}
}

func TestAvroEncoder_FileExtension(t *testing.T) {
	tests := []struct {
		name        string
		compression string
		want        string
	}{
		{"no compression", "none", ".avro"},
		{"gzip compression", "gzip", ".avro.gz"},
		{"GZIP compression", "GZIP", ".avro.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder, err := NewAvroEncoder(tt.compression)
			if err != nil {
				t.Fatalf("NewAvroEncoder() error = %v", err)
			}

			ext := encoder.FileExtension()
			if ext != tt.want {
				t.Errorf("FileExtension() = %v, want %v", ext, tt.want)
			}
		})
	}
}

func TestAvroEncoder_Format(t *testing.T) {
	encoder, err := NewAvroEncoder("gzip")
	if err != nil {
		t.Fatalf("NewAvroEncoder() error = %v", err)
	}

	format := encoder.Format()
	if format != event.FormatAvro {
		t.Errorf("Format() = %v, want %v", format, event.FormatAvro)
	}
}

func TestAvroEncoder_Encode(t *testing.T) {
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "test-avro-encode.avro")
	defer os.Remove(testFile)

	encoder, err := NewAvroEncoder("gzip")
	if err != nil {
		t.Fatalf("NewAvroEncoder() error = %v", err)
	}

	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				SpecVersion:     "1.0",
				ID:              "test-id-1",
				Source:          "test-source",
				Type:            "test.event",
				Subject:         stringPtr("test-subject"),
				DataContentType: stringPtr("application/json"),
				DataSchema:      stringPtr("http://example.com/schema"),
				Time:            &now,
				Data:            []byte(`{"message": "test data 1"}`),
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
				ID:          "test-id-2",
				Source:      "test-source",
				Type:        "test.event",
				Time:        &now,
				Data:        []byte(`{"message": "test data 2"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 1,
				Offset:    200,
				Timestamp: now,
			},
			Offset:      200,
			ProcessedAt: now,
		},
	}

	stats, err := encoder.Encode(testFile, records)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if stats == nil {
		t.Fatal("expected non-nil stats")
	}

	if stats.RecordCount != len(records) {
		t.Errorf("RecordCount = %d, want %d", stats.RecordCount, len(records))
	}

	if stats.SizeBytes <= 0 {
		t.Errorf("SizeBytes = %d, want > 0", stats.SizeBytes)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("expected file at %s", testFile)
	}
}

func TestAvroEncoder_EncodeWithUncompressed(t *testing.T) {
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "test-avro-uncompressed.avro")
	defer os.Remove(testFile)

	encoder, err := NewAvroEncoder("uncompressed")
	if err != nil {
		t.Fatalf("NewAvroEncoder() error = %v", err)
	}

	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				SpecVersion: "1.0",
				ID:          "test-id",
				Source:      "test-source",
				Type:        "test.event",
				Time:        &now,
				Data:        []byte(`{"test": true}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    1,
				Timestamp: now,
			},
			Offset:      1,
			ProcessedAt: now,
		},
	}

	stats, err := encoder.Encode(testFile, records)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if stats.RecordCount != 1 {
		t.Errorf("RecordCount = %d, want 1", stats.RecordCount)
	}
}

func TestAvroEncoder_EncodeEmptyRecords(t *testing.T) {
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "test-avro-empty.avro")
	defer os.Remove(testFile)

	encoder, err := NewAvroEncoder("gzip")
	if err != nil {
		t.Fatalf("NewAvroEncoder() error = %v", err)
	}

	records := []event.Record{}

	_, err = encoder.Encode(testFile, records)
	if err == nil {
		t.Error("expected error for empty records")
	}
}

func TestAvroEncoder_EncodeToBytes(t *testing.T) {
	encoder, err := NewAvroEncoder("gzip")
	if err != nil {
		t.Fatalf("NewAvroEncoder() error = %v", err)
	}

	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			SpecVersion: "1.0",
			ID:          "test-id",
			Source:      "test-source",
			Type:        "test.event",
			Time:        &now,
			Data:        []byte(`{"key": "value"}`),
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    1,
			Timestamp: now,
		},
		Offset:      1,
		ProcessedAt: now,
	}

	bytes, err := encoder.EncodeToBytes([]event.Record{record})
	if err != nil {
		t.Fatalf("EncodeToBytes() error = %v", err)
	}

	if len(bytes) == 0 {
		t.Error("expected non-empty bytes")
	}
}

func TestGetAvroSchema(t *testing.T) {
	schema := avroSchema()

	if len(schema) == 0 {
		t.Error("expected non-empty schema")
	}

	// Verify schema contains required fields
	requiredFields := []string{
		"spec_version",
		"id",
		"source",
		"type",
		"data",
		"kafka_topic",
		"kafka_partition",
		"kafka_offset",
	}

	for _, field := range requiredFields {
		if !contains(schema, field) {
			t.Errorf("schema missing required field: %s", field)
		}
	}
}

func TestConvertToAvroMap(t *testing.T) {
	encoder, err := NewAvroEncoder("gzip")
	if err != nil {
		t.Fatalf("NewAvroEncoder() error = %v", err)
	}

	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			SpecVersion:     "1.0",
			ID:              "test-id",
			Source:          "test-source",
			Type:            "test.event",
			Subject:         stringPtr("test-subject"),
			DataContentType: stringPtr("application/json"),
			DataSchema:      stringPtr("http://schema.example.com"),
			Time:            &now,
			Data:            []byte(`{"key": "value"}`),
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 5,
			Offset:    1000,
			Timestamp: now,
		},
		Offset:      1000,
		ProcessedAt: now,
	}

	avroMap, err := encoder.convertToAvroMap(record)
	if err != nil {
		t.Fatalf("convertToAvroMap() error = %v", err)
	}

	// Verify all required fields are present
	if avroMap["id"] != "test-id" {
		t.Errorf("id = %v, want test-id", avroMap["id"])
	}
	if avroMap["source"] != "test-source" {
		t.Errorf("source = %v, want test-source", avroMap["source"])
	}
	if avroMap["type"] != "test.event" {
		t.Errorf("type = %v, want test.event", avroMap["type"])
	}
	if avroMap["kafka_topic"] != "test-topic" {
		t.Errorf("kafka_topic = %v, want test-topic", avroMap["kafka_topic"])
	}

	// Verify optional fields
	if subject, ok := avroMap["subject"].(map[string]interface{}); ok {
		if subject["string"] != "test-subject" {
			t.Errorf("subject = %v, want test-subject", subject["string"])
		}
	}
}

func TestAvroEncoder_WithNullFields(t *testing.T) {
	tempDir := os.TempDir()
	testFile := filepath.Join(tempDir, "test-avro-nulls.avro")
	defer os.Remove(testFile)

	encoder, err := NewAvroEncoder("gzip")
	if err != nil {
		t.Fatalf("NewAvroEncoder() error = %v", err)
	}

	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			SpecVersion: "1.0",
			ID:          "test-id",
			Source:      "test-source",
			Type:        "test.event",
			// No Subject, DataContentType, DataSchema - should be null
			Time: &now,
			Data: []byte(`{}`),
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    1,
			Timestamp: now,
		},
		Offset:      1,
		ProcessedAt: now,
	}

	stats, err := encoder.Encode(testFile, []event.Record{record})
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	if stats.RecordCount != 1 {
		t.Errorf("RecordCount = %d, want 1", stats.RecordCount)
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
