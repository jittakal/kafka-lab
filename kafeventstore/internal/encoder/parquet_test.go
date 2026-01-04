package encoder

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
	"github.com/parquet-go/parquet-go"
)

// TestParquetEncoder_AthenaCompatibility verifies that generated Parquet files
// have the correct schema and metadata for AWS Athena compatibility.
func TestParquetEncoder_AthenaCompatibility(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "athena-test.parquet")

	encoder := NewParquetEncoder("snappy")

	now := time.Now()
	subject := "test-subject"
	contentType := "application/json"
	dataSchema := "https://schema.example.com"

	records := []event.Record{
		{
			Event: &event.CloudEvent{
				SpecVersion:     "1.0",
				ID:              "test-id-1",
				Source:          "test-source",
				Type:            "test.event",
				Time:            &now,
				Subject:         &subject,
				DataContentType: &contentType,
				DataSchema:      &dataSchema,
				Data:            []byte(`{"message": "test"}`),
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
	}

	// Encode the records
	_, err := encoder.Encode(testFile, records)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Read and verify record count using simpler ReadFile API
	readRecords, err := parquet.ReadFile[CloudEventParquet](testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if len(readRecords) != len(records) {
		t.Errorf("record count = %d, want %d", len(readRecords), len(records))
	}

	// Verify timestamp values are preserved
	if len(readRecords) > 0 {
		rec := readRecords[0]

		// Verify timestamps are time.Time, not strings
		if rec.KafkaTimestamp.IsZero() {
			t.Error("kafka_timestamp should not be zero")
		}
		if rec.IngestedAt.IsZero() {
			t.Error("ingested_at should not be zero")
		}

		// Verify optional fields
		if rec.Subject == nil {
			t.Error("subject should not be nil")
		} else if *rec.Subject != subject {
			t.Errorf("subject = %v, want %v", *rec.Subject, subject)
		}

		if rec.DataContentType == nil {
			t.Error("data_content_type should not be nil")
		}

		if rec.Time != nil {
			// Verify time is close to expected (within 1 second due to precision)
			diff := rec.Time.Sub(now)
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Second {
				t.Errorf("time difference too large: %v", diff)
			}
		}
	}
}

// TestParquetEncoder_CompressionCodecs tests all supported compression codecs.
func TestParquetEncoder_CompressionCodecs(t *testing.T) {
	compressions := []string{"snappy", "gzip", "lz4", "zstd", "uncompressed"}
	tempDir := t.TempDir()

	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				SpecVersion: "1.0",
				ID:          "test-compression",
				Source:      "test",
				Type:        "test.event",
				Time:        &now,
				Data:        []byte(`{"test": "data"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test",
				Partition: 0,
				Offset:    1,
				Timestamp: now,
			},
			ProcessedAt: now,
		},
	}

	for _, compression := range compressions {
		t.Run(compression, func(t *testing.T) {
			encoder := NewParquetEncoder(compression)
			testFile := filepath.Join(tempDir, compression+".parquet")

			stats, err := encoder.Encode(testFile, records)
			if err != nil {
				t.Fatalf("Encode() with %s error = %v", compression, err)
			}

			if stats.RecordCount != 1 {
				t.Errorf("RecordCount = %d, want 1", stats.RecordCount)
			}

			// Verify file can be read back
			readRecs, err := parquet.ReadFile[CloudEventParquet](testFile)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			if len(readRecs) != 1 {
				t.Errorf("read %d records, want 1", len(readRecs))
			}
		})
	}
}

// TestParquetEncoder_NullHandling verifies proper NULL handling for optional fields.
func TestParquetEncoder_NullHandling(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "null-test.parquet")

	encoder := NewParquetEncoder("snappy")

	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				SpecVersion: "1.0",
				ID:          "test-null",
				Source:      "test",
				Type:        "test.event",
				// All optional fields are nil
				Time:            nil,
				Subject:         nil,
				DataContentType: nil,
				DataSchema:      nil,
				Data:            []byte(`{}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test",
				Partition: 0,
				Offset:    1,
				Timestamp: now,
			},
			ProcessedAt: now,
		},
	}

	_, err := encoder.Encode(testFile, records)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	// Read back and verify NULLs are preserved
	readRecs, err := parquet.ReadFile[CloudEventParquet](testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if len(readRecs) != 1 {
		t.Fatalf("read %d records, want 1", len(readRecs))
	}

	rec := readRecs[0]

	// Verify optional fields are nil
	if rec.Time != nil {
		t.Error("time should be nil")
	}
	if rec.Subject != nil {
		t.Error("subject should be nil")
	}
	if rec.DataContentType != nil {
		t.Error("data_content_type should be nil")
	}
	if rec.DataSchema != nil {
		t.Error("data_schema should be nil")
	}

	// Verify required fields are present
	if rec.ID == "" {
		t.Error("id should not be empty")
	}
	if rec.KafkaTimestamp.IsZero() {
		t.Error("kafka_timestamp should not be zero")
	}
}
