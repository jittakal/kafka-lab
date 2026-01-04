package encoder_test

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jittakal/kafeventstore/internal/encoder"
	"github.com/jittakal/kafeventstore/pkg/event"
)

func Example_parquetEncoder() {
	// Create a Parquet encoder with Snappy compression
	enc := encoder.NewParquetEncoder("snappy")

	// Prepare sample records
	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				ID:          "evt-1",
				Source:      "example-service",
				SpecVersion: "1.0",
				Type:        "example.event",
				Time:        &now,
				Data:        []byte(`{"message": "hello"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "example-topic",
				Partition: 0,
				Offset:    100,
				Timestamp: now,
			},
		},
	}

	// Create temp directory and file
	tmpDir := os.TempDir()
	filePath := filepath.Join(tmpDir, "example.parquet")
	defer os.Remove(filePath)

	// Encode records to Parquet file
	stats, err := enc.Encode(filePath, records)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Encoded %d records\n", stats.RecordCount)
	fmt.Printf("File format: %s\n", enc.Format())
	fmt.Printf("File extension: %s\n", enc.FileExtension())

	// Output:
	// Encoded 1 records
	// File format: parquet
	// File extension: .parquet
}

func Example_avroEncoder() {
	// Create an Avro encoder with GZIP compression
	enc, err := encoder.NewAvroEncoder("gzip")
	if err != nil {
		fmt.Println("Error creating encoder:", err)
		return
	}

	// Prepare sample records
	now := time.Now()
	records := []event.Record{
		{
			Event: &event.CloudEvent{
				ID:          "evt-1",
				Source:      "example-service",
				SpecVersion: "1.0",
				Type:        "example.event",
				Time:        &now,
				Data:        []byte(`{"message": "hello"}`),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "example-topic",
				Partition: 0,
				Offset:    100,
				Timestamp: now,
			},
		},
	}

	// Create temp directory and file
	tmpDir := os.TempDir()
	filePath := filepath.Join(tmpDir, "example.avro")
	defer os.Remove(filePath)

	// Encode records to Avro file
	stats, err := enc.Encode(filePath, records)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Encoded %d records\n", stats.RecordCount)
	fmt.Printf("File format: %s\n", enc.Format())
	fmt.Printf("File extension: %s\n", enc.FileExtension())

	// Output:
	// Encoded 1 records
	// File format: avro
	// File extension: .avro.gz
}

func Example_encoderFactory() {
	// Create a factory for Parquet format with Snappy compression
	factory := encoder.NewFactory(event.FormatParquet, "snappy")

	// Create encoder instances
	enc1, err := factory.CreateEncoder()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	enc2, err := factory.CreateEncoder()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Each call creates a new independent encoder
	fmt.Printf("Created independent encoders: %v\n", enc1 != enc2)
	fmt.Printf("Both produce same format: %v\n", enc1.Format() == enc2.Format())

	// Output:
	// Created independent encoders: true
	// Both produce same format: true
}
