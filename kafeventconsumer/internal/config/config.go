package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	Topics   TopicsConfig   `mapstructure:"topics"`
	Consumer ConsumerConfig `mapstructure:"consumer"`
}

// KafkaConfig represents Kafka connection configuration
type KafkaConfig struct {
	Brokers          []string     `mapstructure:"brokers"`
	SecurityProtocol string       `mapstructure:"securityProtocol"` // PLAINTEXT, SASL_SSL, SASL_PLAINTEXT
	SASLMechanism    string       `mapstructure:"saslMechanism"`    // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, AWS_MSK_IAM
	SASLUsername     string       `mapstructure:"saslUsername"`
	SASLPassword     string       `mapstructure:"saslPassword"`
	TLS              TLSConfig    `mapstructure:"tls"`
	ConsumerGroup    string       `mapstructure:"consumerGroup"`
	AWSMSK           AWSMSKConfig `mapstructure:"awsMsk"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	CACertFile         string `mapstructure:"caCertFile"`
	ClientCertFile     string `mapstructure:"clientCertFile"`
	ClientKeyFile      string `mapstructure:"clientKeyFile"`
	InsecureSkipVerify bool   `mapstructure:"insecureSkipVerify"`
}

// AWSMSKConfig represents AWS MSK specific configuration
type AWSMSKConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Region  string `mapstructure:"region"`
}

// TopicsConfig represents topics to consume from
type TopicsConfig struct {
	BookIssued   string `mapstructure:"bookIssued"`
	BookReturned string `mapstructure:"bookReturned"`
}

// ConsumerConfig represents consumer behavior configuration
type ConsumerConfig struct {
	OffsetInitial       string `mapstructure:"offsetInitial"` // oldest, newest
	SessionTimeoutMs    int    `mapstructure:"sessionTimeoutMs"`
	HeartbeatIntervalMs int    `mapstructure:"heartbeatIntervalMs"`
	MaxProcessingTimeMs int    `mapstructure:"maxProcessingTimeMs"`
	AutoCommit          bool   `mapstructure:"autoCommit"`
}

// Load loads configuration from file
func Load(configFile string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.AutomaticEnv()

	// Bind environment variables for sensitive data
	v.BindEnv("kafka.saslUsername", "KAFKA_SASL_USERNAME")
	v.BindEnv("kafka.saslPassword", "KAFKA_SASL_PASSWORD")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
