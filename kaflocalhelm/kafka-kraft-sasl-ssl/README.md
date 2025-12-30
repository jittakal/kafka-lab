# Kafka KRaft SASL_SSL Helm Chart

A production-ready Helm chart for deploying Apache Kafka 4.x with KRaft mode and SASL_SSL security on Kubernetes.

## Features

- ✅ **Apache Kafka 4.0.0** with KRaft mode (no ZooKeeper dependency)
- ✅ **SASL_SSL Security** with automatic certificate management via cert-manager
- ✅ **Flexible Deployment** - Single-node or multi-node cluster support
- ✅ **Production-Ready** - Pod Disruption Budgets, health checks, resource limits
- ✅ **Persistence** - StatefulSet with persistent volumes for data durability
- ✅ **Latest cert-manager API** - Uses cert-manager.io/v1 API
- ✅ **Cloud-Native** - Designed for Kubernetes environments
- ✅ **Observability Ready** - JMX metrics, configurable monitoring
- ✅ **Comprehensive Documentation** - Easy to deploy and operate

## Prerequisites

### Required

- **Kubernetes**: 1.21+ (tested on Rancher Desktop/K3s)
- **Helm**: 3.8+
- **cert-manager**: 1.13+ installed in the cluster
- **Storage Class**: A storage class supporting dynamic provisioning (e.g., `local-path`)

### Optional

- **Prometheus** - For metrics collection (if monitoring is enabled)
- **Grafana** - For metrics visualization

## Quick Start

### 1. Install cert-manager (if not already installed)

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml

# Verify cert-manager is running
kubectl get pods -n cert-manager
```

### 2. Create the namespace

```bash
kubectl create namespace kafka-lab
```

### 3. Install the chart

```bash
# From the chart directory
helm install kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --create-namespace
```

### 4. Wait for deployment to complete

```bash
# Watch the pods
kubectl get pods -n kafka-lab -w

# Check the status
kubectl get all -n kafka-lab
```

### 5. Test the deployment

```bash
# Run Helm tests
helm test kafka -n kafka-lab

# Or manually test
kubectl run kafka-client --rm -it --restart=Never \
  --image=apache/kafka:4.0.0 \
  --namespace=kafka-lab \
  -- bash
```

## Installation Options

### Single-Node Deployment (Default)

Perfect for development and testing:

```bash
helm install kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --set replicaCount=1
```

### Multi-Node Cluster

For production deployments with high availability:

```bash
helm install kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --set replicaCount=3 \
  --set kafka.defaultReplicationFactor=3 \
  --set kafka.offsetsTopicReplicationFactor=3 \
  --set pdb.enabled=true \
  --set pdb.minAvailable=2
```

### Custom Configuration

```bash
helm install kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --set security.sasl.username=myuser \
  --set security.sasl.password=mypassword \
  --set persistence.size=10Gi \
  --set resources.requests.memory=4Gi
```

## Configuration

### Key Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of Kafka brokers | `1` |
| `image.repository` | Kafka image repository | `apache/kafka` |
| `image.tag` | Kafka image tag | `4.0.0` |
| `kafka.clusterId` | KRaft cluster ID (base64 UUID) | `5L6g3nShT-eMCtK--X86sw` |
| `kafka.autoCreateTopics` | Auto-create topics | `true` |
| `kafka.numPartitions` | Default number of partitions | `3` |
| `kafka.defaultReplicationFactor` | Default replication factor | `1` |
| `security.sasl.username` | SASL username | `admin` |
| `security.sasl.password` | SASL password | `admin-secret` |
| `security.saslMechanism` | SASL mechanism (PLAIN, SCRAM-SHA-256) | `PLAIN` |
| `certManager.enabled` | Enable cert-manager for TLS | `true` |
| `certManager.keystorePassword` | Keystore/Truststore password | `changeit` |
| `persistence.enabled` | Enable persistent storage | `true` |
| `persistence.size` | Size of data volume per broker | `5Gi` |
| `persistence.storageClass` | Storage class name | `local-path` |
| `resources.requests.memory` | Memory request | `2Gi` |
| `resources.limits.memory` | Memory limit | `4Gi` |

### Full Configuration

See [values.yaml](values.yaml) for all available configuration options.

## Usage Examples

### Get Client Configuration

Extract the client configuration for external applications:

```bash
# Get client.properties
kubectl get secret kafka-client-config \
  -n kafka-lab \
  -o jsonpath='{.data.client\.properties}' | base64 -d > client.properties

