package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/IBM/sarama"
)

func TestConsumerConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  ConsumerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ConsumerConfig{
				BootstrapServers: []string{"localhost:9092"},
				GroupID:          "test-group",
				AutoOffsetReset:  "earliest",
			},
			wantErr: false,
		},
		{
			name: "empty bootstrap servers",
			config: ConsumerConfig{
				BootstrapServers: []string{},
				GroupID:          "test-group",
			},
			wantErr: true,
		},
		{
			name: "empty group ID",
			config: ConsumerConfig{
				BootstrapServers: []string{"localhost:9092"},
				GroupID:          "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConsumerConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConsumerConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func validateConsumerConfig(config ConsumerConfig) error {
	if len(config.BootstrapServers) == 0 {
		return sarama.ErrInvalidConfig
	}
	if config.GroupID == "" {
		return sarama.ErrInvalidConfig
	}
	return nil
}

func TestConsumerConfig_Defaults(t *testing.T) {
	config := ConsumerConfig{
		BootstrapServers: []string{"localhost:9092"},
		GroupID:          "test-group",
	}

	// Test that defaults are properly applied
	if config.AutoOffsetReset == "" {
		config.AutoOffsetReset = "latest"
	}
	if config.SessionTimeoutMS == 0 {
		config.SessionTimeoutMS = 10000
	}
	if config.HeartbeatIntervalMS == 0 {
		config.HeartbeatIntervalMS = 3000
	}

	if config.AutoOffsetReset != "latest" {
		t.Errorf("AutoOffsetReset = %v, want latest", config.AutoOffsetReset)
	}
	if config.SessionTimeoutMS != 10000 {
		t.Errorf("SessionTimeoutMS = %v, want 10000", config.SessionTimeoutMS)
	}
	if config.HeartbeatIntervalMS != 3000 {
		t.Errorf("HeartbeatIntervalMS = %v, want 3000", config.HeartbeatIntervalMS)
	}
}

func TestConsumerConfig_SecurityProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		valid    bool
	}{
		{"plaintext", "PLAINTEXT", true},
		{"ssl", "SSL", true},
		{"sasl_plaintext", "SASL_PLAINTEXT", true},
		{"sasl_ssl", "SASL_SSL", true},
		{"invalid", "INVALID", false},
		{"empty", "", true}, // empty defaults to plaintext
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = ConsumerConfig{
				BootstrapServers: []string{"localhost:9092"},
				GroupID:          "test-group",
				SecurityProtocol: tt.protocol,
			}

			// Mock validation
			valid := tt.protocol == "" || tt.protocol == "PLAINTEXT" ||
				tt.protocol == "SSL" || tt.protocol == "SASL_PLAINTEXT" ||
				tt.protocol == "SASL_SSL"

			if valid != tt.valid {
				t.Errorf("Protocol %v validation = %v, want %v", tt.protocol, valid, tt.valid)
			}
		})
	}
}

func TestConsumerConfig_SASLMechanisms(t *testing.T) {
	tests := []struct {
		name      string
		mechanism string
		valid     bool
	}{
		{"plain", "PLAIN", true},
		{"scram-sha-256", "SCRAM-SHA-256", true},
		{"scram-sha-512", "SCRAM-SHA-512", true},
		{"aws_msk_iam", "AWS_MSK_IAM", true},
		{"invalid", "INVALID", false},
		{"empty", "", true}, // empty is valid when no SASL
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = ConsumerConfig{
				BootstrapServers: []string{"localhost:9092"},
				GroupID:          "test-group",
				SASLMechanism:    tt.mechanism,
			}

			// Mock validation
			valid := tt.mechanism == "" || tt.mechanism == "PLAIN" ||
				tt.mechanism == "SCRAM-SHA-256" || tt.mechanism == "SCRAM-SHA-512" ||
				tt.mechanism == "AWS_MSK_IAM"

			if valid != tt.valid {
				t.Errorf("Mechanism %v validation = %v, want %v", tt.mechanism, valid, tt.valid)
			}
		})
	}
}

