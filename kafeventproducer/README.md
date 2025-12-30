# Kafka CloudEvents Producer

A lightweight Kafka producer that generates and publishes CloudEvents to Kafka topics for library management system events.

## Features

- **CloudEvents Support**: Produces CloudEvents v1.0 formatted messages
- **Structured Logging**: Logs all production activities using Zap
- **Event Generation**: Automatic generation of realistic library events
- **Security**: SASL_SSL support (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, AWS MSK IAM)
- **Prometheus Metrics**: Exposes metrics for produced events, errors, and latency
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals for clean shutdown
- **Health Checks**: HTTP endpoints for liveness and readiness probes
- **Multiple Environments**: Pre-configured for local Kafka-KRaft and AWS MSK

## Architecture

```
kafeventproducer/
├── cmd/kafeventproducer/     # Main application entry point
├── internal/
│   ├── config/               # Configuration management (Viper)
│   ├── events/               # CloudEvent domain models
│   ├── generator/            # Event faker/generator
│   ├── kafka/                # Kafka producer with CloudEvents
│   └── metrics/              # Prometheus metrics
├── config/                   # Environment-specific configurations
├── helm/kafeventproducerchart/  # Kubernetes Helm chart
└── Dockerfile                # Multi-stage Docker build
```

## Prerequisites

- Go 1.25.5+
- Docker
- Kubernetes cluster
- Kafka cluster (with SASL_SSL enabled)
- Helm 3.x

## Configuration

The application uses Viper for configuration management. Configuration files are located in the `config/` directory:

- `local-kafka-kraft.yaml` - Local Kafka-KRaft cluster
- `aws-msk.yaml` - AWS MSK cluster

### Key Configuration Options

```yaml
kafka:
  brokers: "kafka-kraft-hs.kafka-lab.svc.cluster.local:9093"
  security:
    protocol: "SASL_SSL"
    sasl:
      mechanism: "PLAIN"  # or SCRAM-SHA-256, SCRAM-SHA-512, AWS_MSK_IAM
  producer:
    acks: "all"
    compression: "snappy"
    batchSize: 16384
    linger: 10ms
  topics:
    bookIssued: "library.books.issued"
    bookReturned: "library.books.returned"

generator:
  intervalSeconds: 5  # Generate events every 5 seconds
  eventTypes:
    - "BookIssued"
    - "BookReturned"
```

## Build and Run

### Local Development

```bash
# Install dependencies
go mod download

# Build
make build

# Run locally
./bin/kafeventproducer --config=config/local-kafka-kraft.yaml --log-level=info
```

### Docker Build

```bash
# Build Docker image
make docker-build

# Or manually
docker build -t kafeventproducer:latest .
```

### Deploy to Kubernetes

```bash
# Deploy with Helm
helm install kafeventproducer ./helm/kafeventproducerchart -n kafka-lab

# Check status
kubectl get pods -n kafka-lab -l app.kubernetes.io/name=kafeventproducer

# View logs
kubectl logs -n kafka-lab -l app.kubernetes.io/name=kafeventproducer -f

# Uninstall
helm uninstall kafeventproducer -n kafka-lab
```

## Event Topics

- `library.books.issued` - Published when a book is issued to a member
- `library.books.returned` - Published when a member returns a book

## CloudEvent Examples

### Book Issued Event

```json
{
  "specversion": "1.0",
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "type": "library.book.issued",
  "source": "library-management-system",
  "time": "2025-12-29T10:30:00Z",
  "datacontenttype": "application/json",
  "data": {
    "bookID": "B12a34b56",
    "title": "The Quick Brown Fox Jumps Over",
    "isbn": "978-1234567890",
    "author": "John Smith",
    "category": "Fiction",
    "memberID": "M98c76d54",
    "memberName": "Jane Doe",
    "memberEmail": "jane.doe@example.com",
    "issueDate": "2025-12-29T10:30:00Z",
    "dueDate": "2026-01-12T10:30:00Z",
    "libraryID": "LIB5a6b7c",
    "branchName": "New York Branch"
  }
}
```

### Book Returned Event

