// Package kafka implements Kafka consumer and producer functionality.
package kafka

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	"github.com/jittakal/kafeventstore/internal/errors"
	"github.com/jittakal/kafeventstore/pkg/consumer"
	"github.com/jittakal/kafeventstore/pkg/event"
)

// Ensure implementation satisfies interfaces at compile time.
var (
	_ consumer.Consumer = (*SaramaConsumer)(nil)
)

// ConsumerConfig contains Kafka consumer configuration.
type ConsumerConfig struct {
	BootstrapServers    []string
	GroupID             string
	SecurityProtocol    string
	SASLMechanism       string
	SASLUsername        string
	SASLPassword        string
	AutoOffsetReset     string
	EnableAutoCommit    bool
	MaxPollIntervalMS   int
	SessionTimeoutMS    int
	HeartbeatIntervalMS int
}

// MetricsCollector defines metrics operations for Kafka consumer.
type MetricsCollector interface {
	IncMessagesConsumed(topic string, partition int32)
	IncRebalances(groupID string)
	IncOffsetCommits(topic string, partition int32, status string)
	ObserveRebalanceDuration(groupID string, duration float64)
	ObserveCommitLatency(topic string, partition int32, duration float64)
	SetPartitionsAssigned(topic string, count float64)
}

// SaramaConsumer implements the consumer.Consumer interface using the Sarama library.
// It provides a production-ready Kafka consumer with support for consumer groups,
// offset management, and various security protocols including AWS MSK IAM.
type SaramaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	config        ConsumerConfig
	logger        *slog.Logger
	metrics       MetricsCollector
	topics        []string
	ready         chan bool
	mu            sync.RWMutex
	closed        bool
}

// NewSaramaConsumer creates a new Kafka consumer using Sarama library.
func NewSaramaConsumer(
	config ConsumerConfig,
	logger *slog.Logger,
	metrics MetricsCollector,
) (*SaramaConsumer, error) {
	saramaConfig := sarama.NewConfig()

	// Consumer configuration following AWS MSK best practices
	saramaConfig.Version = sarama.V2_8_0_0
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
	saramaConfig.Consumer.Offsets.Initial = offsetInitial(config.AutoOffsetReset)
	saramaConfig.Consumer.Offsets.AutoCommit.Enable = config.EnableAutoCommit

	// AWS MSK Best Practices: Timeout settings
	// session_timeout_ms should be between 6000-300000 (6s-5min), recommended 10000 (10s)
	saramaConfig.Consumer.Group.Session.Timeout = time.Duration(config.SessionTimeoutMS) * time.Millisecond
	saramaConfig.Consumer.Group.Heartbeat.Interval = time.Duration(config.HeartbeatIntervalMS) * time.Millisecond

	// max_poll_interval_ms prevents rebalancing during long processing
	if config.MaxPollIntervalMS > 0 {
		saramaConfig.Consumer.MaxProcessingTime = time.Duration(config.MaxPollIntervalMS) * time.Millisecond
	} else {
		// Default to 5 minutes if not specified
		saramaConfig.Consumer.MaxProcessingTime = 5 * time.Minute
	}

	saramaConfig.Consumer.Return.Errors = true

	// Security configuration
	if err := configureSecurity(saramaConfig, config); err != nil {
		return nil, fmt.Errorf("failed to configure security: %w", err)
	}

	// Create consumer group
	consumerGroup, err := sarama.NewConsumerGroup(
		config.BootstrapServers,
		config.GroupID,
		saramaConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	logger.Info("kafka consumer created",
		"group_id", config.GroupID,
		"bootstrap_servers", config.BootstrapServers,
		"session_timeout_ms", config.SessionTimeoutMS,
		"max_poll_interval_ms", config.MaxPollIntervalMS,
	)

	return &SaramaConsumer{
		consumerGroup: consumerGroup,
		config:        config,
		logger:        logger,
		metrics:       metrics,
		ready:         make(chan bool),
		closed:        false,
	}, nil
}

// Subscribe subscribes to the specified topics.
func (c *SaramaConsumer) Subscribe(ctx context.Context, topics []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.ErrConsumerClosed
	}

	c.topics = topics
	c.logger.Info("subscribed to topics", "topics", topics)
	return nil
}

// Consume starts consuming messages and returns channels for events and errors.
func (c *SaramaConsumer) Consume(ctx context.Context) (<-chan *event.ConsumedEvent, <-chan error, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, nil, errors.ErrConsumerClosed
	}
	c.mu.RUnlock()

	eventChan := make(chan *event.ConsumedEvent, 100)
	errorChan := make(chan error, 10)

	handler := &consumerGroupHandler{
		consumer:  c,
		eventChan: eventChan,
		errorChan: errorChan,
		ready:     c.ready,
	}

	// Start consuming in background
	go func() {
		defer close(eventChan)
		defer close(errorChan)

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("consumer context cancelled")
				return
			default:
				if err := c.consumerGroup.Consume(ctx, c.topics, handler); err != nil {
					c.logger.Error("consumer group error", "error", err)
					errorChan <- err
					return
				}

				// Check if context was cancelled
				if ctx.Err() != nil {
					return
				}
			}
		}
	}()

	// Wait for consumer to be ready
	<-c.ready

	c.logger.Info("kafka consumer started and ready")
	return eventChan, errorChan, nil
}