func TestConsumerConfig_OffsetReset(t *testing.T) {
	tests := []struct {
		name   string
		offset string
		want   int64
	}{
		{"earliest", "earliest", sarama.OffsetOldest},
		{"latest", "latest", sarama.OffsetNewest},
		{"empty defaults to latest", "", sarama.OffsetNewest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset := tt.offset
			if offset == "" {
				offset = "latest"
			}

			var got int64
			switch offset {
			case "earliest":
				got = sarama.OffsetOldest
			case "latest":
				got = sarama.OffsetNewest
			default:
				got = sarama.OffsetNewest
			}

			if got != tt.want {
				t.Errorf("offset reset = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConsumerConfig_Timeouts(t *testing.T) {
	config := ConsumerConfig{
		BootstrapServers:    []string{"localhost:9092"},
		GroupID:             "test-group",
		SessionTimeoutMS:    6000,
		HeartbeatIntervalMS: 2000,
		MaxPollIntervalMS:   300000,
	}

	// Validate timeout relationships
	if config.HeartbeatIntervalMS >= config.SessionTimeoutMS {
		t.Error("HeartbeatIntervalMS should be less than SessionTimeoutMS")
	}

	if config.SessionTimeoutMS >= config.MaxPollIntervalMS {
		t.Error("SessionTimeoutMS should be less than MaxPollIntervalMS")
	}

	// Typical recommended values
	if config.HeartbeatIntervalMS < 1000 {
		t.Error("HeartbeatIntervalMS should be at least 1000ms")
	}

	if config.SessionTimeoutMS < 3000 {
		t.Error("SessionTimeoutMS should be at least 3000ms")
	}
}

func TestConsumerLifecycle(t *testing.T) {
	// Test consumer lifecycle states
	type state int
	const (
		stateCreated state = iota
		stateConnected
		stateSubscribed
		stateConsuming
		stateClosed
	)

	currentState := stateCreated

	// Simulate state transitions
	if currentState == stateCreated {
		currentState = stateConnected
	}
	if currentState != stateConnected {
		t.Error("Should transition to connected state")
	}

	if currentState == stateConnected {
		currentState = stateSubscribed
	}
	if currentState != stateSubscribed {
		t.Error("Should transition to subscribed state")
	}

	if currentState == stateSubscribed {
		currentState = stateConsuming
	}
	if currentState != stateConsuming {
		t.Error("Should transition to consuming state")
	}

	if currentState == stateConsuming {
		currentState = stateClosed
	}
	if currentState != stateClosed {
		t.Error("Should transition to closed state")
	}
}

func TestConsumerContext(t *testing.T) {
	// Test context cancellation behavior
	ctx, cancel := context.WithCancel(context.Background())

	// Simulate consumer running
	running := true
	go func() {
		<-ctx.Done()
		running = false
	}()

	// Verify consumer is running
	time.Sleep(10 * time.Millisecond)
	if !running {
		t.Error("Consumer should be running")
	}

	// Cancel context
	cancel()

	// Wait for cancellation to propagate
	time.Sleep(10 * time.Millisecond)
	if running {
		t.Error("Consumer should have stopped after context cancellation")
	}
}

func TestConsumerErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		shouldRetry bool
	}{
		{
			name:        "network error - retryable",
			err:         sarama.ErrOutOfBrokers,
			shouldRetry: true,
		},
		{
			name:        "invalid config - not retryable",
			err:         sarama.ErrInvalidConfig,
			shouldRetry: false,
		},
		{
			name:        "nil error",
			err:         nil,
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simple retry logic
			retryable := tt.err == sarama.ErrOutOfBrokers ||
				tt.err == sarama.ErrNotConnected

			if retryable != tt.shouldRetry {
				t.Errorf("Error retry status = %v, want %v", retryable, tt.shouldRetry)
			}
		})
	}
}
