# Kafka Helm Charts

This directory contains Helm charts for deploying Apache Kafka on Kubernetes.

## Available Charts

### 1. kafka-kraft-sasl-ssl (Recommended)

**Location**: `kaflocalhelm/kafka-kraft-sasl-ssl/`

A production-ready Helm chart for Apache Kafka 4.0.0 with KRaft mode and SASL_SSL security.

**Key Features**:
- ✅ Apache Kafka 4.0.0 with KRaft mode (no ZooKeeper)
- ✅ SASL_SSL security with cert-manager integration
- ✅ Latest cert-manager API (v1)
- ✅ Single-node and multi-node cluster support
- ✅ Comprehensive documentation and testing
- ✅ Follows kafeventstore standards
- ✅ Ready for local Kubernetes (Rancher, K3s, etc.)

**Quick Start**:
```bash
# Deploy single-node
./kafka-kraft-sasl-ssl/deploy.sh

# Deploy 3-node cluster
./kafka-kraft-sasl-ssl/deploy.sh --replicas 3
```

**Documentation**:
- [README](kafka-kraft-sasl-ssl/README.md) - Full documentation
- [Quick Start](kafka-kraft-sasl-ssl/QUICKSTART.md) - Quick reference
- [Deployment Guide](kafka-kraft-sasl-ssl/DEPLOYMENT_GUIDE.md) - Step-by-step guide
- [Testing Guide](kafka-kraft-sasl-ssl/TESTING.md) - Testing procedures
- [Chart Summary](kafka-kraft-sasl-ssl/CHART_SUMMARY.md) - Overview and features

### 2. kafka-sasl-ssl (Legacy)

**Location**: `kaflocalhelm/kafka-sasl-ssl/`

Older chart using Kafka 3.9.0. Kept for reference.

**Note**: Use `kafka-kraft-sasl-ssl` for new deployments.

## Comparison

| Feature | kafka-sasl-ssl | kafka-kraft-sasl-ssl |
|---------|----------------|----------------------|
| Kafka Version | 3.9.0 | 4.0.0 |
| cert-manager API | Older | v1 (latest) |
| Documentation | Basic | Comprehensive |
| Deployment Script | No | Yes |
| Testing Suite | Limited | Comprehensive |
| Standards | Basic | kafeventstore compliant |
| Status | Legacy | **Recommended** |

## Prerequisites

All charts require:
- Kubernetes 1.21+
- Helm 3.8+
- cert-manager 1.13+ (for SSL certificates)
- Storage class with dynamic provisioning

## Installation

### cert-manager (Required)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
kubectl wait --for=condition=ready pod --all -n cert-manager --timeout=300s
```

### Kafka Cluster

```bash
# Using the recommended chart
cd kafka-kraft-sasl-ssl
./deploy.sh
```

Or manually:

```bash
helm install kafka ./kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --create-namespace
```

## Connection Information

After deployment, connect to Kafka at:

```
Bootstrap Server: kafka-hs.kafka-lab.svc.cluster.local:9093
Security Protocol: SASL_SSL
SASL Mechanism: PLAIN
Username: admin
Password: admin-secret
```

**Important**: Change credentials in production!

## Integration with kafeventstore

Both charts work with the kafeventstore application:

```bash
helm install kafeventstore ../kafeventstore/helm/kafeventstore \
  --set config.kafka.bootstrapServers=kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --set config.kafka.security.protocol=SASL_SSL \
  --set config.kafka.sasl.username=admin \
  --set config.kafka.sasl.password=admin-secret
```

## Testing

### Quick Test

```bash
# Test deployment
helm test kafka -n kafka-lab

# List topics
kubectl run kafka-client --rm -it --restart=Never \
  --image=apache/kafka:4.0.0 --namespace=kafka-lab -- \
  kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties --list
```

### Comprehensive Testing

See [kafka-kraft-sasl-ssl/TESTING.md](kafka-kraft-sasl-ssl/TESTING.md) for detailed testing procedures.

## Cleanup

```bash
# Uninstall Kafka
helm uninstall kafka -n kafka-lab

# Delete namespace (removes all resources including PVCs)
kubectl delete namespace kafka-lab
```

## Support

For issues:
1. Check the chart-specific README
2. Review pod logs: `kubectl logs kafka-0 -n kafka-lab`
3. Check events: `kubectl get events -n kafka-lab`
4. Consult [Apache Kafka Documentation](https://kafka.apache.org/documentation/)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     kafka-lab namespace                  │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌───────────────┐  ┌───────────────┐  ┌──────────────┐│
│  │   kafka-0     │  │   kafka-1     │  │   kafka-2    ││
│  │  (KRaft Mode) │  │  (KRaft Mode) │  │ (KRaft Mode) ││
│  │  Broker +     │  │  Broker +     │  │  Broker +    ││
│  │  Controller   │  │  Controller   │  │  Controller  ││
│  └───────┬───────┘  └───────┬───────┘  └──────┬───────┘│
│          │                  │                   │        │
│          └──────────────────┴───────────────────┘        │
│                             │                            │
│                    ┌────────▼────────┐                   │
│                    │  Headless Svc   │                   │
│                    │   (kafka-hs)    │                   │
│                    └─────────────────┘                   │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │         cert-manager (SSL/TLS Certs)             │  │
│  └──────────────────────────────────────────────────┘  │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │       Persistent Volumes (per broker)            │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Contributing

Follow the standards established in the kafeventstore project:
- Comprehensive documentation
- Helm best practices
- Security by default
- Production-ready configuration

## License

These charts follow the same license as Apache Kafka.

---

**Recommended**: Use `kafka-kraft-sasl-ssl` for all new deployments.
