# Kafeventstore Helm Chart

This Helm chart deploys the kafeventstore application - a high-performance Kafka consumer that persists CloudEvents to blob storage (AWS S3, Azure Blob, GCS) in Parquet or Avro formats.

## Features

- ✅ Multi-cloud support (AWS, Azure, GCP, local Kubernetes)
- ✅ Cloud-native authentication (IRSA, Workload Identity, Managed Identity)
- ✅ Horizontal Pod Autoscaling with custom metrics
- ✅ Persistent volumes for local/file storage
- ✅ Service account with RBAC
- ✅ Network policies for security
- ✅ Prometheus ServiceMonitor integration
- ✅ Pod Disruption Budget for high availability
- ✅ Comprehensive health checks and observability
- ✅ Environment-specific value files

## Prerequisites

- Kubernetes 1.21+
- Helm 3.8+
- Persistent Volume provisioner (for file storage backend)
- Cloud provider setup (for cloud storage backends):
  - **AWS**: IAM Roles for Service Accounts (IRSA) configured
  - **Azure**: Workload Identity enabled on AKS cluster
  - **GCP**: Workload Identity enabled on GKE cluster

## Installation

### Quick Start (Local Development)

```bash
# Install with local values
helm install kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml \
  --namespace default

# Or using template
helm template kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml
```

### AWS EKS Deployment

```bash
# 1. Create namespace
kubectl create namespace kafeventstore

# 2. Create secrets for Kafka credentials (if needed)
kubectl create secret generic kafeventstore-kafka-creds \
  --from-literal=username='your-kafka-username' \
  --from-literal=password='your-kafka-password' \
  -n kafeventstore

# 3. Install the chart
helm install kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-aws.yaml \
  --namespace kafeventstore \
  --set serviceAccount.annotations.roleArn="arn:aws:iam::123456789012:role/kafeventstore-role" \
  --set config.storage.s3.bucket="your-events-bucket" \
  --set config.kafka.bootstrapServers="your-kafka-cluster:9092"
```

### Azure AKS Deployment

```bash
# 1. Enable Workload Identity on AKS
az aks update \
  --resource-group myResourceGroup \
  --name myAKSCluster \
  --enable-oidc-issuer \
  --enable-workload-identity

# 2. Create namespace
kubectl create namespace kafeventstore

# 3. Create secrets
kubectl create secret generic kafeventstore-kafka-creds \
  --from-literal=username='your-kafka-username' \
  --from-literal=password='your-kafka-password' \
  -n kafeventstore

# 4. Install the chart
helm install kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-azure.yaml \
  --namespace kafeventstore \
  --set serviceAccount.annotations.azure\.workload\.identity/client-id="your-client-id" \
  --set serviceAccount.annotations.azure\.workload\.identity/tenant-id="your-tenant-id" \
  --set config.storage.azure.accountName="yourstorageaccount" \
  --set config.kafka.bootstrapServers="your-kafka-cluster:9092"
```

### GCP GKE Deployment

```bash
# 1. Enable Workload Identity on GKE
gcloud container clusters update CLUSTER_NAME \
  --workload-pool=PROJECT_ID.svc.id.goog

# 2. Create GCP service account and bind to K8s SA
gcloud iam service-accounts create kafeventstore

gcloud iam service-accounts add-iam-policy-binding \
  kafeventstore@PROJECT_ID.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:PROJECT_ID.svc.id.goog[kafeventstore/kafeventstore]"

# Grant storage permissions
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:kafeventstore@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.objectAdmin"

# 3. Install the chart
helm install kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-gcp.yaml \
  --namespace kafeventstore \
  --set serviceAccount.annotations.iam\.gke\.io/gcp-service-account="kafeventstore@PROJECT_ID.iam.gserviceaccount.com" \
  --set config.storage.gcs.bucket="your-events-bucket" \
  --set config.kafka.bootstrapServers="your-kafka-cluster:9092"
```

## Configuration

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.cloudProvider` | Cloud provider: `local`, `aws`, `azure`, `gcp` | `local` |
| `global.environment` | Environment: `dev`, `staging`, `prod` | `dev` |
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `kafeventstore` |
| `image.tag` | Image tag | Chart appVersion |
| `config.storage.backend` | Storage backend: `s3`, `azure`, `gcs`, `file` | `file` |
| `config.storage.format` | Data format: `parquet`, `avro` | `parquet` |
| `config.kafka.bootstrapServers` | Kafka bootstrap servers | `localhost:9092` |
| `config.kafka.topics` | Kafka topics to consume | `["events"]` |
| `autoscaling.enabled` | Enable HPA | `false` |
| `persistence.enabled` | Enable persistent volume | `false` |
| `serviceMonitor.enabled` | Enable Prometheus ServiceMonitor | `false` |

### Storage Backend Configuration

#### AWS S3
```yaml
config:
  storage:
    backend: "s3"
    s3:
      bucket: "events-bucket"
      region: "us-east-1"
      sseEnabled: true
