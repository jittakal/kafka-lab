package event

import (
	"testing"
	"time"
)

func TestPartitionID_String(t *testing.T) {
	tests := []struct {
		name      string
		partition PartitionID
		want      string
	}{
		{
			name:      "basic partition",
			partition: PartitionID{Topic: "test-topic", Partition: 0},
			want:      "test-topic-0",
		},
		{
			name:      "partition 1",
			partition: PartitionID{Topic: "events", Partition: 1},
			want:      "events-1",
		},
		{
			name:      "partition 10",
			partition: PartitionID{Topic: "my-topic", Partition: 10},
			want:      "my-topic-10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.partition.String(); got != tt.want {
				t.Errorf("PartitionID.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloudEvent_Validation(t *testing.T) {
	now := time.Now()
	contentType := "application/json"

	tests := []struct {
		name    string
		event   *CloudEvent
		wantErr bool
	}{
		{
			name: "valid event",
			event: &CloudEvent{
				ID:          "test-id-123",
				Source:      "test-source",
				SpecVersion: "1.0",
				Type:        "test.event",
				Time:        &now,
				Data:        []byte(`{"key":"value"}`),
			},
			wantErr: false,
		},
		{
			name: "valid event with extensions",
			event: &CloudEvent{
				ID:              "test-id-456",
				Source:          "test-source-2",
				SpecVersion:     "1.0",
				Type:            "test.event.extended",
				DataContentType: &contentType,
				Extensions:      map[string]interface{}{"custom": "value"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.event == nil {
				t.Error("Event should not be nil")
			}
			if tt.event.ID == "" {
				t.Error("Event ID should not be empty")
			}
			if tt.event.Source == "" {
				t.Error("Event Source should not be empty")
			}
			if tt.event.SpecVersion == "" {
				t.Error("Event SpecVersion should not be empty")
			}
			if tt.event.Type == "" {
				t.Error("Event Type should not be empty")
			}
		})
	}
}

func TestRecord_Creation(t *testing.T) {
	now := time.Now()
	event := &CloudEvent{
		ID:          "record-test-1",
		Source:      "test",
		SpecVersion: "1.0",
		Type:        "test.record",
		Data:        []byte("test data"),
	}

	metadata := KafkaMetadata{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    100,
		Timestamp: now,
		Key:       []byte("key"),
	}

	record := Record{
		Event:       event,
		Kafka:       metadata,
		Offset:      100,
		ProcessedAt: now,
	}

	if record.Event == nil {
		t.Error("Record Event should not be nil")
	}
	if record.Event.ID != "record-test-1" {
		t.Errorf("Record Event ID = %v, want record-test-1", record.Event.ID)
	}
	if record.Kafka.Topic != "test-topic" {
		t.Errorf("Record Kafka Topic = %v, want test-topic", record.Kafka.Topic)
	}
	if record.Offset != 100 {
		t.Errorf("Record Offset = %v, want 100", record.Offset)
	}
}

func TestFileStats(t *testing.T) {
	stats := FileStats{
		RecordCount:    10,
		SizeBytes:      1024,
		FirstWriteTime: time.Now(),
		LastWriteTime:  time.Now(),
	}

	if stats.RecordCount != 10 {
		t.Errorf("RecordCount = %v, want 10", stats.RecordCount)
	}
	if stats.SizeBytes != 1024 {
		t.Errorf("SizeBytes = %v, want 1024", stats.SizeBytes)
	}
}

func TestFileFormat(t *testing.T) {
	tests := []struct {
		name   string
		format FileFormat
		want   string
	}{
		{
			name:   "parquet format",
			format: FormatParquet,
			want:   "parquet",
		},
		{
			name:   "avro format",
			format: FormatAvro,
			want:   "avro",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.format) != tt.want {
				t.Errorf("FileFormat = %v, want %v", tt.format, tt.want)
			}
		})
	}
}

func TestConsumedEvent(t *testing.T) {
	event := &CloudEvent{
		ID:          "consumed-1",
		Source:      "test",
		SpecVersion: "1.0",
		Type:        "test.consumed",
	}

	metadata := KafkaMetadata{
		Topic:     "consumed-topic",
		Partition: 5,
		Offset:    200,
	}

	committed := false
	commitFunc := func() error {
		committed = true
		return nil
	}

	consumed := ConsumedEvent{
		Event:      event,
		Metadata:   metadata,
		CommitFunc: commitFunc,
	}

	if consumed.Event == nil {
		t.Error("ConsumedEvent Event should not be nil")
	}
	if consumed.Metadata.Topic != "consumed-topic" {
		t.Errorf("ConsumedEvent Topic = %v, want consumed-topic", consumed.Metadata.Topic)
	}
	if consumed.CommitFunc == nil {
		t.Error("ConsumedEvent CommitFunc should not be nil")
	}

	// Test commit function
	if err := consumed.CommitFunc(); err != nil {
		t.Errorf("CommitFunc returned error: %v", err)
	}
	if !committed {
		t.Error("CommitFunc should have been called")
	}
}

func TestRecord_GetEventTime(t *testing.T) {
	kafkaTime := time.Date(2025, 12, 18, 10, 0, 0, 0, time.UTC)
	eventTime := time.Date(2025, 12, 18, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		record Record
		want   time.Time
	}{
		{
			name: "uses CloudEvent.Time when present",
			record: Record{
				Event: &CloudEvent{
					ID:          "test-1",
					Source:      "test",
					SpecVersion: "1.0",
					Type:        "test.event",
					Time:        &eventTime,
				},
				Kafka: KafkaMetadata{
					Timestamp: kafkaTime,
				},
			},
			want: eventTime,
		},
		{
			name: "falls back to Kafka timestamp when CloudEvent.Time is nil",
			record: Record{
				Event: &CloudEvent{
					ID:          "test-2",
					Source:      "test",
					SpecVersion: "1.0",
					Type:        "test.event",
					Time:        nil,
				},
				Kafka: KafkaMetadata{
					Timestamp: kafkaTime,
				},
			},
			want: kafkaTime,
		},
		{
			name: "falls back to Kafka timestamp when Event is nil",
			record: Record{
				Event: nil,
				Kafka: KafkaMetadata{
					Timestamp: kafkaTime,
				},
			},
			want: kafkaTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.record.GetEventTime()
			if !got.Equal(tt.want) {
				t.Errorf("GetEventTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecord_GetEventTimeUnix(t *testing.T) {
	eventTime := time.Date(2025, 12, 18, 9, 30, 0, 0, time.UTC)
	expectedUnix := eventTime.Unix()

	record := Record{
		Event: &CloudEvent{
			ID:          "test-unix",
			Source:      "test",
			SpecVersion: "1.0",
			Type:        "test.event",
			Time:        &eventTime,
		},
		Kafka: KafkaMetadata{
			Timestamp: time.Now(),
		},
	}

	got := record.GetEventTimeUnix()
	if got != expectedUnix {
		t.Errorf("GetEventTimeUnix() = %v, want %v", got, expectedUnix)
	}
}

// Benchmark tests

func BenchmarkPartitionID_String(b *testing.B) {
	pid := PartitionID{Topic: "benchmark-topic-with-long-name", Partition: 42}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pid.String()
	}
}

func BenchmarkPartitionID_String_Parallel(b *testing.B) {
	pid := PartitionID{Topic: "benchmark-topic-with-long-name", Partition: 42}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = pid.String()
		}
	})
}

func BenchmarkRecord_GetEventTime(b *testing.B) {
	now := time.Now()
	record := Record{
		Event: &CloudEvent{
			ID:          "bench-1",
			Source:      "benchmark",
			SpecVersion: "1.0",
			Type:        "bench.event",
			Time:        &now,
		},
		Kafka: KafkaMetadata{
			Timestamp: now,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = record.GetEventTime()
	}
}

func BenchmarkRecord_GetEventTimeUnix(b *testing.B) {
	now := time.Now()
	record := Record{
		Event: &CloudEvent{
			ID:          "bench-1",
			Source:      "benchmark",
			SpecVersion: "1.0",
			Type:        "bench.event",
			Time:        &now,
		},
		Kafka: KafkaMetadata{
			Timestamp: now,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = record.GetEventTimeUnix()
	}
}