// Commit commits the offset for a specific partition.
func (c *SaramaConsumer) Commit(ctx context.Context, partition event.PartitionID, offset int64) error {
	startTime := time.Now()

	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return errors.ErrConsumerClosed
	}

	// Note: In Sarama, offset commits are handled within the ConsumerGroupHandler
	// via session.MarkMessage or session.MarkOffset
	// This method is kept for interface compatibility but actual commits happen in ConsumeClaim

	c.logger.Debug("commit requested",
		"topic", partition.Topic,
		"partition", partition.Partition,
		"offset", offset,
	)

	if c.metrics != nil {
		// Track commit latency
		c.metrics.ObserveCommitLatency(
			partition.Topic,
			partition.Partition,
			time.Since(startTime).Seconds(),
		)

		// Track commit success
		c.metrics.IncOffsetCommits(partition.Topic, partition.Partition, "success")
	}

	return nil
}

// Close closes the consumer and releases resources.
func (c *SaramaConsumer) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.logger.Info("closing kafka consumer")

	if err := c.consumerGroup.Close(); err != nil {
		c.logger.Error("error closing consumer group", "error", err)
		return err
	}

	c.logger.Info("kafka consumer closed")
	return nil
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler.
type consumerGroupHandler struct {
	consumer       *SaramaConsumer
	eventChan      chan<- *event.ConsumedEvent
	errorChan      chan<- error
	ready          chan bool
	readyOnce      sync.Once
	rebalanceStart time.Time
}

// Setup is run at the beginning of a new session, before ConsumeClaim.
func (h *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	// Track rebalance start time
	h.rebalanceStart = time.Now()

	h.consumer.logger.Info("consumer group session setup",
		"member_id", session.MemberID(),
		"generation_id", session.GenerationID(),
		"claims", session.Claims(),
	)

	if h.consumer.metrics != nil {
		// Increment rebalance counter
		h.consumer.metrics.IncRebalances(h.consumer.config.GroupID)

		// Track partition assignments per topic
		topicPartitions := make(map[string]int)
		for topic, partitions := range session.Claims() {
			topicPartitions[topic] = len(partitions)
		}

		for topic, count := range topicPartitions {
			h.consumer.metrics.SetPartitionsAssigned(topic, float64(count))
		}
	}

	// Signal that consumer is ready (only close once)
	h.readyOnce.Do(func() {
		close(h.ready)
	})
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited.
func (h *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	// Record rebalance duration
	if h.consumer.metrics != nil && !h.rebalanceStart.IsZero() {
		rebalanceDuration := time.Since(h.rebalanceStart).Seconds()
		h.consumer.metrics.ObserveRebalanceDuration(
			h.consumer.config.GroupID,
			rebalanceDuration,
		)
	}

	h.consumer.logger.Info("consumer group session cleanup",
		"member_id", session.MemberID(),
	)
	return nil
}

// ConsumeClaim processes messages from a partition.
func (h *consumerGroupHandler) ConsumeClaim(
	session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim,
) error {
	topic := claim.Topic()
	partition := claim.Partition()

	h.consumer.logger.Info("started consuming partition",
		"topic", topic,
		"partition", partition,
		"initial_offset", claim.InitialOffset(),
	)

	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			// Log raw message for debugging
			h.consumer.logger.Debug("received kafka message",
				"topic", message.Topic,
				"partition", message.Partition,
				"offset", message.Offset,
				"timestamp", message.Timestamp,
				"value_size", len(message.Value),
			)

			// Parse CloudEvent from message
			cloudEvent, err := h.parseCloudEvent(message)
			if err != nil {
				h.consumer.logger.Error("failed to parse cloud event",
					"error", err,
					"topic", message.Topic,
					"partition", message.Partition,
					"offset", message.Offset,
				)
				h.errorChan <- fmt.Errorf("failed to parse cloud event: %w", err)
				continue
			}

			// Log parsed event details
			h.consumer.logger.Debug("parsed cloud event",
				"event_id", cloudEvent.ID,
				"source", cloudEvent.Source,
				"type", cloudEvent.Type,
				"specversion", cloudEvent.SpecVersion,
				"data_size", len(cloudEvent.Data),
				"subject", cloudEvent.Subject,
				"data_content_type", cloudEvent.DataContentType,
				"time", cloudEvent.Time,
			)

			// Create consumed event
			consumedEvent := &event.ConsumedEvent{
				Event: cloudEvent,
				Metadata: event.KafkaMetadata{
					Topic:     message.Topic,
					Partition: message.Partition,
					Offset:    message.Offset,
					Timestamp: message.Timestamp,
					Headers:   h.extractHeaders(message.Headers),
				},
				CommitFunc: func() error {
					session.MarkMessage(message, "")
					return nil
				},
			}

			// Send to event channel
			select {
			case h.eventChan <- consumedEvent:
				if h.consumer.metrics != nil {
					h.consumer.metrics.IncMessagesConsumed(message.Topic, message.Partition)
				}
			case <-session.Context().Done():
				return nil
			}

		case <-session.Context().Done():
			h.consumer.logger.Info("session context done, stopping partition consumption",
				"topic", topic,
				"partition", partition,
			)
			return nil
		}
	}
}

