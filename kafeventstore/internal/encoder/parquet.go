// Package encoder implements file format encoders.
package encoder

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jittakal/kafeventstore/pkg/encoder"
	"github.com/jittakal/kafeventstore/pkg/event"
	"github.com/parquet-go/parquet-go"
)

// Ensure implementation satisfies interface at compile time.
var _ encoder.Encoder = (*ParquetEncoder)(nil)

// CloudEventParquet represents the Parquet schema for CloudEvents storage.
// Uses native Parquet types for Athena compatibility, including TIMESTAMP_MICROS for time fields.
type CloudEventParquet struct {
	// CloudEvent fields - required
	SpecVersion string `parquet:"spec_version,dict"`
	ID          string `parquet:"id,dict"`
	Source      string `parquet:"source,dict"`
	Type        string `parquet:"type,dict"`
	Data        string `parquet:"data"`

	// CloudEvent fields - optional (using pointers for proper NULL handling)
	Subject         *string    `parquet:"subject,dict,optional"`
	DataContentType *string    `parquet:"data_content_type,dict,optional"`
	DataSchema      *string    `parquet:"data_schema,dict,optional"`
	Time            *time.Time `parquet:"time,timestamp(microsecond),optional"`

	// Kafka metadata fields
	KafkaTopic     string    `parquet:"kafka_topic,dict"`
	KafkaPartition int32     `parquet:"kafka_partition"`
	KafkaOffset    int64     `parquet:"kafka_offset"`
	KafkaTimestamp time.Time `parquet:"kafka_timestamp,timestamp(microsecond)"`

	// Storage metadata
	IngestedAt time.Time `parquet:"ingested_at,timestamp(microsecond)"`
}

// ParquetEncoder implements encoder.Encoder for Apache Parquet columnar format.
// Uses Apache parquet-go library for full Athena/Hive compatibility with proper metadata.
// Supports multiple compression codecs: SNAPPY (default), GZIP, LZ4, ZSTD.
type ParquetEncoder struct {
	compressionName string
}

// NewParquetEncoder creates a new Parquet encoder with specified compression.
func NewParquetEncoder(compression string) *ParquetEncoder {
	return &ParquetEncoder{
		compressionName: compression,
	}
}

// compressionCodec converts string compression name to parquet WriterOption.
func compressionCodec(compression string) parquet.WriterOption {
	// parquet-go uses WriterOptions for compression configuration
	// The compression codec is applied at the column level via tags
	// or globally via parquet.Compression() with codec name
	switch compression {
	case "snappy", "SNAPPY":
		return parquet.Compression(&parquet.Snappy)
	case "gzip", "GZIP":
		return parquet.Compression(&parquet.Gzip)
	case "lz4", "LZ4":
		return parquet.Compression(&parquet.Lz4Raw)
	case "zstd", "ZSTD":
		return parquet.Compression(&parquet.Zstd)
	case "uncompressed", "UNCOMPRESSED", "none", "NONE":
		return parquet.Compression(&parquet.Uncompressed)
	default:
		return parquet.Compression(&parquet.Snappy) // Default to Snappy
	}
}

// Encode writes records to a Parquet file using Apache parquet-go.
// Creates files with proper Hive-compatible metadata for Athena queries.
func (e *ParquetEncoder) Encode(filePath string, records []event.Record) (*event.FileStats, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records to encode")
	}

	// Create output file
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}

	// Convert records to Parquet schema
	parquetRecords := make([]CloudEventParquet, len(records))
	for i, record := range records {
		parquetRec, err := e.convertToParquetRecord(record)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to convert record %d: %w", i, err)
		}
		parquetRecords[i] = *parquetRec
	}

	// Create schema from struct
	schema := parquet.SchemaOf(new(CloudEventParquet))

	// Write Parquet file with compression
	writer := parquet.NewGenericWriter[CloudEventParquet](
		file,
		schema,
		compressionCodec(e.compressionName),
		parquet.CreatedBy("kafka-event-blob-store", "1.0", "0"),
	)

	// Write all records
	if _, err := writer.Write(parquetRecords); err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to write records: %w", err)
	}

	// Flush and close writer
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close writer: %w", err)
	}

	// Close file before getting stats to ensure all data is flushed
	if err := file.Close(); err != nil {
		return nil, fmt.Errorf("failed to close file: %w", err)
	}

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

// convertToParquetRecord converts a Record to CloudEventParquet with native types.
func (e *ParquetEncoder) convertToParquetRecord(record event.Record) (*CloudEventParquet, error) {
	// Serialize data to JSON string
	dataJSON, err := json.Marshal(record.Event.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	parquetRec := &CloudEventParquet{
		SpecVersion:    record.Event.SpecVersion,
		ID:             record.Event.ID,
		Source:         record.Event.Source,
		Type:           record.Event.Type,
		Data:           string(dataJSON),
		KafkaTopic:     record.Kafka.Topic,
		KafkaPartition: record.Kafka.Partition,
		KafkaOffset:    record.Kafka.Offset,
		KafkaTimestamp: record.Kafka.Timestamp,
		IngestedAt:     record.ProcessedAt,
	}

	// Optional fields - assign pointers for proper NULL representation
	if record.Event.Subject != nil {
		parquetRec.Subject = record.Event.Subject
	}
	if record.Event.DataContentType != nil {
		parquetRec.DataContentType = record.Event.DataContentType
	}
	if record.Event.DataSchema != nil {
		parquetRec.DataSchema = record.Event.DataSchema
	}
	if record.Event.Time != nil {
		parquetRec.Time = record.Event.Time
	}

	return parquetRec, nil
}

// Format returns the file format.
func (e *ParquetEncoder) Format() event.FileFormat {
	return event.FormatParquet
}

// FileExtension returns the file extension.
func (e *ParquetEncoder) FileExtension() string {
	return ".parquet"
}
