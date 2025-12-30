package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Kafka     KafkaConfig     `mapstructure:"kafka"`
	Topics    TopicsConfig    `mapstructure:"topics"`
	Generator GeneratorConfig `mapstructure:"generator"`
}

// KafkaConfig represents Kafka connection configuration
type KafkaConfig struct {
	Brokers          []string       `mapstructure:"brokers"`
	SecurityProtocol string         `mapstructure:"securityProtocol"` // PLAINTEXT, SASL_SSL, SASL_PLAINTEXT
	SASLMechanism    string         `mapstructure:"saslMechanism"`    // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, AWS_MSK_IAM
	SASLUsername     string         `mapstructure:"saslUsername"`
	SASLPassword     string         `mapstructure:"saslPassword"`
	TLS              TLSConfig      `mapstructure:"tls"`
	Producer         ProducerConfig `mapstructure:"producer"`
	AWSMSK           AWSMSKConfig   `mapstructure:"awsMsk"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	CACertFile         string `mapstructure:"caCertFile"`
	ClientCertFile     string `mapstructure:"clientCertFile"`
	ClientKeyFile      string `mapstructure:"clientKeyFile"`
	InsecureSkipVerify bool   `mapstructure:"insecureSkipVerify"`
}

// ProducerConfig represents Kafka producer configuration
type ProducerConfig struct {
	RequiredAcks     int    `mapstructure:"requiredAcks"`    // 0=NoResponse, 1=WaitForLocal, -1=WaitForAll
	CompressionType  string `mapstructure:"compressionType"` // none, gzip, snappy, lz4, zstd
	MaxMessageBytes  int    `mapstructure:"maxMessageBytes"`
	IdempotentWrites bool   `mapstructure:"idempotentWrites"`
	RetryMax         int    `mapstructure:"retryMax"`
	RetryBackoffMs   int    `mapstructure:"retryBackoffMs"`
}

// AWSMSKConfig represents AWS MSK specific configuration
type AWSMSKConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Region  string `mapstructure:"region"`
}

// TopicsConfig represents topic configuration
type TopicsConfig struct {
	BookIssued   string `mapstructure:"bookIssued"`
	BookReturned string `mapstructure:"bookReturned"`
}

// GeneratorConfig represents event generator configuration
type GeneratorConfig struct {
	IntervalMs   int             `mapstructure:"intervalMs"` // Interval between event generations in milliseconds
	BookIssued   EventTypeConfig `mapstructure:"bookIssued"`
	BookReturned EventTypeConfig `mapstructure:"bookReturned"`
}

// EventTypeConfig represents configuration for a specific event type
type EventTypeConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	Probability float64 `mapstructure:"probability"` // 0.0 to 1.0 (for conditional generation)
}

// Load loads configuration from a file
func Load(configFile string) (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")

	// Read environment variables
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal config
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Override with environment variables if set
	overrideFromEnv(&config)

	return &config, nil
}

// validate validates the configuration
func validate(config *Config) error {
	if len(config.Kafka.Brokers) == 0 {
		return fmt.Errorf("at least one Kafka broker must be configured")
	}

	if config.Topics.BookIssued == "" {
		return fmt.Errorf("bookIssued topic must be configured")
	}

	if config.Topics.BookReturned == "" {
		return fmt.Errorf("bookReturned topic must be configured")
	}

	if config.Generator.IntervalMs <= 0 {
		return fmt.Errorf("generator intervalMs must be greater than 0")
	}

	return nil
}

// overrideFromEnv overrides configuration with environment variables
func overrideFromEnv(config *Config) {
	// Kafka overrides
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		// Simple comma-separated parsing (could be enhanced)
		config.Kafka.Brokers = []string{brokers}
	}

	if username := os.Getenv("KAFKA_SASL_USERNAME"); username != "" {
		config.Kafka.SASLUsername = username
	}

	if password := os.Getenv("KAFKA_SASL_PASSWORD"); password != "" {
		config.Kafka.SASLPassword = password
	}

	// Topic overrides
	if topic := os.Getenv("TOPIC_BOOK_ISSUED"); topic != "" {
		config.Topics.BookIssued = topic
	}

	if topic := os.Getenv("TOPIC_BOOK_RETURNED"); topic != "" {
		config.Topics.BookReturned = topic
	}
}
