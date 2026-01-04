// Package validator provides CloudEvents validation.
package validator

import (
	"fmt"

	"github.com/jittakal/kafeventstore/internal/errors"
	"github.com/jittakal/kafeventstore/pkg/event"
)

// CloudEventsValidator validates CloudEvents according to the specification.
type CloudEventsValidator struct{}

// NewCloudEventsValidator creates a new CloudEvents validator.
func NewCloudEventsValidator() *CloudEventsValidator {
	return &CloudEventsValidator{}
}

// Validate validates a CloudEvent.
func (v *CloudEventsValidator) Validate(e *event.CloudEvent) error {
	if e.ID == "" {
		return &errors.ValidationError{
			EventID: e.ID,
			Field:   "id",
			Reason:  "required field is missing",
		}
	}

	if e.Source == "" {
		return &errors.ValidationError{
			EventID: e.ID,
			Field:   "source",
			Reason:  "required field is missing",
		}
	}

	if e.SpecVersion == "" {
		return &errors.ValidationError{
			EventID: e.ID,
			Field:   "specversion",
			Reason:  "required field is missing",
		}
	}

	if e.Type == "" {
		return &errors.ValidationError{
			EventID: e.ID,
			Field:   "type",
			Reason:  "required field is missing",
		}
	}

	// Normalize spec version (0.1 -> 1.0)
	if e.SpecVersion == "0.1" {
		e.SpecVersion = "1.0"
	}

	// Validate spec version
	if e.SpecVersion != "1.0" {
		return &errors.ValidationError{
			EventID: e.ID,
			Field:   "specversion",
			Reason:  fmt.Sprintf("unsupported version: %s (supported: 1.0)", e.SpecVersion),
		}
	}

	return nil
}
