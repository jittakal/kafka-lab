// Package errors defines application-specific error types and sentinel errors.
package errors

import (
	"errors"
	"fmt"

	"github.com/jittakal/kafeventstore/pkg/event"
)

// Sentinel errors for common conditions.
var (
	ErrBufferFull      = errors.New("buffer is full")
	ErrConsumerClosed  = errors.New("consumer is closed")
	ErrInvalidEvent    = errors.New("invalid event")
	ErrOffsetNotFound  = errors.New("offset not found")
	ErrPartitionClosed = errors.New("partition processor is closed")
	ErrWriterClosed    = errors.New("storage writer is closed")
	ErrConnectionLost  = errors.New("connection lost")
)

// ProcessingError represents an error during event processing.
type ProcessingError struct {
	PartitionID event.PartitionID
	Offset      int64
	EventID     string
	Err         error
}

func (e *ProcessingError) Error() string {
	return fmt.Sprintf("processing error: partition=%s offset=%d event_id=%s: %v",
		e.PartitionID, e.Offset, e.EventID, e.Err)
}

func (e *ProcessingError) Unwrap() error {
	return e.Err
}

// ValidationError represents an event validation failure.
type ValidationError struct {
	EventID string
	Field   string
	Reason  string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: event_id=%s field=%s: %s",
		e.EventID, e.Field, e.Reason)
}

// StorageError represents a storage operation failure.
type StorageError struct {
	Operation string
	Path      string
	Err       error
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage error: operation=%s path=%s: %v",
		e.Operation, e.Path, e.Err)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

// CommitError represents an offset commit failure.
type CommitError struct {
	PartitionID event.PartitionID
	Offset      int64
	Err         error
}

func (e *CommitError) Error() string {
	return fmt.Sprintf("commit error: partition=%s offset=%d: %v",
		e.PartitionID, e.Offset, e.Err)
}

func (e *CommitError) Unwrap() error {
	return e.Err
}

// Retryable defines an interface for errors that can indicate if they are retryable.
type Retryable interface {
	error
	IsRetryable() bool
}

// IsRetryable checks if an error is retryable.
// It first checks if the error implements the Retryable interface,
// then falls back to checking specific error types and sentinel errors.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if error implements Retryable interface
	var retryable Retryable
	if errors.As(err, &retryable) {
		return retryable.IsRetryable()
	}

	// Check specific error types
	var storageErr *StorageError
	if errors.As(err, &storageErr) {
		return storageErr.IsRetryable()
	}

	// Check sentinel errors
	if errors.Is(err, ErrConnectionLost) {
		return true
	}

	return false
}

// IsRetryable determines if a StorageError is retryable based on the operation type.
func (e *StorageError) IsRetryable() bool {
	// Write and upload operations are generally retryable
	return e.Operation == "write" || e.Operation == "upload" || e.Operation == "create"
}

// IsRetryable determines if a ProcessingError is retryable.
func (e *ProcessingError) IsRetryable() bool {
	// Check if the underlying error is retryable
	return IsRetryable(e.Err)
}
