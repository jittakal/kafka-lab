package buffer

import (
	"testing"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestNew(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	maxSize := int64(1024 * 1024)
	maxRecords := 1000

	buf := New(partitionID, maxSize, maxRecords)

	if buf == nil {
		t.Fatal("expected non-nil buffer")
	}
	if buf.partitionID != partitionID {
		t.Errorf("partitionID = %v, want %v", buf.partitionID, partitionID)
	}
	if buf.maxSizeBytes != maxSize {
		t.Errorf("maxSizeBytes = %d, want %d", buf.maxSizeBytes, maxSize)
	}
	if buf.maxRecords != maxRecords {
		t.Errorf("maxRecords = %d, want %d", buf.maxRecords, maxRecords)
	}
}

func TestPartitionBuffer_Add(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	buf := New(partitionID, 1024*1024, 100)

	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "test-1",
			Source:      "test",
			SpecVersion: "1.0",
			Type:        "test.event",
			Time:        &now,
			Data:        []byte(`{"test": "data"}`),
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    100,
			Timestamp: now,
		},
		Offset:      100,
		ProcessedAt: now,
	}

	err := buf.Add(record)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	stats := buf.Stats()
	if stats.RecordCount != 1 {
		t.Errorf("RecordCount = %d, want 1", stats.RecordCount)
	}
	if stats.SizeBytes == 0 {
		t.Error("expected non-zero size")
	}
}

func TestPartitionBuffer_AddMaxRecords(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	maxRecords := 2
	buf := New(partitionID, 1024*1024, maxRecords)

	now := time.Now()
	for i := 0; i < maxRecords; i++ {
		record := event.Record{
			Event: &event.CloudEvent{
				ID:          "test",
				Source:      "test",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    int64(i),
				Timestamp: now,
			},
			Offset:      int64(i),
			ProcessedAt: now,
		}

		if err := buf.Add(record); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	// Try to add one more - should fail
	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "test",
			Source:      "test",
			SpecVersion: "1.0",
			Type:        "test.event",
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    100,
			Timestamp: now,
		},
	}

	err := buf.Add(record)
	if err == nil {
		t.Error("expected error when exceeding max records")
	}
}

func TestPartitionBuffer_Drain(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	buf := New(partitionID, 1024*1024, 100)

	now := time.Now()
	recordCount := 5
	for i := 0; i < recordCount; i++ {
		record := event.Record{
			Event: &event.CloudEvent{
				ID:          "test",
				Source:      "test",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    int64(i),
				Timestamp: now,
			},
			Offset:      int64(i),
			ProcessedAt: now,
		}

		if err := buf.Add(record); err != nil {
			t.Fatalf("Add() error = %v", err)
		}
	}

	records := buf.Drain()

	if len(records) != recordCount {
		t.Errorf("len(records) = %d, want %d", len(records), recordCount)
	}

	// Buffer should be empty after drain
	if !buf.IsEmpty() {
		t.Error("buffer should be empty after drain")
	}

	stats := buf.Stats()
	if stats.RecordCount != 0 {
		t.Errorf("RecordCount after drain = %d, want 0", stats.RecordCount)
	}
}

func TestPartitionBuffer_IsEmpty(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	buf := New(partitionID, 1024*1024, 100)

	if !buf.IsEmpty() {
		t.Error("new buffer should be empty")
	}

	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "test",
			Source:      "test",
			SpecVersion: "1.0",
			Type:        "test.event",
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    100,
			Timestamp: now,
		},
		Offset:      100,
		ProcessedAt: now,
	}

	buf.Add(record)

	if buf.IsEmpty() {
		t.Error("buffer should not be empty after adding record")
	}

	buf.Drain()

	if !buf.IsEmpty() {
		t.Error("buffer should be empty after drain")
	}
}

