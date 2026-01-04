// Package storage defines interfaces for event storage operations.
//
// This package provides abstractions for writing events to various
// storage backends (S3, Azure Blob, local filesystem).
package storage

import (
	"context"

	"github.com/jittakal/kafeventstore/pkg/event"
)

// Writer writes event records to storage.
type Writer interface {
	// Write writes records to storage at the specified path.
	// Returns the number of bytes written.
	Write(ctx context.Context, records []event.Record, path string, format event.FileFormat) (int64, error)

	// Close closes the writer and releases resources.
	Close() error
}

// Router determines storage paths for events based on partitioning strategy.
type Router interface {
	// Route returns the storage path for a partition at a given time.
	// timestamp: Unix timestamp (seconds) representing the event time
	// specVersion: CloudEvents spec version (e.g., \"1.0\") for dynamic versioning, empty string uses default
	Route(partitionID event.PartitionID, timestamp int64, specVersion string) string
}

// RotationPolicy determines when to rotate (flush) buffered events to storage.
type RotationPolicy interface {
	// ShouldRotate returns true if the buffer should be flushed based on stats.
	ShouldRotate(stats event.FileStats) bool
}
