// Package buffer implements event buffering for batch processing.
package buffer

import (
	"fmt"
	"sync"
	"time"

	"github.com/jittakal/kafeventstore/internal/errors"
	"github.com/jittakal/kafeventstore/pkg/buffer"
	"github.com/jittakal/kafeventstore/pkg/event"
)

// Ensure implementation satisfies interface at compile time.
var _ buffer.Buffer = (*PartitionBuffer)(nil)

// PartitionBuffer buffers events for a single Kafka partition.
// It provides thread-safe buffering with size limits and record count limits.
// The buffer tracks first and last write times for file rotation decisions.
type PartitionBuffer struct {
	partitionID    event.PartitionID
	records        []event.Record
	maxSizeBytes   int64
	maxRecords     int
	currentSize    int64
	firstWriteTime time.Time
	lastWriteTime  time.Time
	mu             sync.RWMutex
}

// New creates a new partition buffer.
func New(partitionID event.PartitionID, maxSizeBytes int64, maxRecords int) *PartitionBuffer {
	return &PartitionBuffer{
		partitionID:  partitionID,
		records:      make([]event.Record, 0, maxRecords),
		maxSizeBytes: maxSizeBytes,
		maxRecords:   maxRecords,
	}
}

// Add adds a record to the buffer.
func (b *PartitionBuffer) Add(record event.Record) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	recordSize := int64(estimateSize(record))

	if len(b.records) >= b.maxRecords {
		return fmt.Errorf("%w: max records (%d) reached", errors.ErrBufferFull, b.maxRecords)
	}

	if b.maxSizeBytes > 0 && b.currentSize+recordSize > b.maxSizeBytes {
		return fmt.Errorf("%w: max size (%d bytes) would be exceeded", errors.ErrBufferFull, b.maxSizeBytes)
	}

	b.records = append(b.records, record)
	b.currentSize += recordSize

	now := time.Now()
	if b.firstWriteTime.IsZero() {
		b.firstWriteTime = now
	}
	b.lastWriteTime = now

	return nil
}

// Drain removes and returns all records from the buffer.
// The returned slice is owned by the caller and will not be modified by the buffer.
// The caller should process the records promptly as the underlying array may be
// reused after subsequent calls to Add.
func (b *PartitionBuffer) Drain() []event.Record {
	b.mu.Lock()
	defer b.mu.Unlock()

	records := b.records
	b.reset()
	return records
}

// Stats returns current buffer statistics.
func (b *PartitionBuffer) Stats() event.FileStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return event.FileStats{
		RecordCount:    len(b.records),
		SizeBytes:      b.currentSize,
		FirstWriteTime: b.firstWriteTime,
		LastWriteTime:  b.lastWriteTime,
	}
}

// IsEmpty returns true if the buffer is empty.
func (b *PartitionBuffer) IsEmpty() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.records) == 0
}

// Reset clears the buffer and resets all statistics.
func (b *PartitionBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.reset()
}

func (b *PartitionBuffer) reset() {
	b.records = make([]event.Record, 0, b.maxRecords)
	b.currentSize = 0
	b.firstWriteTime = time.Time{}
	b.lastWriteTime = time.Time{}
}

// estimateSize estimates the size of a record in bytes.
func estimateSize(record event.Record) int {
	size := 0

	size += len(record.Event.ID)
	size += len(record.Event.Source)
	size += len(record.Event.SpecVersion)
	size += len(record.Event.Type)

	if record.Event.DataContentType != nil {
		size += len(*record.Event.DataContentType)
	}
	if record.Event.DataSchema != nil {
		size += len(*record.Event.DataSchema)
	}
	if record.Event.Subject != nil {
		size += len(*record.Event.Subject)
	}

	size += len(record.Event.Data)
	size += len(record.Kafka.Topic)
	size += len(record.Kafka.Key)

	for k, v := range record.Kafka.Headers {
		size += len(k) + len(v)
	}

	return size
}

// Manager manages buffers for multiple Kafka partitions.
// It provides thread-safe access to partition-specific buffers, creating them on-demand.
// Uses double-checked locking for efficient concurrent access.
type Manager struct {
	buffers      map[event.PartitionID]*PartitionBuffer
	maxSizeBytes int64
	maxRecords   int
	mu           sync.RWMutex
}

// NewManager creates a new buffer manager.
func NewManager(maxSizeBytes int64, maxRecords int) *Manager {
	return &Manager{
		buffers:      make(map[event.PartitionID]*PartitionBuffer),
		maxSizeBytes: maxSizeBytes,
		maxRecords:   maxRecords,
	}
}

// GetOrCreate returns a buffer for the partition, creating if needed.
func (m *Manager) GetOrCreate(partitionID event.PartitionID) buffer.Buffer {
	m.mu.RLock()
	buf, exists := m.buffers[partitionID]
	m.mu.RUnlock()

	if exists {
		return buf
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if buf, exists := m.buffers[partitionID]; exists {
		return buf
	}

	buf = New(partitionID, m.maxSizeBytes, m.maxRecords)
	m.buffers[partitionID] = buf
	return buf
}