```json
{
  "specversion": "1.0",
  "id": "f9e8d7c6-b5a4-3210-fedc-ba0987654321",
  "type": "library.book.returned",
  "source": "library-management-system",
  "time": "2025-12-29T10:30:00Z",
  "datacontenttype": "application/json",
  "data": {
    "bookID": "Bfe98dc76",
    "title": "A Journey Through Time and Space",
    "isbn": "978-9876543210",
    "memberID": "Mba98fe76",
    "memberName": "Michael Johnson",
    "issueDate": "2025-12-15T09:00:00Z",
    "returnDate": "2025-12-29T10:30:00Z",
    "dueDate": "2025-12-29T09:00:00Z",
    "isLate": true,
    "lateDays": 1,
    "lateFeeAmount": 0.5,
    "libraryID": "LIB9d8e7f",
    "branchName": "Los Angeles Branch",
    "condition": "good"
  }
}
```

## Metrics

Prometheus metrics are exposed on port 9090:

- `kafeventproducer_events_produced_total` - Total events produced (by topic and event_type)
- `kafeventproducer_producer_errors_total` - Total producer errors (by topic and error_type)
- `kafeventproducer_producer_duration_seconds` - Event production duration (by topic)

Access metrics:
```bash
kubectl port-forward -n kafka-lab svc/kafeventproducer 9090:9090
curl http://localhost:9090/metrics
```

## Health Checks

- `/health` - Health check endpoint (returns 200 OK)
- `/metrics` - Prometheus metrics endpoint

## Structured Logging

The producer logs all production activities in structured JSON format:

```json
{
  "level": "info",
  "ts": "2024-01-15T10:30:45.123Z",
  "msg": "Produced CloudEvent",
  "topic": "library.books.issued",
  "event_id": "e123-456",
  "event_type": "library.book.issued",
  "event_source": "library-management-system",
  "partition": 0,
  "offset": 42
}
```

## Environment Variables

- `KAFKA_SASL_USERNAME` - SASL username (from secret)
- `KAFKA_SASL_PASSWORD` - SASL password (from secret)

## Command-Line Flags

- `--config` - Path to configuration file (default: config/application.yaml)
- `--log-level` - Log level: debug, info, warn, error (default: info)
- `--metrics-port` - Metrics server port (default: 9090)

## Testing with kafeventconsumer

1. Deploy kafeventproducer to produce CloudEvents:
   ```bash
   helm install producer ./helm/kafeventproducerchart -n kafka-lab
   ```

2. Deploy kafeventconsumer:
   ```bash
   helm install consumer ./kafeventconsumer/helm/kafeventconsumerchart -n kafka-lab
   ```

3. View producer logs to see produced events:
   ```bash
   kubectl logs -n kafka-lab -l app.kubernetes.io/name=kafeventproducer -f
   ```

## Troubleshooting

### Producer not connecting to Kafka

Check broker connectivity:
```bash
kubectl exec -n kafka-lab deploy/kafeventproducer -- \
  wget -O- --timeout=2 kafka-kraft-hs.kafka-lab.svc.cluster.local:9093
```

### Events not being produced

1. Check if topics exist (or will be auto-created):
   ```bash
   kubectl exec -it -n kafka-lab kafka-kraft-0 -- \
     kafka-topics.sh --bootstrap-server localhost:9092 --list
   ```

2. Verify producer configuration:
   ```bash
   kubectl describe configmap kafeventproducer-config -n kafka-lab
   ```

3. Check producer logs for errors:
   ```bash
   kubectl logs -n kafka-lab -l app.kubernetes.io/name=kafeventproducer --tail=100
   ```

### Topic Auto-Creation

By default, Kafka will auto-create topics when the producer first writes to them. However, for production use, it's recommended to create topics explicitly with appropriate settings:

```bash
# Create topics with 3 partitions and replication factor 3
kubectl exec -it -n kafka-lab kafka-kraft-0 -- \
  kafka-topics.sh --bootstrap-server localhost:9092 \
  --create --topic library.books.issued \
  --partitions 3 --replication-factor 3

kubectl exec -it -n kafka-lab kafka-kraft-0 -- \
  kafka-topics.sh --bootstrap-server localhost:9092 \
  --create --topic library.books.returned \
  --partitions 3 --replication-factor 3
```