// parseCloudEvent parses a Kafka message into a CloudEvent.
// Automatically normalizes CloudEvents 0.1 to 1.0 for backward compatibility.
func (h *consumerGroupHandler) parseCloudEvent(message *sarama.ConsumerMessage) (*event.CloudEvent, error) {
	var cloudEvent event.CloudEvent

	if err := json.Unmarshal(message.Value, &cloudEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cloud event: %w", err)
	}

	// Normalize legacy CloudEvents 0.1 to 1.0 for consistent processing
	if cloudEvent.SpecVersion == "0.1" {
		cloudEvent.SpecVersion = "1.0"
	}

	return &cloudEvent, nil
}

// extractHeaders extracts headers from Kafka message.
func (h *consumerGroupHandler) extractHeaders(headers []*sarama.RecordHeader) map[string]string {
	result := make(map[string]string)
	for _, header := range headers {
		result[string(header.Key)] = string(header.Value)
	}
	return result
}

// MSKAccessTokenProvider implements sarama.AccessTokenProvider for AWS MSK IAM authentication.
type MSKAccessTokenProvider struct {
	region string
}

// Token generates an AWS MSK IAM authentication token.
func (m *MSKAccessTokenProvider) Token() (*sarama.AccessToken, error) {
	// Generate auth token using AWS credentials from environment/profile
	token, expiryMs, err := signer.GenerateAuthToken(context.Background(), m.region)
	if err != nil {
		return nil, fmt.Errorf("failed to generate MSK IAM token: %w", err)
	}

	return &sarama.AccessToken{
		Token: token,
		Extensions: map[string]string{
			"expiry": fmt.Sprintf("%d", expiryMs),
		},
	}, nil
}

// Helper functions

// offsetInitial converts the AutoOffsetReset config to Sarama's offset constant.
func offsetInitial(autoOffsetReset string) int64 {
	switch autoOffsetReset {
	case "earliest":
		return sarama.OffsetOldest
	case "latest":
		return sarama.OffsetNewest
	default:
		return sarama.OffsetNewest
	}
}

func configureSecurity(config *sarama.Config, kafkaConfig ConsumerConfig) error {
	switch kafkaConfig.SecurityProtocol {
	case "PLAINTEXT":
		// No security configuration needed
		return nil

	case "SASL_PLAINTEXT", "SASL_SSL":
		config.Net.SASL.Enable = true

		switch kafkaConfig.SASLMechanism {
		case "PLAIN":
			config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
			config.Net.SASL.User = kafkaConfig.SASLUsername
			config.Net.SASL.Password = kafkaConfig.SASLPassword

		case "SCRAM-SHA-256":
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
			config.Net.SASL.User = kafkaConfig.SASLUsername
			config.Net.SASL.Password = kafkaConfig.SASLPassword
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA256()}
			}

		case "SCRAM-SHA-512":
			config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
			config.Net.SASL.User = kafkaConfig.SASLUsername
			config.Net.SASL.Password = kafkaConfig.SASLPassword
			config.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &XDGSCRAMClient{HashGeneratorFcn: SHA512()}
			}

		case "AWS_MSK_IAM":
			// AWS MSK IAM authentication
			config.Net.SASL.Mechanism = sarama.SASLTypeOAuth
			config.Net.SASL.Enable = true

			// OAuth doesn't use username/password, but Sarama requires them to be set
			// Set to dummy values to pass validation
			config.Net.SASL.User = "token"
			config.Net.SASL.Password = "token"

			// Create IAM token provider using AWS CLI credentials
			config.Net.SASL.TokenProvider = &MSKAccessTokenProvider{
				region: "us-east-1", // Extract from broker address if needed
			}

		default:
			return fmt.Errorf("unsupported SASL mechanism: %s", kafkaConfig.SASLMechanism)
		}

		if kafkaConfig.SecurityProtocol == "SASL_SSL" {
			config.Net.TLS.Enable = true
			config.Net.TLS.Config = &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: true, // For local development with self-signed certs
			}
		}

	case "SSL":
		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, // For local development with self-signed certs
		}

	default:
		return fmt.Errorf("unsupported security protocol: %s", kafkaConfig.SecurityProtocol)
	}

	return nil
}
