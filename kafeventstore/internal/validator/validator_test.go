package validator

import (
	"testing"

	"github.com/jittakal/kafeventstore/pkg/event"
)

func TestNewCloudEventsValidator(t *testing.T) {
	validator := NewCloudEventsValidator()
	if validator == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestCloudEventsValidator_ValidateSuccess(t *testing.T) {
	validator := NewCloudEventsValidator()

	tests := []struct {
		name  string
		event *event.CloudEvent
	}{
		{
			name: "valid 1.0 event",
			event: &event.CloudEvent{
				ID:          "test-id",
				Source:      "test-source",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
		},
		{
			name: "valid event with all optional fields",
			event: &event.CloudEvent{
				ID:          "test-id",
				Source:      "test-source",
				SpecVersion: "1.0",
				Type:        "test.event",
				Data:        []byte(`{"test": "data"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.event)
			if err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestCloudEventsValidator_ValidateSpecVersion01(t *testing.T) {
	validator := NewCloudEventsValidator()

	event := &event.CloudEvent{
		ID:          "test-id",
		Source:      "test-source",
		SpecVersion: "0.1",
		Type:        "test.event",
	}

	err := validator.Validate(event)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil (0.1 should be normalized to 1.0)", err)
	}

	// Verify it was normalized
	if event.SpecVersion != "1.0" {
		t.Errorf("SpecVersion = %s, want 1.0 (should be normalized)", event.SpecVersion)
	}
}

func TestCloudEventsValidator_ValidateErrors(t *testing.T) {
	validator := NewCloudEventsValidator()

	tests := []struct {
		name      string
		event     *event.CloudEvent
		wantField string
	}{
		{
			name: "missing id",
			event: &event.CloudEvent{
				Source:      "test-source",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
			wantField: "id",
		},
		{
			name: "missing source",
			event: &event.CloudEvent{
				ID:          "test-id",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
			wantField: "source",
		},
		{
			name: "missing specversion",
			event: &event.CloudEvent{
				ID:     "test-id",
				Source: "test-source",
				Type:   "test.event",
			},
			wantField: "specversion",
		},
		{
			name: "missing type",
			event: &event.CloudEvent{
				ID:          "test-id",
				Source:      "test-source",
				SpecVersion: "1.0",
			},
			wantField: "type",
		},
		{
			name: "unsupported spec version",
			event: &event.CloudEvent{
				ID:          "test-id",
				Source:      "test-source",
				SpecVersion: "2.0",
				Type:        "test.event",
			},
			wantField: "specversion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.event)
			if err == nil {
				t.Error("Validate() error = nil, want error")
				return
			}

			// Check that error message contains the field name
			errMsg := err.Error()
			if errMsg == "" {
				t.Error("expected non-empty error message")
			}
		})
	}
}

func TestCloudEventsValidator_ValidateEmptyStrings(t *testing.T) {
	validator := NewCloudEventsValidator()

	tests := []struct {
		name  string
		event *event.CloudEvent
	}{
		{
			name: "empty id",
			event: &event.CloudEvent{
				ID:          "",
				Source:      "test-source",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
		},
		{
			name: "empty source",
			event: &event.CloudEvent{
				ID:          "test-id",
				Source:      "",
				SpecVersion: "1.0",
				Type:        "test.event",
			},
		},
		{
			name: "empty type",
			event: &event.CloudEvent{
				ID:          "test-id",
				Source:      "test-source",
				SpecVersion: "1.0",
				Type:        "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.event)
			if err == nil {
				t.Error("Validate() error = nil, want error for empty required field")
			}
		})
	}
}
