// Package kafka implements Kafka DLQ (Dead Letter Queue) publishing.
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/jittakal/kafeventstore/internal/errors"
	"github.com/jittakal/kafeventstore/pkg/consumer"
	"github.com/jittakal/kafeventstore/pkg/event"
)

// Ensure implementation satisfies interface at compile time.
var _ consumer.DLQPublisher = (*DLQPublisher)(nil)

// DLQEvent represents an event published to the dead letter queue.
type DLQEvent struct {
	OriginalEvent     json.RawMessage `json:"original_event"`
	OriginalTopic     string          `json:"original_topic"`
	OriginalPartition int32           `json:"original_partition"`
	OriginalOffset    int64           `json:"original_offset"`
	FailureReason     string          `json:"failure_reason"`
	FailureTimestamp  time.Time       `json:"failure_timestamp"`
	RetryCount        int             `json:"retry_count"`
	ProcessorID       string          `json:"processor_id"`
}

// DLQConfig contains DLQ configuration.
type DLQConfig struct {
	Enabled     bool
	TopicSuffix string
	MaxRetries  int
}

// DLQPublisher publishes failed events to a dead letter queue.
type DLQPublisher struct {
	producer    sarama.SyncProducer
	config      DLQConfig
	logger      *slog.Logger
	mu          sync.RWMutex
	closed      bool
	processorID string
}

// NewDLQPublisher creates a new DLQ publisher.
func NewDLQPublisher(
	bootstrapServers []string,
	securityConfig ConsumerConfig,
	dlqConfig DLQConfig,
	logger *slog.Logger,
	processorID string,
) (*DLQPublisher, error) {
	if !dlqConfig.Enabled {
		logger.Info("DLQ is disabled")
		return &DLQPublisher{
			config:      dlqConfig,
			logger:      logger,
			processorID: processorID,
			closed:      true,
		}, nil
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_0_0
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = 5
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.Compression = sarama.CompressionSnappy
	saramaConfig.Producer.Idempotent = true
	saramaConfig.Net.MaxOpenRequests = 1

	// Security configuration (reuse consumer security)
	if err := configureSecurity(saramaConfig, securityConfig); err != nil {
		return nil, fmt.Errorf("failed to configure security: %w", err)
	}

	producer, err := sarama.NewSyncProducer(bootstrapServers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync producer: %w", err)
	}

	logger.Info("DLQ publisher created",
		"bootstrap_servers", bootstrapServers,
		"topic_suffix", dlqConfig.TopicSuffix,
	)

	return &DLQPublisher{
		producer:    producer,
		config:      dlqConfig,
		logger:      logger,
		processorID: processorID,
		closed:      false,
	}, nil
}

// Publish publishes a failed event to the DLQ.
func (p *DLQPublisher) Publish(
	ctx context.Context,
	cloudEvent *event.CloudEvent,
	metadata event.KafkaMetadata,
	reason string,
) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return errors.ErrConsumerClosed
	}

	if !p.config.Enabled {
		p.logger.Debug("DLQ disabled, skipping publish")
		return nil
	}

	// Construct DLQ topic name
	dlqTopic := metadata.Topic + p.config.TopicSuffix

	// Marshal original event
	eventData, err := json.Marshal(cloudEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create DLQ event
	dlqEvent := DLQEvent{
		OriginalEvent:     eventData,
		OriginalTopic:     metadata.Topic,
		OriginalPartition: metadata.Partition,
		OriginalOffset:    metadata.Offset,
		FailureReason:     reason,
		FailureTimestamp:  time.Now().UTC(),
		RetryCount:        0,
		ProcessorID:       p.processorID,
	}

	// Marshal DLQ event
	dlqData, err := json.Marshal(dlqEvent)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ event: %w", err)
	}

	// Create Kafka message
	msg := &sarama.ProducerMessage{
		Topic: dlqTopic,
		Key:   sarama.StringEncoder(cloudEvent.ID),
		Value: sarama.ByteEncoder(dlqData),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("failure_reason"),
				Value: []byte(reason),
			},
			{
				Key:   []byte("original_topic"),
				Value: []byte(metadata.Topic),
			},
			{
				Key:   []byte("processor_id"),
				Value: []byte(p.processorID),
			},
		},
		Timestamp: time.Now(),
	}

	// Send message
	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Error("failed to publish to DLQ",
			"error", err,
			"dlq_topic", dlqTopic,
			"event_id", cloudEvent.ID,
		)
		return fmt.Errorf("failed to send message to DLQ: %w", err)
	}

	p.logger.Info("published event to DLQ",
		"dlq_topic", dlqTopic,
		"partition", partition,
		"offset", offset,
		"event_id", cloudEvent.ID,
		"reason", reason,
	)

	return nil
}

// Close closes the DLQ publisher.
func (p *DLQPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	p.logger.Info("closing DLQ publisher")

	if p.producer != nil {
		if err := p.producer.Close(); err != nil {
			p.logger.Error("error closing producer", "error", err)
			return err
		}
	}

	p.logger.Info("DLQ publisher closed")
	return nil
}
