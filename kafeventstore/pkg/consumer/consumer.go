// Package consumer defines interfaces for Kafka event consumption.
//
// This package provides abstractions for consuming events from Kafka
// and managing consumer lifecycle.
package consumer

import (
	"context"

	"github.com/jittakal/kafeventstore/pkg/event"
)

// Consumer reads events from Kafka topics.
type Consumer interface {
	// Subscribe subscribes to one or more topics.
	Subscribe(ctx context.Context, topics []string) error

	// Consume starts consuming messages from subscribed topics.
	// Returns channels for events and errors.
	Consume(ctx context.Context) (<-chan *event.ConsumedEvent, <-chan error, error)

	// Commit commits the offset for a partition.
	Commit(ctx context.Context, partition event.PartitionID, offset int64) error

	// Close closes the consumer and releases resources.
	Close() error
}

// DLQPublisher publishes failed events to a dead letter queue.
type DLQPublisher interface {
	// Publish sends an event to the DLQ with error information.
	Publish(ctx context.Context, event *event.CloudEvent, metadata event.KafkaMetadata, reason string) error

	// Close closes the publisher and releases resources.
	Close() error
}
