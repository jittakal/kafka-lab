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
	"time"

	"github.com/IBM/sarama"
	"github.com/aws/aws-msk-iam-sasl-signer-go/signer"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/jittakal/kafeventproducer/internal/config"
	"github.com/xdg-go/scram"
	"go.uber.org/zap"
)

// Producer wraps Sarama Kafka producer with CloudEvents support
type Producer struct {
	producer sarama.SyncProducer
	logger   *zap.Logger
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg config.KafkaConfig, logger *zap.Logger) (*Producer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true

	// Producer settings
	saramaConfig.Producer.RequiredAcks = sarama.RequiredAcks(cfg.Producer.RequiredAcks)
	saramaConfig.Producer.Compression = parseCompressionType(cfg.Producer.CompressionType)
	saramaConfig.Producer.MaxMessageBytes = cfg.Producer.MaxMessageBytes
	saramaConfig.Producer.Idempotent = cfg.Producer.IdempotentWrites
	saramaConfig.Producer.Retry.Max = cfg.Producer.RetryMax
	saramaConfig.Producer.Retry.Backoff = time.Duration(cfg.Producer.RetryBackoffMs) * time.Millisecond

	// Idempotent producer requires Net.MaxOpenRequests to be 1
	if cfg.Producer.IdempotentWrites {
		saramaConfig.Net.MaxOpenRequests = 1
	}

	// Configure security
	if err := configureSecurity(saramaConfig, cfg, logger); err != nil {
		return nil, fmt.Errorf("failed to configure security: %w", err)
	}

	// Create producer
	producer, err := sarama.NewSyncProducer(cfg.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	logger.Info("Kafka producer created successfully",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("securityProtocol", cfg.SecurityProtocol),
	)

	return &Producer{
		producer: producer,
		logger:   logger,
	}, nil
}

// ProduceEvent produces a CloudEvent to the specified topic
func (p *Producer) ProduceEvent(ctx context.Context, topic string, event cloudevents.Event) error {
	// Serialize CloudEvent to JSON
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal CloudEvent: %w", err)
	}

	// Create Kafka message
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Key:   sarama.StringEncoder(event.ID()),
		Value: sarama.ByteEncoder(eventBytes),
		Headers: []sarama.RecordHeader{
			{
				Key:   []byte("ce_specversion"),
				Value: []byte(event.SpecVersion()),
			},
			{
				Key:   []byte("ce_type"),
				Value: []byte(event.Type()),
			},
			{
				Key:   []byte("ce_source"),
				Value: []byte(event.Source()),
			},
			{
				Key:   []byte("ce_id"),
				Value: []byte(event.ID()),
			},
		},
	}

	// Send message
	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message to Kafka: %w", err)
	}

	p.logger.Info("Event produced successfully",
		zap.String("topic", topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
		zap.String("eventId", event.ID()),
		zap.String("eventType", event.Type()),
		zap.String("source", event.Source()),
		zap.Time("timestamp", event.Time()),
	)

	return nil
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}

// configureSecurity configures SASL and TLS settings
func configureSecurity(saramaConfig *sarama.Config, cfg config.KafkaConfig, logger *zap.Logger) error {
	switch cfg.SecurityProtocol {
	case "PLAINTEXT":
		// No security
		logger.Info("Using PLAINTEXT security protocol")

	case "SASL_SSL":
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.TLS.Enable = true

		// Configure SASL mechanism
		if err := configureSASL(saramaConfig, cfg, logger); err != nil {
			return err
		}

		// Configure TLS
		if err := configureTLS(saramaConfig, cfg, logger); err != nil {
			return err
		}

	case "SASL_PLAINTEXT":
		saramaConfig.Net.SASL.Enable = true

		// Configure SASL mechanism
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

	// Load CA certificate if provided
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

	// Load client certificate if provided
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

// parseCompressionType parses compression type string
func parseCompressionType(compressionType string) sarama.CompressionCodec {
	switch compressionType {
	case "gzip":
		return sarama.CompressionGZIP
	case "snappy":
		return sarama.CompressionSnappy
	case "lz4":
		return sarama.CompressionLZ4
	case "zstd":
		return sarama.CompressionZSTD
	default:
		return sarama.CompressionNone
	}
}

// XDGSCRAMClient implements SCRAM client using xdg-go/scram
type XDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

// SHA256 hash generator
var SHA256 scram.HashGeneratorFcn = func() hash.Hash { return sha256.New() }

// SHA512 hash generator
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

// newMskTokenProvider creates a token provider for AWS MSK IAM authentication
func newMskTokenProvider(region string) sarama.AccessTokenProvider {
	return &MSKAccessTokenProvider{
		region: region,
	}
}

// MSKAccessTokenProvider implements AWS MSK IAM token provider
type MSKAccessTokenProvider struct {
	region string
}

func (m *MSKAccessTokenProvider) Token() (*sarama.AccessToken, error) {
	token, _, err := signer.GenerateAuthToken(context.Background(), m.region)
	if err != nil {
		return nil, err
	}

	return &sarama.AccessToken{
		Token: token,
	}, nil
}
