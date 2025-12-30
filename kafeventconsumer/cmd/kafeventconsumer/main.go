package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jittakal/kafeventconsumer/internal/config"
	"github.com/jittakal/kafeventconsumer/internal/kafka"
	"github.com/jittakal/kafeventconsumer/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Version information (set during build)
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"

	// Command-line flags
	configFile  = flag.String("config", getEnv("CONFIG_FILE", "config/local-kafka-kraft.yaml"), "Path to configuration file")
	logLevel    = flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	metricsPort = flag.String("metrics-port", getEnv("METRICS_PORT", "9090"), "Prometheus metrics port")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger, err := initLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Info("Starting kafeventconsumer",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("buildTime", buildTime),
	)

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	logger.Info("Configuration loaded",
		zap.String("configFile", *configFile),
		zap.Strings("brokers", cfg.Kafka.Brokers),
		zap.String("consumerGroup", cfg.Kafka.ConsumerGroup),
	)

	// Initialize metrics
	_ = metrics.NewMetrics()

	// Get topics to consume
	topics := []string{cfg.Topics.BookIssued, cfg.Topics.BookReturned}

	// Create Kafka consumer
	consumer, err := kafka.NewConsumer(cfg, topics, logger)
	if err != nil {
		logger.Fatal("Failed to create Kafka consumer", zap.Error(err))
	}

	logger.Info("Kafka consumer initialized successfully")

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		addr := ":" + *metricsPort
		logger.Info("Starting metrics server", zap.String("address", addr))

		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Error("Metrics server failed", zap.Error(err))
		}
	}()

	// Start consumer
	if err := consumer.Start(); err != nil {
		logger.Fatal("Failed to start consumer", zap.Error(err))
	}

	// Setup signal handling for graceful shutdown
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-sigterm
	logger.Info("Received termination signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	if err := consumer.Stop(); err != nil {
		logger.Error("Error stopping consumer", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("kafeventconsumer stopped gracefully")
}

// initLogger initializes the zap logger based on the log level
func initLogger(level string) (*zap.Logger, error) {
	var config zap.Config

	switch level {
	case "debug":
		config = zap.NewDevelopmentConfig()
	case "info", "warn", "error":
		config = zap.NewProductionConfig()
		config.Level = parseLogLevel(level)
	default:
		config = zap.NewProductionConfig()
	}

	return config.Build()
}

// parseLogLevel parses the log level string
func parseLogLevel(level string) zap.AtomicLevel {
	switch level {
	case "debug":
		return zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		return zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		return zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
