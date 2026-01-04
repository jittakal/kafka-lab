# Kafka Event Blob Store

High-performance Kafka event consumer that persists CloudEvents to blob storage (AWS S3/Azure Blob/GCS/filesystem) in Parquet/Avro formats.

## Features

- CloudEvents v1.0 compliant event consumption
- Multiple storage backends (AWS S3, Azure Blob Storage, Google Cloud Storage, filesystem)
- Columnar formats (Parquet, Avro) with compression
- Partition-aware processing with automatic scaling
- At-least-once delivery guarantee
- Exponential backoff retry with circuit breaker
- Full observability (metrics, logging, tracing, health checks)
- Clean architecture with clear layer separation

## Architecture

### High-Level Overview

```
┌─────────────┐      ┌──────────────┐      ┌──────────────┐      ┌──────────┐
│   Kafka     │─────▶│   Consumer   │─────▶│    Buffer    │─────▶│  Encoder │
│   Cluster   │      │   (Sarama)   │      │   Manager    │      │ (Factory)│
└─────────────┘      └──────────────┘      └──────────────┘      └──────────┘
                            │                      │                    │
                            │                      │                    ▼
                            │                      │            ┌──────────────┐
                            │                      │            │   Storage    │
                            │                      │            │   Router     │
                            │                      │            └──────────────┘
                            ▼                      ▼                    │
                     ┌──────────────┐      ┌──────────────┐           ▼
                     │     DLQ      │      │ Observability│    ┌──────────────┐
                     │   Handler    │      │  (Metrics)   │    │ S3 / Azure / │
                     └──────────────┘      └──────────────┘    │  GCS / File  │
                                                                └──────────────┘
```

### Layer Architecture

The project follows clean architecture principles with clear layer separation:

```
┌─────────────────────────────────────────────────────────────┐
│ Presentation Layer (cmd/)                                   │
│  • HTTP Server (health checks, metrics)                     │
│  • CLI entry point                                          │
└─────────────────────────────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ Application Layer (cmd/main.go orchestration)               │
│  • Event consumption workflow                               │
│  • Error handling & retries                                 │
│  • Buffer management                                        │
└─────────────────────────────────────────────────────────────┘
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ Domain Layer (pkg/)                                         │
│  • Core interfaces (Consumer, Storage, Encoder, Buffer)     │
│  • Event types (CloudEvent, Record, PartitionID)            │
│  • No external dependencies                                 │
└─────────────────────────────────────────────────────────────┘
                           ▲
┌─────────────────────────────────────────────────────────────┐
│ Infrastructure Layer (internal/)                            │
│  • Kafka (Sarama consumer, SCRAM auth, DLQ)                 │
│  • Storage (S3, Azure, File writers)                        │
│  • Encoders (Parquet, Avro with compression)                │
│  • Buffer (thread-safe partition buffers)                   │
│  • Observability (logger, metrics)                          │
│  • Config (YAML loader, validation)                         │
└─────────────────────────────────────────────────────────────┘
```

### Package Structure

```
pkg/                      # Public API (interfaces)
├── buffer/              # Buffer interface
├── consumer/            # Consumer interface
├── encoder/             # Encoder interface
├── event/               # Core event types (CloudEvent, Record)
└── storage/             # Storage interface

internal/                # Private implementations
├── buffer/              # Thread-safe partition buffers
├── config/              # Configuration loading & validation
├── encoder/             # Parquet & Avro encoders
├── errors/              # Custom error types
├── kafka/               # Sarama consumer, SCRAM, DLQ
├── observability/       # Logging & metrics
├── server/              # HTTP health server
├── storage/             # S3/Azure/GCS/File writers
└── validator/           # CloudEvent validation

cmd/                     # Application entry points
└── main.go             # Main orchestration

config/                  # Configuration files
└── *.yaml              # Environment-specific configs
```

### Data Flow