func TestPartitionBuffer_ConcurrentAdd(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	buf := New(partitionID, 1024*1024*10, 1000)

	concurrency := 10
	recordsPerGoroutine := 10
	done := make(chan bool, concurrency)

	now := time.Now()

	// Add records concurrently
	for g := 0; g < concurrency; g++ {
		go func(goroutineID int) {
			for i := 0; i < recordsPerGoroutine; i++ {
				record := event.Record{
					Event: &event.CloudEvent{
						ID:          "test",
						Source:      "test",
						SpecVersion: "1.0",
						Type:        "test.event",
					},
					Kafka: event.KafkaMetadata{
						Topic:     "test-topic",
						Partition: 0,
						Offset:    int64(goroutineID*recordsPerGoroutine + i),
						Timestamp: now,
					},
					Offset:      int64(goroutineID*recordsPerGoroutine + i),
					ProcessedAt: now,
				}
				buf.Add(record)
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}

	stats := buf.Stats()
	expectedCount := concurrency * recordsPerGoroutine
	if stats.RecordCount != expectedCount {
		t.Errorf("RecordCount = %d, want %d", stats.RecordCount, expectedCount)
	}
}

func TestPartitionBuffer_ConcurrentDrain(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	buf := New(partitionID, 1024*1024, 100)

	now := time.Now()
	// Add some records
	for i := 0; i < 50; i++ {
		record := event.Record{
			Event: &event.CloudEvent{
				ID:          "test",
				Source:      "test",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    int64(i),
				Timestamp: now,
			},
			Offset:      int64(i),
			ProcessedAt: now,
		}
		buf.Add(record)
	}

	// Drain concurrently
	done := make(chan int, 5)
	for i := 0; i < 5; i++ {
		go func() {
			records := buf.Drain()
			done <- len(records)
		}()
	}

	totalDrained := 0
	for i := 0; i < 5; i++ {
		count := <-done
		totalDrained += count
	}

	// Only one drain should get records, others should get empty
	if totalDrained != 50 {
		t.Errorf("total drained = %d, want 50", totalDrained)
	}
}

func TestPartitionBuffer_SizeLimit(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	maxSize := int64(1000) // Small size to trigger limit
	buf := New(partitionID, maxSize, 1000)

	now := time.Now()
	largeData := make([]byte, 500) // Large payload
	for i := range largeData {
		largeData[i] = byte('x')
	}

	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "test",
			Source:      "test",
			SpecVersion: "1.0",
			Type:        "test.event",
			Data:        largeData,
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    1,
			Timestamp: now,
		},
		Offset:      1,
		ProcessedAt: now,
	}

	// First record should succeed
	err := buf.Add(record)
	if err != nil {
		t.Fatalf("First Add() error = %v", err)
	}

	// Second record should fail due to size limit
	record.Kafka.Offset = 2
	record.Offset = 2
	err = buf.Add(record)
	if err == nil {
		t.Error("expected error when exceeding size limit")
	}
}

func TestPartitionBuffer_Reset(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	buf := New(partitionID, 1024*1024, 100)

	now := time.Now()
	for i := 0; i < 10; i++ {
		record := event.Record{
			Event: &event.CloudEvent{
				ID:          "test",
				Source:      "test",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
			Kafka: event.KafkaMetadata{
				Topic:     "test-topic",
				Partition: 0,
				Offset:    int64(i),
				Timestamp: now,
			},
			Offset:      int64(i),
			ProcessedAt: now,
		}
		buf.Add(record)
	}

	buf.Reset()

	if !buf.IsEmpty() {
		t.Error("buffer should be empty after reset")
	}

	stats := buf.Stats()
	if stats.RecordCount != 0 {
		t.Errorf("RecordCount after reset = %d, want 0", stats.RecordCount)
	}
	if stats.SizeBytes != 0 {
		t.Errorf("SizeBytes after reset = %d, want 0", stats.SizeBytes)
	}
}

func TestPartitionBuffer_FirstLastWriteTime(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	buf := New(partitionID, 1024*1024, 100)

	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "test",
			Source:      "test",
			SpecVersion: "1.0",
			Type:        "test.event",
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    1,
			Timestamp: now,
		},
		Offset:      1,
		ProcessedAt: now,
	}

	buf.Add(record)

	stats := buf.Stats()
	if stats.FirstWriteTime.IsZero() {
		t.Error("FirstWriteTime should not be zero")
	}
	if stats.LastWriteTime.IsZero() {
		t.Error("LastWriteTime should not be zero")
	}

	// Add another record
	time.Sleep(10 * time.Millisecond)
	record.Kafka.Offset = 2
	record.Offset = 2
	buf.Add(record)

	stats2 := buf.Stats()
	if !stats2.LastWriteTime.After(stats.LastWriteTime) {
		t.Error("LastWriteTime should be updated")
	}
	if stats2.FirstWriteTime != stats.FirstWriteTime {
		t.Error("FirstWriteTime should not change")
	}
}

func TestEstimateSize(t *testing.T) {
	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "test-id",
			Source:      "test-source",
			SpecVersion: "1.0",
			Type:        "test.event",
			Data:        []byte(`{"test": "data"}`),
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    100,
			Timestamp: now,
		},
		Offset:      100,
		ProcessedAt: now,
	}

	size := estimateSize(record)
	if size <= 0 {
		t.Error("estimated size should be positive")
	}

	// Verify size includes data
	expectedMinSize := len(record.Event.Data)
	if size < expectedMinSize {
		t.Errorf("estimated size %d should be at least %d", size, expectedMinSize)
	}
}