# Get client certificates
kubectl get secret kafka-client-tls-certificates \
  -n kafka-lab \
  -o jsonpath='{.data.keystore\.p12}' | base64 -d > client-keystore.p12

kubectl get secret kafka-client-tls-certificates \
  -n kafka-lab \
  -o jsonpath='{.data.truststore\.p12}' | base64 -d > client-truststore.p12
```

### Connect to Kafka from Inside the Cluster

```bash
# Start an interactive Kafka client pod
kubectl run kafka-client --rm -it --restart=Never \
  --image=apache/kafka:4.0.0 \
  --namespace=kafka-lab \
  -- bash

# Inside the pod, list topics
kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --list
```

### Create a Topic

```bash
kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --create \
  --topic my-topic \
  --partitions 3 \
  --replication-factor 1
```

### Produce Messages

```bash
kafka-console-producer.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --producer.config /etc/kafka/client/secrets/client.properties \
  --topic my-topic
```

### Consume Messages

```bash
kafka-console-consumer.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --consumer.config /etc/kafka/client/secrets/client.properties \
  --topic my-topic \
  --from-beginning
```

## Accessing Kafka

### Internal Access (from within Kubernetes)

Bootstrap server address:
```
kafka-hs.kafka-lab.svc.cluster.local:9093
```

For individual brokers:
```
kafka-0.kafka-hs.kafka-lab.svc.cluster.local:9093
kafka-1.kafka-hs.kafka-lab.svc.cluster.local:9093
kafka-2.kafka-hs.kafka-lab.svc.cluster.local:9093
```

### External Access (via NodePort)

If you need external access, update the service type:

```bash
helm upgrade kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --set broker.service.type=NodePort \
  --set broker.service.baseNodePort=30090
```

Then access via:
```
<NODE_IP>:30090
```

## Security

### SASL Authentication

The chart uses SASL/PLAIN authentication by default. Credentials:
- **Username**: `admin` (configurable via `security.sasl.username`)
- **Password**: `admin-secret` (configurable via `security.sasl.password`)

**Important**: Change the default credentials in production!

```bash
helm install kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --set security.sasl.username=produser \
  --set security.sasl.password='<strong-password>'
```

### SSL/TLS Certificates

The chart automatically generates SSL certificates using cert-manager:
- **CA Certificate**: Self-signed CA for the cluster
- **Server Certificate**: For Kafka brokers
- **Client Certificate**: For Kafka clients

Certificates are valid for 10 years and auto-renew 30 days before expiration.

### Adding Additional Users

```yaml
security:
  sasl:
    users:
      - username: "producer"
        password: "producer-secret"
      - username: "consumer"
        password: "consumer-secret"
```

## Monitoring

### Enable JMX Metrics

```bash
helm install kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --set monitoring.jmx.enabled=true \
  --set monitoring.jmx.port=9090
```

### Prometheus Integration

Add Prometheus annotations to scrape metrics:

```yaml
podAnnotations:
  prometheus.io/scrape: "true"
  prometheus.io/port: "9090"
  prometheus.io/path: "/metrics"
```

## Maintenance

### Upgrade the Chart

```bash
helm upgrade kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --reuse-values \
  --set image.tag=4.0.1
```

### Scale the Cluster

```bash
# Scale up to 3 replicas
helm upgrade kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --reuse-values \
  --set replicaCount=3 \
  --set kafka.defaultReplicationFactor=3
```

### Backup Data

```bash
# Get persistent volume claims
kubectl get pvc -n kafka-lab

# Backup using your preferred backup solution
# (e.g., Velero, custom scripts, cloud provider snapshots)
```

### View Logs

```bash
# All Kafka pods
kubectl logs -n kafka-lab -l app.kubernetes.io/name=kafka-kraft-sasl-ssl --tail=100