1. **Consumption**: Kafka consumer receives messages from subscribed topics
2. **Validation**: CloudEvents are validated against v1.0 spec
3. **Buffering**: Events buffered per-partition until size/count limits reached
4. **Encoding**: Buffered events encoded to Parquet/Avro with compression
5. **Storage**: Encoded files written to S3/Azure/GCS/filesystem with partitioning
6. **Observability**: Metrics, logs, and health checks throughout

### Key Design Patterns

#### Interface-Based Design

```go
// pkg/ defines interfaces (what)
type Storage interface {
    Write(ctx context.Context, data []byte, path string) error
}

// internal/ implements (how)
type S3Writer struct { ... }
var _ storage.Storage = (*S3Writer)(nil)  // Compile-time verification
```

#### Factory Pattern

```go
// Encoder factory for format-agnostic creation
factory := encoder.NewFactory(format, compression)
encoder, err := factory.CreateEncoder()
```

#### Thread-Safe Buffering

```go
// Manager handles multiple partition buffers
manager := buffer.NewManager(maxSize, maxRecords)
buf := manager.GetOrCreate(partitionID)  // Double-checked locking
```

#### Retry with Backoff

```go
// Exponential backoff for transient failures
for attempt := 0; attempt < maxRetries; attempt++ {
    if err := operation(); err == nil {
        break
    }
    time.Sleep(backoff * time.Duration(1<<attempt))
}
```

### Storage Partitioning

Files are organized with this directory structure:

```
{base_path}/
  {topic}/
    {schema_version}/
      dt={YYYY-MM-DD}/
        pid={partition}/
          {timestamp}_{offset}_{count}.{format}
```

Example:
```
/events/
  user-events/
    v1/
      dt=2025-12-21/
        pid=0/
          20251221_100000_offset12345_count1000.parquet
```

### Configuration Management

Configuration uses hierarchical YAML with environment overrides:

```yaml
kafka:
  bootstrap_servers: ${KAFKA_BROKERS}
  security_protocol: SASL_SSL
  consumer:
    group_id: ${CONSUMER_GROUP}
    topics: ${TOPICS}
    
storage:
  backend: s3  # or azure, file
  format: parquet  # or avro
  compression: snappy  # or gzip, zstd
```

### Observability

#### Metrics

- Event consumption rate
- Buffer utilization
- Encoding latency
- Storage write latency
- Error rates

#### Logging

Structured logging with levels (DEBUG, INFO, WARN, ERROR):

```go
logger.Info("event processed",
    "partition", partitionID,
    "offset", offset,
    "latency_ms", latency.Milliseconds())
```

#### Health Checks

HTTP endpoints for liveness and readiness:

- `GET /health` - Overall health status
- `GET /metrics` - Prometheus metrics (if enabled)

## Requirements

- Go 1.21+
- Kafka cluster with SASL_SSL support
- S3/Azure Blob/filesystem access
- Kubernetes cluster (for deployment)

## Quick Start

### Configuration

Create `config/application.yaml`:

```yaml
kafka:
  bootstrap_servers: ["localhost:9092"]
  security_protocol: SASL_SSL
  consumer:
    group_id: "event-store-test"
    topics: ["test-topic"]

storage:
  backend: file
  format: parquet
  file:
    base_path: "/tmp/events"
```

### Run

```bash
go run cmd/main.go start --config config/application.yaml
```

## Development

### Setup

This project uses Go 1.25+'s `tool` directive to manage development tools in `go.mod`:

```bash
# Install development tools (golangci-lint, etc.)
make tools

# Or manually:
go get github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go mod tidy
```

### Common Commands

```bash
# Run tests
go test ./...
make test

# Run with coverage
go test -cover ./...
make test-coverage

# Build
go build -o bin/event-store cmd/main.go
make build

# Run linter (using go tool)
go tool golangci-lint run
make lint

# Format code
go fmt ./...
make fmt

# Run all checks
make check
```

### Available Make Targets

Run `make help` to see all available targets:
- `build` - Build the application
- `test` - Run tests with race detector
- `test-coverage` - Generate coverage report
- `lint` - Run golangci-lint
- `fmt` - Format code
- `vet` - Run go vet
- `tools` - Install development tools
- `check` - Run fmt, vet, lint, and test

## License

MIT
