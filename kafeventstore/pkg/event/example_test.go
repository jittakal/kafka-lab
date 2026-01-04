package event_test

import (
	"fmt"
	"time"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func ExamplePartitionID_String() {
	pid := event.PartitionID{
		Topic:     "user-events",
		Partition: 5,
	}

	fmt.Println(pid.String())
	// Output: user-events-5
}

func ExampleRecord_GetEventTime() {
	now := time.Date(2025, 12, 21, 10, 30, 0, 0, time.UTC)

	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "evt-123",
			Source:      "user-service",
			SpecVersion: "1.0",
			Type:        "user.created",
			Time:        &now,
		},
		Kafka: event.KafkaMetadata{
			Topic:     "user-events",
			Partition: 0,
			Offset:    42,
			Timestamp: now,
		},
	}

	eventTime := record.GetEventTime()
	fmt.Println(eventTime.Format("2006-01-02 15:04:05"))
	// Output: 2025-12-21 10:30:00
}

func ExampleRecord_GetEventTimeUnix() {
	now := time.Date(2025, 12, 21, 10, 30, 0, 0, time.UTC)

	record := event.Record{
		Event: &event.CloudEvent{
			ID:          "evt-123",
			Source:      "user-service",
			SpecVersion: "1.0",
			Type:        "user.created",
			Time:        &now,
		},
		Kafka: event.KafkaMetadata{
			Timestamp: now,
		},
	}

	unixTime := record.GetEventTimeUnix()
	fmt.Println(unixTime)
	// Output: 1766313000
}
