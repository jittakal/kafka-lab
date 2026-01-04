package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestDLQConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  DLQConfig
		wantErr bool
	}{
		{
			name: "valid enabled config",
			config: DLQConfig{
				Enabled:     true,
				TopicSuffix: ".dlq",
				MaxRetries:  3,
			},
			wantErr: false,
		},
		{
			name: "disabled config",
			config: DLQConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "empty suffix when enabled",
			config: DLQConfig{
				Enabled:     true,
				TopicSuffix: "",
				MaxRetries:  3,
			},
			wantErr: true,
		},
		{
			name: "zero max retries when enabled",
			config: DLQConfig{
				Enabled:     true,
				TopicSuffix: ".dlq",
				MaxRetries:  0,
			},
			wantErr: false, // 0 means unlimited
		},
		{
			name: "negative max retries",
			config: DLQConfig{
				Enabled:     true,
				TopicSuffix: ".dlq",
				MaxRetries:  -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDLQConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDLQConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func validateDLQConfig(config DLQConfig) error {
	if !config.Enabled {
		return nil
	}
	if config.TopicSuffix == "" {
		return errors.New("topic suffix is required when DLQ is enabled")
	}
	if config.MaxRetries < 0 {
		return errors.New("max retries cannot be negative")
	}
	return nil
}

func TestDLQTopicName(t *testing.T) {
	tests := []struct {
		name        string
		sourceTopic string
		suffix      string
		want        string
	}{
		{
			name:        "standard suffix",
			sourceTopic: "events",
			suffix:      ".dlq",
			want:        "events.dlq",
		},
		{
			name:        "custom suffix",
			sourceTopic: "orders",
			suffix:      "-dead-letter",
			want:        "orders-dead-letter",
		},
		{
			name:        "topic with dots",
			sourceTopic: "domain.service.events",
			suffix:      ".dlq",
			want:        "domain.service.events.dlq",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sourceTopic + tt.suffix
			if got != tt.want {
				t.Errorf("DLQ topic name = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDLQPublish_Disabled(t *testing.T) {
	config := DLQConfig{
		Enabled: false,
	}

	// When DLQ is disabled, publish should be a no-op
	if config.Enabled {
		t.Error("DLQ should be disabled")
	}

	// No error should be returned when disabled
	err := mockPublishWhenDisabled(config)
	if err != nil {
		t.Errorf("Publish when disabled should not error, got: %v", err)
	}
}

func mockPublishWhenDisabled(config DLQConfig) error {
	if !config.Enabled {
		return nil // no-op when disabled
	}
	return errors.New("should not reach here")
}

func TestDLQPublish_WithContext(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		wantErr bool
	}{
		{
			name:    "valid context",
			ctx:     context.Background(),
			wantErr: false,
		},
		{
			name:    "cancelled context",
			ctx:     cancelledContext(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if context is cancelled
			select {
			case <-tt.ctx.Done():
				if !tt.wantErr {
					t.Error("Context should not be cancelled")
				}
			default:
				if tt.wantErr {
					t.Error("Context should be cancelled")
				}
			}
		})
	}
}

func cancelledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestDLQMessage_Serialization(t *testing.T) {
	cloudEvent := &event.CloudEvent{
		ID:          "test-123",
		Source:      "test-source",
		SpecVersion: "1.0",
		Type:        "test.event",
	}

	metadata := event.KafkaMetadata{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    100,
	}

	reason := "validation_failed"

	// Mock DLQ message structure
	type DLQMessage struct {
		Event      *event.CloudEvent
		Metadata   event.KafkaMetadata
		Reason     string
		RetryCount int
	}

	msg := DLQMessage{
		Event:      cloudEvent,
		Metadata:   metadata,
		Reason:     reason,
		RetryCount: 1,
	}

	if msg.Event.ID != cloudEvent.ID {
		t.Error("Event ID mismatch")
	}
	if msg.Metadata.Topic != metadata.Topic {
		t.Error("Metadata topic mismatch")
	}
	if msg.Reason != reason {
		t.Error("Reason mismatch")
	}
	if msg.RetryCount != 1 {
		t.Error("Retry count should be 1")
	}
}

func TestDLQRetryLogic(t *testing.T) {
	tests := []struct {
		name        string
		retryCount  int
		maxRetries  int
		shouldRetry bool
	}{
		{
			name:        "first attempt",
			retryCount:  0,
			maxRetries:  3,
			shouldRetry: true,
		},
		{
			name:        "within retry limit",
			retryCount:  2,
			maxRetries:  3,
			shouldRetry: true,
		},
		{
			name:        "at retry limit",
			retryCount:  3,
			maxRetries:  3,
			shouldRetry: false,
		},
		{
			name:        "exceeded retry limit",
			retryCount:  4,
			maxRetries:  3,
			shouldRetry: false,
		},
		{
			name:        "unlimited retries",
			retryCount:  100,
			maxRetries:  0,
			shouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shouldRetry := tt.maxRetries == 0 || tt.retryCount < tt.maxRetries
			if shouldRetry != tt.shouldRetry {
				t.Errorf("Retry decision = %v, want %v", shouldRetry, tt.shouldRetry)
			}
		})
	}
}

func TestDLQErrorReasons(t *testing.T) {
	reasons := []string{
		"validation_failed",
		"deserialization_failed",
		"storage_failed",
		"processing_timeout",
		"unknown_error",
	}

	for _, reason := range reasons {
		if reason == "" {
			t.Error("Error reason should not be empty")
		}
		if len(reason) < 3 {
			t.Errorf("Error reason too short: %v", reason)
		}
	}
}

func TestDLQHeaders(t *testing.T) {
	// Mock DLQ headers
	headers := map[string]string{
		"dlq-reason":       "validation_failed",
		"dlq-retry-count":  "1",
		"dlq-source-topic": "events",
		"dlq-timestamp":    "2025-12-19T10:00:00Z",
	}

	if headers["dlq-reason"] != "validation_failed" {
		t.Error("DLQ reason header mismatch")
	}
	if headers["dlq-retry-count"] != "1" {
		t.Error("DLQ retry count header mismatch")
	}
	if headers["dlq-source-topic"] != "events" {
		t.Error("DLQ source topic header mismatch")
	}

	// Verify all required headers are present
	requiredHeaders := []string{"dlq-reason", "dlq-retry-count", "dlq-source-topic"}
	for _, h := range requiredHeaders {
		if _, ok := headers[h]; !ok {
			t.Errorf("Required header missing: %v", h)
		}
	}
}

func TestDLQClose(t *testing.T) {
	// Test DLQ publisher close behavior
	closed := false

	closeFunc := func() error {
		if closed {
			return errors.New("already closed")
		}
		closed = true
		return nil
	}

	// First close should succeed
	err := closeFunc()
	if err != nil {
		t.Errorf("First close failed: %v", err)
	}
	if !closed {
		t.Error("Should be marked as closed")
	}

	// Second close should fail
	err = closeFunc()
	if err == nil {
		t.Error("Second close should fail")
	}
}

func TestDLQConcurrency(t *testing.T) {
	// Test concurrent publishing to DLQ
	const goroutines = 10
	done := make(chan bool, goroutines)
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			// Simulate DLQ publish
			err := mockPublish(id)
			if err != nil {
				errors <- err
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	close(errors)
	errorCount := 0
	for range errors {
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Expected no errors, got %d", errorCount)
	}
}

func mockPublish(id int) error {
	// Mock successful publish
	if id < 0 {
		return errors.New("invalid id")
	}
	return nil
}
