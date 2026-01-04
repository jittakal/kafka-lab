// Package buffer defines interfaces for event buffering operations.
//
// Buffers are used to batch events before writing to storage,
// improving throughput and reducing storage operations.
package buffer

import (
	"github.com/jittakal/kafeventstore/pkg/event"
)

// Buffer manages buffering of events before storage.
// All implementations must be thread-safe.
type Buffer interface {
	// Add adds a record to the buffer.
	// Returns an error if the buffer is full or capacity would be exceeded.
	Add(record event.Record) error

	// Drain removes and returns all records from the buffer.
	// The buffer is reset after draining.
	Drain() []event.Record

	// Stats returns current buffer statistics without modifying the buffer.
	Stats() event.FileStats

	// IsEmpty returns true if the buffer contains no records.
	IsEmpty() bool

	// Reset clears the buffer and resets all statistics.
	Reset()
}

// Manager creates and manages buffers for partitions.
type Manager interface {
	// GetOrCreate returns a buffer for the given partition,
	// creating one if it doesn't exist.
	GetOrCreate(partitionID event.PartitionID) Buffer
}
