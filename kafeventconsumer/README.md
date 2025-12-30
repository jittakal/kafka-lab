# Kafka CloudEvents Consumer

A lightweight Kafka consumer that consumes CloudEvents from Kafka topics and logs them to console with structured logging.

## Features

- **CloudEvents Support**: Consumes CloudEvents v1.0 formatted messages
- **Structured Logging**: Logs all CloudEvent fields (id, type, source, time, data) using Zap
- **Kafka Consumer Group**: Supports consumer group with automatic rebalancing
- **Security**: SASL_SSL support (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512, AWS MSK IAM)
- **Prometheus Metrics**: Exposes metrics for consumed events, errors, and offsets
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals for clean shutdown
- **Health Checks**: HTTP endpoints for liveness and readiness probes
- **Multiple Environments**: Pre-configured for local Kafka-KRaft and AWS MSK

## Architecture

```
kafeventconsumer/
├── cmd/kafeventconsumer/     # Main application entry point
├── internal/
│   ├── config/               # Configuration management (Viper)
│   ├── kafka/                # Kafka consumer with CloudEvents
│   └── metrics/              # Prometheus metrics
├── config/                   # Environment-specific configurations
├── helm/kafeventconsumer/    # Kubernetes Helm chart
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
  consumer:
    groupId: "kafeventconsumer-group"
    offsetInitial: "oldest"  # or "newest"
    autoCommit: true
  topics:
    - "library.books.issued"
    - "library.books.returned"
```

## Build and Run

### Local Development

```bash
# Install dependencies
go mod download

# Build
make build

# Run locally
./bin/kafeventconsumer --config=config/local-kafka-kraft.yaml --log-level=info
```

### Docker Build

```bash
# Build Docker image
make docker-build

# Or manually
docker build -t kafeventconsumer:latest .
```

### Deploy to Kubernetes

```bash
# Deploy with Helm
helm install kafeventconsumer ./helm/kafeventconsumerchart -n kafka-lab

# Check status
kubectl get pods -n kafka-lab -l app.kubernetes.io/name=kafeventconsumer

# View logs
kubectl logs -n kafka-lab -l app.kubernetes.io/name=kafeventconsumer -f

# Uninstall
helm uninstall kafeventconsumer -n kafka-lab
```

## Metrics

Prometheus metrics are exposed on port 9090:

- `kafeventconsumer_events_consumed_total` - Total events consumed (by topic and event_type)
- `kafeventconsumer_consumer_errors_total` - Total consumer errors (by topic and error_type)
- `kafeventconsumer_last_consumed_offset` - Last consumed offset (by topic and partition)

Access metrics:
```bash
kubectl port-forward -n kafka-lab svc/kafeventconsumer 9090:9090
curl http://localhost:9090/metrics
```

## Health Checks

- `/health` - Health check endpoint (returns 200 OK)
- `/metrics` - Prometheus metrics endpoint

## Structured Logging

The consumer logs all CloudEvent fields in structured JSON format:

```json
{
  "level": "info",
  "ts": "2024-01-15T10:30:45.123Z",
  "msg": "Consumed CloudEvent",
  "topic": "library.books.issued",
  "partition": 0,
  "offset": 42,
  "event_id": "e123-456",
  "event_type": "com.library.books.issued.v1",
  "event_source": "/library/circulation",
  "event_time": "2024-01-15T10:30:44Z",
  "data": {...}
}
```

## Environment Variables

- `KAFKA_SASL_USERNAME` - SASL username (from secret)
- `KAFKA_SASL_PASSWORD` - SASL password (from secret)

## Command-Line Flags

- `--config` - Path to configuration file (default: config/application.yaml)
- `--log-level` - Log level: debug, info, warn, error (default: info)
- `--metrics-port` - Metrics server port (default: 9090)

## Testing with kafeventproducer

1. Deploy kafeventproducer to produce CloudEvents:
   ```bash
   helm install producer ./kafeventproducer/helm/kafeventproducer -n kafka-lab
   ```

2. Deploy kafeventconsumer:
   ```bash
   helm install consumer ./helm/kafeventconsumer -n kafka-lab
   ```

3. View consumer logs to see consumed events:
   ```bash
   kubectl logs -n kafka-lab -l app.kubernetes.io/name=kafeventconsumer -f
   ```

## Troubleshooting

### Consumer not connecting to Kafka

Check broker connectivity:
```bash
kubectl exec -n kafka-lab deploy/kafeventconsumer -- \
  wget -O- --timeout=2 kafka-kraft-hs.kafka-lab.svc.cluster.local:9093
```

### No events being consumed

1. Check if topics exist:
   ```bash
   kubectl exec -it -n kafka-lab kafka-kraft-0 -- \
     kafka-topics.sh --bootstrap-server localhost:9092 --list
   ```

2. Verify consumer group:
   ```bash
   kubectl exec -it -n kafka-lab kafka-kraft-0 -- \
     kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
     --group kafeventconsumer-group --describe
   ```

3. Check consumer logs for errors:
   ```bash
   kubectl logs -n kafka-lab -l app.kubernetes.io/name=kafeventconsumer --tail=100
   ```

## Development

### Run Tests

```bash
make test
```

### Lint

```bash
make lint
```

### Clean

```bash
make clean
```

## License

MIT