func TestPartitionBuffer_Stats(t *testing.T) {
	partitionID := event.PartitionID{Topic: "test-topic", Partition: 0}
	buf := New(partitionID, 1024*1024, 100)

	// Empty buffer stats
	stats := buf.Stats()
	if stats.RecordCount != 0 {
		t.Errorf("empty buffer RecordCount = %d, want 0", stats.RecordCount)
	}
	if stats.SizeBytes != 0 {
		t.Errorf("empty buffer SizeBytes = %d, want 0", stats.SizeBytes)
	}

	// Add records and check stats
	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "test",
			Source:      "test",
			SpecVersion: "1.0",
			Type:        "test.event",
			Data:        []byte(`{"test": "data"}`),
		},
		Kafka: event.KafkaMetadata{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    100,
			Timestamp: now,
		},
		Offset:      100,
		ProcessedAt: now,
	}

	buf.Add(record)

	stats = buf.Stats()
	if stats.RecordCount != 1 {
		t.Errorf("RecordCount = %d, want 1", stats.RecordCount)
	}
	if stats.SizeBytes == 0 {
		t.Error("expected non-zero size")
	}
	if stats.FirstWriteTime.IsZero() {
		t.Error("expected non-zero FirstWriteTime")
	}
	if stats.LastWriteTime.IsZero() {
		t.Error("expected non-zero LastWriteTime")
	}
}

// Benchmark tests for hot paths

func BenchmarkPartitionBuffer_Add(b *testing.B) {
	partitionID := event.PartitionID{Topic: "benchmark-topic", Partition: 0}
	buf := New(partitionID, 1024*1024*100, 100000) // Large buffer to avoid full condition

	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "bench-1",
			Source:      "benchmark",
			SpecVersion: "1.0",
			Type:        "bench.event",
			Time:        &now,
			Data:        []byte(`{"benchmark": "data with some reasonable payload size to simulate real events"}`),
		},
		Kafka: event.KafkaMetadata{
			Topic:     "benchmark-topic",
			Partition: 0,
			Offset:    1,
			Timestamp: now,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		record.Kafka.Offset = int64(i)
		record.Event.ID = string(rune(i))
		if err := buf.Add(record); err != nil {
			b.Fatal(err)
		}
		// Drain periodically to avoid buffer full
		if i%1000 == 999 {
			buf.Drain()
		}
	}
}

func BenchmarkPartitionBuffer_Drain(b *testing.B) {
	partitionID := event.PartitionID{Topic: "benchmark-topic", Partition: 0}

	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "bench-1",
			Source:      "benchmark",
			SpecVersion: "1.0",
			Type:        "bench.event",
			Time:        &now,
			Data:        []byte(`{"benchmark": "data"}`),
		},
		Kafka: event.KafkaMetadata{
			Topic:     "benchmark-topic",
			Partition: 0,
			Offset:    1,
			Timestamp: now,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		buf := New(partitionID, 1024*1024, 1000)
		// Add 100 records
		for j := 0; j < 100; j++ {
			buf.Add(record)
		}
		b.StartTimer()

		records := buf.Drain()

		if len(records) != 100 {
			b.Fatalf("expected 100 records, got %d", len(records))
		}
	}
}

func BenchmarkPartitionBuffer_Stats(b *testing.B) {
	partitionID := event.PartitionID{Topic: "benchmark-topic", Partition: 0}
	buf := New(partitionID, 1024*1024, 1000)

	now := time.Now()
	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "bench-1",
			Source:      "benchmark",
			SpecVersion: "1.0",
			Type:        "bench.event",
			Time:        &now,
			Data:        []byte(`{"benchmark": "data"}`),
		},
		Kafka: event.KafkaMetadata{
			Topic:     "benchmark-topic",
			Partition: 0,
			Offset:    1,
			Timestamp: now,
		},
	}

	// Pre-populate buffer
	for i := 0; i < 50; i++ {
		buf.Add(record)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.Stats()
	}
}

func BenchmarkManager_GetOrCreate(b *testing.B) {
	manager := NewManager(1024*1024, 1000)

	partitionIDs := make([]event.PartitionID, 10)
	for i := 0; i < 10; i++ {
		partitionIDs[i] = event.PartitionID{Topic: "benchmark-topic", Partition: int32(i)}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pid := partitionIDs[i%10]
		_ = manager.GetOrCreate(pid)
	}
}

func BenchmarkManager_GetOrCreate_Parallel(b *testing.B) {
	manager := NewManager(1024*1024, 1000)

	partitionID := event.PartitionID{Topic: "benchmark-topic", Partition: 0}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = manager.GetOrCreate(partitionID)
		}
	})
}
