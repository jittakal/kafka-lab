package buffer_test

import (
	"fmt"
	"time"

	"github.com/jittakal/kafeventstore/internal/buffer"
	"github.com/jittakal/kafeventstore/pkg/event"
)

func Example_partitionBuffer() {
	// Create a partition buffer with 1MB max size and 1000 max records
	partitionID := event.PartitionID{Topic: "orders", Partition: 0}
	buf := buffer.New(partitionID, 1024*1024, 1000)

	// Add records to the buffer
	now := time.Now()
	for i := 0; i < 5; i++ {
		record := event.Record{
			Event: &event.CloudEvent{
				ID:          fmt.Sprintf("order-%d", i),
				Source:      "order-service",
				SpecVersion: "1.0",
				Type:        "order.created",
				Time:        &now,
				Data:        []byte(fmt.Sprintf(`{"orderId": %d}`, i)),
			},
			Kafka: event.KafkaMetadata{
				Topic:     "orders",
				Partition: 0,
				Offset:    int64(i),
				Timestamp: now,
			},
		}

		if err := buf.Add(record); err != nil {
			fmt.Println("Error adding record:", err)
			return
		}
	}

	// Get buffer statistics
	stats := buf.Stats()
	fmt.Printf("Records buffered: %d\n", stats.RecordCount)
	fmt.Printf("Buffer is empty: %v\n", buf.IsEmpty())

	// Drain the buffer
	records := buf.Drain()
	fmt.Printf("Drained %d records\n", len(records))
	fmt.Printf("Buffer is empty after drain: %v\n", buf.IsEmpty())

	// Output:
	// Records buffered: 5
	// Buffer is empty: false
	// Drained 5 records
	// Buffer is empty after drain: true
}

func Example_bufferManager() {
	// Create a manager for handling multiple partition buffers
	manager := buffer.NewManager(1024*1024, 1000)

	// Get or create buffers for different partitions
	buf0 := manager.GetOrCreate(event.PartitionID{Topic: "orders", Partition: 0})
	buf1 := manager.GetOrCreate(event.PartitionID{Topic: "orders", Partition: 1})

	fmt.Printf("Buffer 0 and Buffer 1 are different: %v\n", buf0 != buf1)

	// Getting the same partition returns the same buffer
	buf0Again := manager.GetOrCreate(event.PartitionID{Topic: "orders", Partition: 0})
	fmt.Printf("Getting partition 0 again returns same buffer: %v\n", buf0 == buf0Again)

	// Output:
	// Buffer 0 and Buffer 1 are different: true
	// Getting partition 0 again returns same buffer: true
}