```

#### Azure Blob
```yaml
config:
  storage:
    backend: "azure"
    azure:
      accountName: "storageaccount"
      container: "events"
      useManagedIdentity: true
```

#### Google Cloud Storage
```yaml
config:
  storage:
    backend: "gcs"
    gcs:
      bucket: "events-bucket"
      projectId: "your-project"
      authMethod: "adc"
```

#### File System
```yaml
config:
  storage:
    backend: "file"
    file:
      basePath: "/data/events"

persistence:
  enabled: true
  size: 10Gi
```

### Kafka Configuration

```yaml
config:
  kafka:
    bootstrapServers: "kafka:9092"
    securityProtocol: "SASL_SSL"
    saslMechanism: "PLAIN"
    consumerGroupId: "kafeventstore"
    topics:
      - "events"
    autoOffsetReset: "earliest"
    
    dlq:
      enabled: true
      topicSuffix: "-dlq"
      maxRetries: 3
```

### Autoscaling

```yaml
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
  
  # Custom metrics (Kafka consumer lag)
  customMetrics:
    - type: External
      external:
        metric:
          name: kafka_consumer_lag
        target:
          type: AverageValue
          averageValue: "1000"
```

## Upgrading

```bash
# Upgrade with new values
helm upgrade kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-aws.yaml \
  --namespace kafeventstore

# Rollback if needed
helm rollback kafeventstore --namespace kafeventstore
```

## Uninstallation

```bash
helm uninstall kafeventstore --namespace kafeventstore
```

## Monitoring

### Prometheus Metrics

The application exposes metrics on port 9090:
- Kafka consumer metrics
- Storage operation metrics
- Buffer metrics
- Application health metrics

Enable ServiceMonitor for Prometheus Operator:
```yaml
serviceMonitor:
  enabled: true
  namespace: monitoring
  interval: 30s
```

### Health Checks

Health endpoints on port 8080:
- `/health` - Liveness probe
- `/ready` - Readiness probe

## Troubleshooting

### Check pod status
```bash
kubectl get pods -n kafeventstore
```

### View logs
```bash
kubectl logs -f deployment/kafeventstore -n kafeventstore
```

### Check events
```bash
kubectl get events -n kafeventstore --sort-by='.lastTimestamp'
```

### Debug configuration
```bash
# View rendered templates
helm template kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml

# Get deployed values
helm get values kafeventstore -n kafeventstore
```

### Common Issues

1. **Pod not starting**: Check image pull secrets and repository access
2. **Kafka connection issues**: Verify bootstrap servers and credentials
3. **Storage access denied**: Check service account annotations and IAM roles
4. **Persistent volume issues**: Verify storage class and PV provisioner

## Security Best Practices

1. **Use cloud-native authentication** (IRSA, Workload Identity) instead of storing credentials
2. **Enable Pod Security Standards**
3. **Use Network Policies** to restrict traffic
4. **Enable read-only root filesystem**
5. **Run as non-root user**
6. **Use Pod Security Context**
7. **Rotate credentials regularly**
8. **Enable audit logging**

## Advanced Configuration

### Using envsubst for Dynamic Values

The chart supports environment variable substitution in configuration files. See [Using envsubst](#using-envsubst) section below.

### Multi-Environment Deployment

Use different values files for each environment:

```bash
# Development
helm install kafeventstore-dev ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml \
  --namespace dev

# Staging
helm install kafeventstore-staging ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-aws.yaml \
  --namespace staging \
  --set global.environment=staging

# Production
helm install kafeventstore-prod ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-aws.yaml \
  --namespace production \
  --set global.environment=production
```

## Using envsubst

For dynamic configuration with environment variables, use [envsubst](https://github.com/a8m/envsubst):

```bash
# Install envsubst
go install github.com/a8m/envsubst/cmd/envsubst@latest

# Use with Helm
export KAFKA_BOOTSTRAP_SERVERS="kafka:9092"
export S3_BUCKET="my-events-bucket"

envsubst < values-template.yaml | helm install kafeventstore ./helm/kafeventstore -f -
```

Example template:
```yaml
# values-template.yaml
config:
  kafka:
    bootstrapServers: "${KAFKA_BOOTSTRAP_SERVERS}"
  storage:
    s3:
      bucket: "${S3_BUCKET}"
```

## Contributing

Please read [CONTRIBUTING.md](../../CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

See [LICENSE](../../LICENSE) for details.