# Specific broker
kubectl logs -n kafka-lab kafka-0 --tail=100 -f
```

## Troubleshooting

### Pods not starting

1. Check cert-manager is installed and running:
```bash
kubectl get pods -n cert-manager
```

2. Check certificate status:
```bash
kubectl get certificates -n kafka-lab
kubectl describe certificate kafka-ca -n kafka-lab
```

3. Check pod events:
```bash
kubectl describe pod kafka-0 -n kafka-lab
```

### Connection issues

1. Verify the service is running:
```bash
kubectl get svc -n kafka-lab
```

2. Test network connectivity:
```bash
kubectl run test --rm -it --restart=Never --image=busybox -n kafka-lab -- sh
# Inside the pod:
nc -zv kafka-hs.kafka-lab.svc.cluster.local 9093
```

3. Check SASL credentials:
```bash
kubectl get secret kafka-sasl-jaas -n kafka-lab -o yaml
```

### Storage issues

1. Check PVCs:
```bash
kubectl get pvc -n kafka-lab
```

2. Check storage class:
```bash
kubectl get storageclass
```

## Uninstallation

```bash
# Delete the Helm release
helm uninstall kafka -n kafka-lab

# Delete the namespace (this will delete all resources including PVCs)
kubectl delete namespace kafka-lab
```

**Warning**: This will permanently delete all Kafka data!

## Advanced Configuration

### Custom Kafka Configuration

You can add custom Kafka configurations via environment variables:

```yaml
extraEnv:
  - name: KAFKA_MESSAGE_MAX_BYTES
    value: "10485760"
  - name: KAFKA_REPLICA_FETCH_MAX_BYTES
    value: "10485760"
```

### Node Affinity

Run Kafka on specific nodes:

```yaml
nodeSelector:
  kafka: "true"

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - kafka-kraft-sasl-ssl
        topologyKey: kubernetes.io/hostname
```

### Resource Limits

Adjust resources based on your workload:

```yaml
resources:
  requests:
    memory: "4Gi"
    cpu: "2000m"
  limits:
    memory: "8Gi"
    cpu: "4000m"
```

## Integration with kafeventstore

This Kafka cluster can be used with the kafeventstore application:

```bash
# Install Kafka
helm install kafka ./kaflocalhelm/kafka-kraft-sasl-ssl --namespace kafka-lab

# Install kafeventstore with Kafka connection
helm install kafeventstore ./kafeventstore/helm/kafeventstore \
  --namespace default \
  --set config.kafka.bootstrapServers=kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --set config.kafka.security.protocol=SASL_SSL \
  --set config.kafka.sasl.mechanism=PLAIN \
  --set config.kafka.sasl.username=admin \
  --set config.kafka.sasl.password=admin-secret
```

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     kafka-lab namespace                  │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌───────────────┐  ┌───────────────┐  ┌──────────────┐│
│  │   kafka-0     │  │   kafka-1     │  │   kafka-2    ││
│  │  (KRaft Mode) │  │  (KRaft Mode) │  │ (KRaft Mode) ││
│  │               │  │               │  │              ││
│  │  Broker +     │  │  Broker +     │  │  Broker +    ││
│  │  Controller   │  │  Controller   │  │  Controller  ││
│  └───────────────┘  └───────────────┘  └──────────────┘│
│         │                  │                   │        │
│         └──────────────────┴───────────────────┘        │
│                           │                             │
│                  ┌────────▼────────┐                    │
│                  │  Headless Svc   │                    │
│                  │   (kafka-hs)    │                    │
│                  └─────────────────┘                    │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │            cert-manager (TLS/SSL)                 │  │
│  │  - CA Certificate                                 │  │
│  │  - Server Certificates (per broker)              │  │
│  │  - Client Certificates                           │  │
│  └──────────────────────────────────────────────────┘  │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │         Persistent Volumes (per broker)          │  │
│  │  - Kafka data (logs, metadata)                   │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Contributing

Contributions are welcome! Please follow the standards established in the kafeventstore project.

## License

This chart follows the same license as Apache Kafka.

## Support

For issues and questions:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review Kafka logs: `kubectl logs -n kafka-lab kafka-0`
3. Check Kubernetes events: `kubectl get events -n kafka-lab`
4. Refer to [Apache Kafka Documentation](https://kafka.apache.org/documentation/)

## References

- [Apache Kafka](https://kafka.apache.org/)
- [KRaft Mode](https://kafka.apache.org/documentation/#kraft)
- [cert-manager](https://cert-manager.io/)
- [Kubernetes StatefulSets](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
