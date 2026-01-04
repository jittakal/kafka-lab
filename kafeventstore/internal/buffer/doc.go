// Package buffer provides thread-safe buffering for event records.
//
// This package implements in-memory buffering with automatic size and count limits,
// designed for batching events before writing to storage.
//
// # PartitionBuffer
//
// PartitionBuffer is a thread-safe buffer for a single Kafka partition:
//
//	buf := buffer.New(partitionID, maxSizeBytes, maxRecords)
//
//	// Add records until buffer is full
//	for _, record := range records {
//	    if err := buf.Add(record); err != nil {
//	        if errors.Is(err, buffer.ErrBufferFull) {
//	            // Buffer is full, drain and process
//	            records := buf.Drain()
//	            processRecords(records)
//	        }
//	    }
//	}
//
// # Buffer Manager
//
// Manager handles multiple partition buffers with automatic creation:
//
//	manager := buffer.NewManager(maxSizeBytes, maxRecords)
//
//	// Get or create buffer for partition
//	buf := manager.GetOrCreate(partitionID)
//
//	// Manager automatically creates new buffers as needed
//	// and maintains them in a thread-safe map
//
// # Buffer Lifecycle
//
// 1. Create: Initialize buffer with size and count limits
//
//	buf := buffer.New(partitionID, 1024*1024, 1000)
//
// 2. Add: Add records until buffer is full
//
//	err := buf.Add(record)
//	if errors.Is(err, ErrBufferFull) {
//	    // Handle full buffer
//	}
//
// 3. Check: Query buffer state
//
//	stats := buf.Stats()
//	isEmpty := buf.IsEmpty()
//
// 4. Drain: Extract all records (resets buffer)
//
//	records := buf.Drain()
//	// Buffer is now empty and ready for reuse
//
// # Thread Safety
//
// All buffer operations are thread-safe using read-write mutexes:
//
//   - Add(), Drain(), Reset() use write locks
//   - Stats(), IsEmpty() use read locks
//   - Manager.GetOrCreate() uses double-checked locking
//
// # Memory Management
//
// Buffers pre-allocate slices with capacity equal to maxRecords to minimize
// allocations during normal operation. The Drain() method returns the internal
// slice, which is safe to use but should be processed promptly.
//
// # Statistics
//
// FileStats provides buffer metrics:
//
//	stats := buf.Stats()
//	fmt.Printf("Records: %d, Size: %d bytes\n",
//	    stats.RecordCount, stats.SizeBytes)
//	fmt.Printf("First write: %v\n", stats.FirstWriteTime)
//	fmt.Printf("Last write: %v\n", stats.LastWriteTime)
package buffer
