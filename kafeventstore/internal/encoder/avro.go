// Package encoder implements file format encoders.
package encoder

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jittakal/kafeventstore/pkg/encoder"
	"github.com/jittakal/kafeventstore/pkg/event"
	"github.com/linkedin/goavro/v2"
)

// Ensure implementation satisfies interface at compile time.
var _ encoder.Encoder = (*AvroEncoder)(nil)

// AvroEncoder implements encoder.Encoder for Apache Avro binary format.
// It supports optional gzip compression and follows the CloudEvents specification
// for schema design. Produces OCF (Object Container File) format compatible with
// Apache Spark and other Avro readers.
type AvroEncoder struct {
	codec       *goavro.Codec
	compression string
}

// NewAvroEncoder creates a new Avro encoder with specified compression.
func NewAvroEncoder(compression string) (*AvroEncoder, error) {
	schema := avroSchema()
	codec, err := goavro.NewCodec(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create avro codec: %w", err)
	}

	return &AvroEncoder{
		codec:       codec,
		compression: compression,
	}, nil
}

// avroSchema returns the Avro schema for storage records.
func avroSchema() string {
	return `{
		"type": "record",
		"name": "StorageRecord",
		"namespace": "com.kafka.event.store",
		"fields": [
			{"name": "spec_version", "type": "string"},
			{"name": "id", "type": "string"},
			{"name": "source", "type": "string"},
			{"name": "type", "type": "string"},
			{"name": "subject", "type": ["null", "string"], "default": null},
			{"name": "data_content_type", "type": ["null", "string"], "default": null},
			{"name": "data_schema", "type": ["null", "string"], "default": null},
			{"name": "time", "type": ["null", "string"], "default": null},
			{"name": "data", "type": "string"},
			{"name": "kafka_topic", "type": "string"},
			{"name": "kafka_partition", "type": "int"},
			{"name": "kafka_offset", "type": "long"},
			{"name": "kafka_timestamp", "type": "string"},
			{"name": "ingested_at", "type": "string"}
		]
	}`
}

// Encode writes records to an Avro file.
func (e *AvroEncoder) Encode(filePath string, records []event.Record) (*event.FileStats, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records to encode")
	}

	// Create output file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	var writer io.Writer = file
	var gzipWriter *gzip.Writer

	// Apply compression if specified
	if e.compression == "gzip" || e.compression == "GZIP" {
		gzipWriter = gzip.NewWriter(file)
		writer = gzipWriter
		defer gzipWriter.Close()
	}

	// Create OCF writer (Object Container File)
	ocfWriter, err := goavro.NewOCFWriter(goavro.OCFConfig{
		W:     writer,
		Codec: e.codec,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OCF writer: %w", err)
	}

	// Convert and write records
	for _, record := range records {
		avroMap, err := e.convertToAvroMap(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record: %w", err)
		}

		if err := ocfWriter.Append([]interface{}{avroMap}); err != nil {
			return nil, fmt.Errorf("failed to write record: %w", err)
		}
	}

	// Ensure all data is flushed
	if gzipWriter != nil {
		if err := gzipWriter.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}
	}

	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("failed to close file: %w", err)
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	stats := &event.FileStats{
		RecordCount:    len(records),
		SizeBytes:      fileInfo.Size(),
		FirstWriteTime: time.Now(),
		LastWriteTime:  time.Now(),
	}

	return stats, nil
}

// convertToAvroMap converts a Record to Avro map representation.
func (e *AvroEncoder) convertToAvroMap(record event.Record) (map[string]interface{}, error) {
	// Serialize data to JSON string
	dataJSON, err := json.Marshal(record.Event.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	avroMap := map[string]interface{}{
		"spec_version":    record.Event.SpecVersion,
		"id":              record.Event.ID,
		"source":          record.Event.Source,
		"type":            record.Event.Type,
		"data":            string(dataJSON),
		"kafka_topic":     record.Kafka.Topic,
		"kafka_partition": int32(record.Kafka.Partition),
		"kafka_offset":    record.Kafka.Offset,
		"kafka_timestamp": record.Kafka.Timestamp.Format(time.RFC3339Nano),
		"ingested_at":     record.ProcessedAt.Format(time.RFC3339Nano),
	}

	// Optional fields - use goavro.Union for nullable fields
	if record.Event.Subject != nil && *record.Event.Subject != "" {
		avroMap["subject"] = goavro.Union("string", *record.Event.Subject)
	} else {
		avroMap["subject"] = nil
	}

	if record.Event.DataContentType != nil && *record.Event.DataContentType != "" {
		avroMap["data_content_type"] = goavro.Union("string", *record.Event.DataContentType)
	} else {
		avroMap["data_content_type"] = nil
	}

	if record.Event.DataSchema != nil && *record.Event.DataSchema != "" {
		avroMap["data_schema"] = goavro.Union("string", *record.Event.DataSchema)
	} else {
		avroMap["data_schema"] = nil
	}

	if record.Event.Time != nil {
		avroMap["time"] = goavro.Union("string", record.Event.Time.Format(time.RFC3339Nano))
	} else {
		avroMap["time"] = nil
	}

	return avroMap, nil
}

// EncodeToBytes encodes records to bytes (useful for testing).
func (e *AvroEncoder) EncodeToBytes(records []event.Record) ([]byte, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records to encode")
	}

	var buf bytes.Buffer
	var writer io.Writer = &buf

	// Apply compression if specified
	var gzipWriter *gzip.Writer
	if e.compression == "gzip" || e.compression == "GZIP" {
		gzipWriter = gzip.NewWriter(&buf)
		writer = gzipWriter
	}

	// Create OCF writer
	ocfWriter, err := goavro.NewOCFWriter(goavro.OCFConfig{
		W:     writer,
		Codec: e.codec,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OCF writer: %w", err)
	}

	// Convert and write records
	for _, record := range records {
		avroMap, err := e.convertToAvroMap(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record: %w", err)
		}

		if err := ocfWriter.Append([]interface{}{avroMap}); err != nil {
			return nil, fmt.Errorf("failed to write record: %w", err)
		}
	}

	if gzipWriter != nil {
		if err := gzipWriter.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// Format returns the file format.
func (e *AvroEncoder) Format() event.FileFormat {
	return event.FormatAvro
}

// FileExtension returns the file extension.
func (e *AvroEncoder) FileExtension() string {
	if e.compression == "gzip" || e.compression == "GZIP" {
		return ".avro.gz"
	}
	return ".avro"
}
