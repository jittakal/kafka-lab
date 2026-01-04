// Package event defines core event types and interfaces for event processing.
//
// This package contains the public API for working with CloudEvents
// and event metadata. It follows CloudEvents 1.0 specification.
package event

import (
	"encoding/json"
	"fmt"
	"time"
)

// CloudEvent represents a CloudEvents 1.0 event.
// See https://github.com/cloudevents/spec/blob/v1.0/spec.md
type CloudEvent struct {
	// Required attributes
	ID          string `json:"id"`
	Source      string `json:"source"`
	SpecVersion string `json:"specversion"`
	Type        string `json:"type"`

	// Optional attributes
	DataContentType *string    `json:"datacontenttype,omitempty"`
	DataSchema      *string    `json:"dataschema,omitempty"`
	Subject         *string    `json:"subject,omitempty"`
	Time            *time.Time `json:"time,omitempty"`

	// Event data - can be any JSON value (object, array, string, number, etc.)
	Data json.RawMessage `json:"data,omitempty"`

	// Extension attributes
	Extensions map[string]interface{} `json:"-"`
}

// KafkaMetadata contains Kafka-specific metadata for an event.
type KafkaMetadata struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Headers   map[string]string
	Timestamp time.Time
}

// PartitionID uniquely identifies a Kafka partition.
type PartitionID struct {
	Topic     string
	Partition int32
}

// String returns a string representation of the partition ID in the format "topic-partition".
func (p PartitionID) String() string {
	return fmt.Sprintf("%s-%d", p.Topic, p.Partition)
}

// Record represents a processed event ready for storage.
type Record struct {
	Event       *CloudEvent
	Kafka       KafkaMetadata
	Offset      int64
	ProcessedAt time.Time
}

// FileStats contains statistics about buffered events.
type FileStats struct {
	RecordCount    int
	SizeBytes      int64
	FirstWriteTime time.Time
	LastWriteTime  time.Time
}

// FileFormat represents the storage file format.
type FileFormat string

const (
	FormatParquet FileFormat = "parquet"
	FormatAvro    FileFormat = "avro"
)

// Validator validates CloudEvents.
type Validator interface {
	// Validate checks if a CloudEvent is valid according to the spec.
	Validate(event *CloudEvent) error
}

// ConsumedEvent represents an event consumed from Kafka.
type ConsumedEvent struct {
	Event      *CloudEvent
	Metadata   KafkaMetadata
	CommitFunc func() error
}

// GetEventTime returns the event's timestamp.
// It returns the CloudEvent.Time if present, otherwise falls back to Kafka message timestamp.
func (r *Record) GetEventTime() time.Time {
	if r.Event != nil && r.Event.Time != nil {
		return *r.Event.Time
	}
	// Fallback to Kafka timestamp (when message was produced)
	return r.Kafka.Timestamp
}

// GetEventTimeUnix returns the event's timestamp as Unix seconds.
func (r *Record) GetEventTimeUnix() int64 {
	return r.GetEventTime().Unix()
}
