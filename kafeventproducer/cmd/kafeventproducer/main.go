package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jittakal/kafeventproducer/internal/config"
	"github.com/jittakal/kafeventproducer/internal/generator"
	"github.com/jittakal/kafeventproducer/internal/kafka"
	"github.com/jittakal/kafeventproducer/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
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

	logger.Info("Starting kafeventproducer",
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
	)

	// Initialize metrics
	metricsCollector := metrics.NewCollector()

	// Start metrics server
	go startMetricsServer(*metricsPort, logger)

	// Create Kafka producer
	producer, err := kafka.NewProducer(cfg.Kafka, logger)
	if err != nil {
		logger.Fatal("Failed to create Kafka producer", zap.Error(err))
	}
	defer producer.Close()

	logger.Info("Kafka producer initialized successfully")

	// Create event generator
	eventGen := generator.NewGenerator(cfg.Generator, logger)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start producing events
	go produceEvents(ctx, producer, eventGen, metricsCollector, cfg, logger)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	// Graceful shutdown
	cancel()
	time.Sleep(2 * time.Second) // Allow in-flight messages to complete

	logger.Info("Shutdown complete")
}

// produceEvents continuously generates and produces events
func produceEvents(
	ctx context.Context,
	producer *kafka.Producer,
	eventGen *generator.Generator,
	metricsCollector *metrics.Collector,
	cfg *config.Config,
	logger *zap.Logger,
) {
	ticker := time.NewTicker(time.Duration(cfg.Generator.IntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping event production")
			return
		case <-ticker.C:
			// Generate and produce book issued event
			if cfg.Generator.BookIssued.Enabled {
				bookIssuedEvent := eventGen.GenerateBookIssuedEvent()
				if err := producer.ProduceEvent(ctx, cfg.Topics.BookIssued, bookIssuedEvent); err != nil {
					logger.Error("Failed to produce book issued event",
						zap.Error(err),
						zap.String("eventId", bookIssuedEvent.ID()),
					)
					metricsCollector.IncEventsFailedTotal(cfg.Topics.BookIssued, bookIssuedEvent.Type())
				} else {
					logger.Debug("Produced book issued event",
						zap.String("eventId", bookIssuedEvent.ID()),
						zap.String("topic", cfg.Topics.BookIssued),
					)
					metricsCollector.IncEventsProducedTotal(cfg.Topics.BookIssued, bookIssuedEvent.Type())
				}
			}

			// Generate and produce book returned event (less frequently)
			if cfg.Generator.BookReturned.Enabled && shouldGenerateReturnEvent(cfg.Generator.BookReturned.Probability) {
				bookReturnedEvent := eventGen.GenerateBookReturnedEvent()
				if err := producer.ProduceEvent(ctx, cfg.Topics.BookReturned, bookReturnedEvent); err != nil {
					logger.Error("Failed to produce book returned event",
						zap.Error(err),
						zap.String("eventId", bookReturnedEvent.ID()),
					)
					metricsCollector.IncEventsFailedTotal(cfg.Topics.BookReturned, bookReturnedEvent.Type())
				} else {
					logger.Debug("Produced book returned event",
						zap.String("eventId", bookReturnedEvent.ID()),
						zap.String("topic", cfg.Topics.BookReturned),
					)
					metricsCollector.IncEventsProducedTotal(cfg.Topics.BookReturned, bookReturnedEvent.Type())
				}
			}
		}
	}
}

// shouldGenerateReturnEvent determines if a return event should be generated based on probability
func shouldGenerateReturnEvent(probability float64) bool {
	// Simple probability check (0.0 to 1.0)
	// For example, 0.3 means 30% chance
	return time.Now().UnixNano()%100 < int64(probability*100)
}

// startMetricsServer starts the Prometheus metrics HTTP server
func startMetricsServer(port string, logger *zap.Logger) {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := ":" + port
	logger.Info("Starting metrics server", zap.String("address", addr))

	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Fatal("Metrics server failed", zap.Error(err))
	}
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
		return zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		return zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		return zap.NewAtomicLevelAt(zap.InfoLevel)
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
