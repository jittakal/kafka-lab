package kafka

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"hash"
	"os"
	"sync"

	"github.com/IBM/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/jittakal/kafeventconsumer/internal/config"
	"github.com/xdg-go/scram"
	"go.uber.org/zap"
)

// Consumer wraps Sarama Kafka consumer with CloudEvents support
type Consumer struct {
	client  sarama.ConsumerGroup
	topics  []string
	logger  *zap.Logger
	handler *consumerGroupHandler
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *config.Config, topics []string, logger *zap.Logger) (*Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V3_0_0_0
	saramaConfig.Consumer.Return.Errors = true

	// Consumer settings
	if cfg.Consumer.OffsetInitial == "oldest" {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	saramaConfig.Consumer.Group.Session.Timeout = 10000 * 1000000   // 10s in nanoseconds
	saramaConfig.Consumer.Group.Heartbeat.Interval = 3000 * 1000000 // 3s in nanoseconds
	saramaConfig.Consumer.MaxProcessingTime = 60000 * 1000000       // 60s in nanoseconds

	if cfg.Consumer.AutoCommit {
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
	}

	// Configure security
	if err := configureSecurity(saramaConfig, cfg.Kafka, logger); err != nil {
		return nil, fmt.Errorf("failed to configure security: %w", err)
	}

	// Create consumer group
	client, err := sarama.NewConsumerGroup(cfg.Kafka.Brokers, cfg.Kafka.ConsumerGroup, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	logger.Info("Kafka consumer created successfully",
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.String("consumerGroup", cfg.Kafka.ConsumerGroup),
		zap.Strings("topics", topics),
	)

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		client: client,
		topics: topics,
		logger: logger,
		handler: &consumerGroupHandler{
			logger: logger,
		},
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start starts consuming messages
func (c *Consumer) Start() error {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			// Consume should be called inside an infinite loop
			// When a server-side rebalance happens, the consumer session will need to be recreated
			if err := c.client.Consume(c.ctx, c.topics, c.handler); err != nil {
				c.logger.Error("Error from consumer", zap.Error(err))
			}

			// Check if context was cancelled, signaling that the consumer should stop
			if c.ctx.Err() != nil {
				return
			}
		}
	}()

	c.logger.Info("Kafka consumer started",
		zap.Strings("topics", c.topics),
	)

	return nil
}

// Stop stops the consumer gracefully
func (c *Consumer) Stop() error {
	c.logger.Info("Stopping Kafka consumer...")
	c.cancel()
	c.wg.Wait()

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close consumer: %w", err)
	}

	c.logger.Info("Kafka consumer stopped successfully")
	return nil
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	logger *zap.Logger
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Consumer group session started")
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	h.logger.Info("Consumer group session ended")
	return nil
}

// ConsumeClaim processes messages from a partition
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			h.processMessage(message)
			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}

// processMessage processes a single Kafka message
func (h *consumerGroupHandler) processMessage(msg *sarama.ConsumerMessage) {
	// Deserialize CloudEvent
	var event cloudevents.Event
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		h.logger.Error("Failed to unmarshal CloudEvent",
			zap.Error(err),
			zap.String("topic", msg.Topic),
			zap.Int32("partition", msg.Partition),
			zap.Int64("offset", msg.Offset),
		)
		return
	}

	// Log the consumed event with key details
	h.logger.Info("Event consumed",
		zap.String("topic", msg.Topic),
		zap.Int32("partition", msg.Partition),
		zap.Int64("offset", msg.Offset),
		zap.String("key", string(msg.Key)),
		zap.String("eventId", event.ID()),
		zap.String("eventType", event.Type()),
		zap.String("eventSource", event.Source()),
		zap.Time("eventTime", event.Time()),
		zap.String("specVersion", event.SpecVersion()),
		zap.String("dataContentType", event.DataContentType()),
		zap.ByteString("data", event.Data()),
	)
}

