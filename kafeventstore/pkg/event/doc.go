// Package event defines core event types and interfaces for CloudEvents processing.
//
// This package provides the public API for working with CloudEvents following the
// CloudEvents 1.0 specification, along with Kafka metadata for event streaming.
//
// # Core Types
//
// CloudEvent represents a CloudEvents 1.0 event with required and optional fields:
//
//	event := &event.CloudEvent{
//	    ID:          "unique-event-id",
//	    Source:      "service-name",
//	    SpecVersion: "1.0",
//	    Type:        "com.example.event",
//	    Time:        &now,
//	    Data:        []byte(`{"key": "value"}`),
//	}
//
// # Record Structure
//
// Record combines a CloudEvent with Kafka metadata for complete event context:
//
//	record := event.Record{
//	    Event: cloudEvent,
//	    Kafka: event.KafkaMetadata{
//	        Topic:     "events",
//	        Partition: 0,
//	        Offset:    12345,
//	        Timestamp: time.Now(),
//	    },
//	}
//
// # Partition Identification
//
// PartitionID uniquely identifies a Kafka topic partition:
//
//	pid := event.PartitionID{
//	    Topic:     "user-events",
//	    Partition: 5,
//	}
//	key := pid.String() // "user-events-5"
//
// # File Formats
//
// The package defines supported file formats for event storage:
//
//	event.FormatParquet  // Columnar format for analytics
//	event.FormatAvro     // Row-based format with schema
//
// # Validation
//
// Use the Validator interface to validate CloudEvents before processing:
//
//	type Validator interface {
//	    Validate(event *CloudEvent) error
//	}
//
// # Time Utilities
//
// Records provide convenient methods for extracting event timestamps:
//
//	eventTime := record.GetEventTime()      // Returns time.Time
//	unixTime := record.GetEventTimeUnix()   // Returns Unix timestamp
//
// The methods fall back to Kafka timestamp if CloudEvent.Time is not set.
package event
