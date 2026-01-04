package errors

import (
	"errors"
	"testing"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrConsumerClosed", ErrConsumerClosed},
		{"ErrInvalidEvent", ErrInvalidEvent},
		{"ErrOffsetNotFound", ErrOffsetNotFound},
		{"ErrBufferFull", ErrBufferFull},
		{"ErrPartitionClosed", ErrPartitionClosed},
		{"ErrWriterClosed", ErrWriterClosed},
		{"ErrConnectionLost", ErrConnectionLost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s should have an error message", tt.name)
			}
		})
	}
}

func TestProcessingError(t *testing.T) {
	baseErr := errors.New("base error")
	procErr := &ProcessingError{
		PartitionID: event.PartitionID{Topic: "test", Partition: 0},
		Offset:      100,
		EventID:     "event-123",
		Err:         baseErr,
	}

	if procErr.Error() == "" {
		t.Error("ProcessingError should have an error message")
	}

	if !errors.Is(procErr, baseErr) {
		t.Error("ProcessingError should wrap base error")
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{
		EventID: "test-123",
		Field:   "source",
		Reason:  "required field missing",
	}

	if err.Error() == "" {
		t.Error("ValidationError should have an error message")
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("ValidationError should have error message")
	}
}

func TestStorageError(t *testing.T) {
	baseErr := errors.New("disk full")
	storageErr := &StorageError{
		Operation: "write",
		Path:      "/data/file.parquet",
		Err:       baseErr,
	}

	if storageErr.Error() == "" {
		t.Error("StorageError should have an error message")
	}

	if !errors.Is(storageErr, baseErr) {
		t.Error("StorageError should wrap base error")
	}
}

func TestCommitError(t *testing.T) {
	baseErr := errors.New("commit failed")
	commitErr := &CommitError{
		PartitionID: event.PartitionID{Topic: "test", Partition: 0},
		Offset:      200,
		Err:         baseErr,
	}

	if commitErr.Error() == "" {
		t.Error("CommitError should have an error message")
	}

	if !errors.Is(commitErr, baseErr) {
		t.Error("CommitError should wrap base error")
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "storage error is retryable",
			err:  &StorageError{Operation: "write", Path: "/tmp/file", Err: errors.New("failed")},
			want: true,
		},
		{
			name: "connection lost is retryable",
			err:  ErrConnectionLost,
			want: true,
		},
		{
			name: "validation error is not retryable",
			err:  &ValidationError{EventID: "123", Field: "source", Reason: "missing"},
			want: false,
		},
		{
			name: "generic error is not retryable",
			err:  errors.New("generic error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}