// configureSecurity configures SASL and TLS settings
func configureSecurity(saramaConfig *sarama.Config, cfg config.KafkaConfig, logger *zap.Logger) error {
	switch cfg.SecurityProtocol {
	case "PLAINTEXT":
		logger.Info("Using PLAINTEXT security protocol")

	case "SASL_SSL":
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.TLS.Enable = true

		if err := configureSASL(saramaConfig, cfg, logger); err != nil {
			return err
		}

		if err := configureTLS(saramaConfig, cfg, logger); err != nil {
			return err
		}

	case "SASL_PLAINTEXT":
		saramaConfig.Net.SASL.Enable = true

		if err := configureSASL(saramaConfig, cfg, logger); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported security protocol: %s", cfg.SecurityProtocol)
	}

	return nil
}

// configureSASL configures SASL authentication
func configureSASL(saramaConfig *sarama.Config, cfg config.KafkaConfig, logger *zap.Logger) error {
	switch cfg.SASLMechanism {
	case "PLAIN":
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		saramaConfig.Net.SASL.User = cfg.SASLUsername
		saramaConfig.Net.SASL.Password = cfg.SASLPassword
		logger.Info("Using SASL PLAIN authentication")

	case "SCRAM-SHA-256":
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
		saramaConfig.Net.SASL.User = cfg.SASLUsername
		saramaConfig.Net.SASL.Password = cfg.SASLPassword
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA256}
		}
		logger.Info("Using SASL SCRAM-SHA-256 authentication")

	case "SCRAM-SHA-512":
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
		saramaConfig.Net.SASL.User = cfg.SASLUsername
		saramaConfig.Net.SASL.Password = cfg.SASLPassword
		saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
			return &XDGSCRAMClient{HashGeneratorFcn: SHA512}
		}
		logger.Info("Using SASL SCRAM-SHA-512 authentication")

	case "AWS_MSK_IAM":
		if !cfg.AWSMSK.Enabled {
			return fmt.Errorf("AWS MSK IAM authentication requires awsMsk.enabled=true")
		}
		saramaConfig.Net.SASL.Mechanism = sarama.SASLTypeOAuth
		saramaConfig.Net.SASL.TokenProvider = newMskTokenProvider(cfg.AWSMSK.Region)
		logger.Info("Using AWS MSK IAM authentication", zap.String("region", cfg.AWSMSK.Region))

	default:
		return fmt.Errorf("unsupported SASL mechanism: %s", cfg.SASLMechanism)
	}

	return nil
}

// configureTLS configures TLS settings
func configureTLS(saramaConfig *sarama.Config, cfg config.KafkaConfig, logger *zap.Logger) error {
	if !cfg.TLS.Enabled {
		logger.Warn("TLS is required for SASL_SSL but not enabled in config")
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.TLS.InsecureSkipVerify,
	}

	if cfg.TLS.CACertFile != "" {
		caCert, err := os.ReadFile(cfg.TLS.CACertFile)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
		logger.Info("Loaded CA certificate", zap.String("file", cfg.TLS.CACertFile))
	}

	if cfg.TLS.ClientCertFile != "" && cfg.TLS.ClientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.TLS.ClientCertFile, cfg.TLS.ClientKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load client certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
		logger.Info("Loaded client certificate",
			zap.String("certFile", cfg.TLS.ClientCertFile),
			zap.String("keyFile", cfg.TLS.ClientKeyFile),
		)
	}

	saramaConfig.Net.TLS.Config = tlsConfig
	return nil
}

// XDGSCRAMClient implements SCRAM client using xdg-go/scram
type XDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

var SHA256 scram.HashGeneratorFcn = func() hash.Hash { return sha256.New() }
var SHA512 scram.HashGeneratorFcn = func() hash.Hash { return sha512.New() }

func (x *XDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

func (x *XDGSCRAMClient) Step(challenge string) (response string, err error) {
	return x.ClientConversation.Step(challenge)
}

func (x *XDGSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}

func newMskTokenProvider(region string) sarama.AccessTokenProvider {
	return &MSKAccessTokenProvider{region: region}
}

type MSKAccessTokenProvider struct {
	region string
}

func (m *MSKAccessTokenProvider) Token() (*sarama.AccessToken, error) {
	token, _, err := signer.GenerateAuthToken(context.Background(), m.region)
	if err != nil {
		return nil, err
	}
	return &sarama.AccessToken{Token: token}, nil
}
