package encoder

import (
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
	"github.com/parquet-go/parquet-go"
)

// TestParquetEncoder_WriteAndRead_Simple validates basic write and read functionality.
func TestParquetEncoder_WriteAndRead_Simple(t *testing.T) {
	tempDir := t.TempDir()
	testFile := tempDir + "/simple.parquet"

	encoder := NewParquetEncoder("snappy")

	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				SpecVersion: "1.0",
				ID:          "test-1",
				Source:      "test",
				Type:        "test.event",
				Time:        &now,
				Data:        []byte(`{"key":"value"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    1,
				Timestamp: now,
			},
			ProcessedAt: now,
		},
	}

	// Write
	stats, err := encoder.Encode(testFile, records)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	t.Logf("Wrote %d records, %d bytes", stats.RecordCount, stats.SizeBytes)

	// Read back
	buf, err := parquet.ReadFile[CloudEventParquet](testFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if len(buf) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(buf))
	}

	rec := buf[0]
	if rec.ID != "test-1" {
		t.Errorf("ID = %q, want %q", rec.ID, "test-1")
	}
	if rec.Type != "test.event" {
		t.Errorf("Type = %q, want %q", rec.Type, "test.event")
	}
	if rec.KafkaTimestamp.IsZero() {
		t.Error("KafkaTimestamp should not be zero")
	}
	if rec.IngestedAt.IsZero() {
		t.Error("IngestedAt should not be zero")
	}

	t.Log("✅ Successfully wrote and read Parquet file with time.Time timestamps")
}
