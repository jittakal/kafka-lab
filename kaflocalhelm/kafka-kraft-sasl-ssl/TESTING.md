# Testing Guide - Kafka KRaft SASL_SSL

This guide covers various testing scenarios for the Kafka KRaft SASL_SSL Helm chart.

## Table of Contents

1. [Pre-deployment Testing](#pre-deployment-testing)
2. [Post-deployment Testing](#post-deployment-testing)
3. [Functional Testing](#functional-testing)
4. [Performance Testing](#performance-testing)
5. [Security Testing](#security-testing)
6. [High Availability Testing](#high-availability-testing)

## Pre-deployment Testing

### Validate Helm Chart

```bash
# Lint the chart
helm lint ./kaflocalhelm/kafka-kraft-sasl-ssl

# Dry run to see generated manifests
helm install kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --namespace kafka-lab \
  --dry-run --debug > rendered-manifests.yaml

# Check for syntax errors
kubectl apply --dry-run=client -f rendered-manifests.yaml
```

### Template Validation

```bash
# Test with different replica counts
helm template kafka ./kaflocalhelm/kafka-kraft-sasl-ssl --set replicaCount=1 | kubectl apply --dry-run=client -f -
helm template kafka ./kaflocalhelm/kafka-kraft-sasl-ssl --set replicaCount=3 | kubectl apply --dry-run=client -f -

# Test with different configurations
helm template kafka ./kaflocalhelm/kafka-kraft-sasl-ssl \
  --set security.sasl.username=testuser \
  --set security.sasl.password=testpass \
  | kubectl apply --dry-run=client -f -
```

## Post-deployment Testing

### Helm Test Suite

Run the built-in Helm tests:

```bash
# Run all tests
helm test kafka -n kafka-lab

# View test logs
kubectl logs -n kafka-lab kafka-test-connection
```

### Pod Health Checks

```bash
# Check all pods are running
kubectl get pods -n kafka-lab -l app.kubernetes.io/name=kafka-kraft-sasl-ssl

# Check pod readiness
kubectl get pods -n kafka-lab -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.conditions[?(@.type=="Ready")].status}{"\n"}{end}'

# Check pod logs for errors
kubectl logs -n kafka-lab kafka-0 --tail=100 | grep -i error
```

### Service Validation

```bash
# Check services
kubectl get svc -n kafka-lab

# Test headless service DNS resolution
kubectl run dns-test --rm -it --restart=Never --image=busybox -n kafka-lab -- nslookup kafka-hs.kafka-lab.svc.cluster.local

# Test individual broker DNS
kubectl run dns-test --rm -it --restart=Never --image=busybox -n kafka-lab -- nslookup kafka-0.kafka-hs.kafka-lab.svc.cluster.local
```

### Certificate Validation

```bash
# Check all certificates are ready
kubectl get certificates -n kafka-lab

# Verify certificate details
kubectl get certificate kafka-server-tls -n kafka-lab -o yaml

# Check certificate secrets
kubectl get secret kafka-tls-certificates -n kafka-lab -o jsonpath='{.data}' | jq 'keys'
```

## Functional Testing

### Topic Management

```bash
# Start a client pod
kubectl run kafka-client --rm -it --restart=Never \
  --image=apache/kafka:4.0.0 \
  --namespace=kafka-lab \
  -- bash

# Inside the pod:

# 1. List topics
kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --list

# 2. Create a topic
kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --create \
  --topic test-topic \
  --partitions 3 \
  --replication-factor 1

# 3. Describe topic
kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --describe \
  --topic test-topic

# 4. Delete topic
kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --delete \
  --topic test-topic
```

### Producer/Consumer Testing

```bash
# Terminal 1: Start producer
kubectl run kafka-producer --rm -it --restart=Never \
  --image=apache/kafka:4.0.0 \
  --namespace=kafka-lab \
  -- kafka-console-producer.sh \
    --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
    --producer.config /etc/kafka/client/secrets/client.properties \
    --topic test-topic

# Terminal 2: Start consumer
kubectl run kafka-consumer --rm -it --restart=Never \
  --image=apache/kafka:4.0.0 \
  --namespace=kafka-lab \
  -- kafka-console-consumer.sh \
    --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
    --consumer.config /etc/kafka/client/secrets/client.properties \
    --topic test-topic \
    --from-beginning

# Type messages in producer and verify they appear in consumer
```

### Consumer Groups

```bash
# Create a consumer group
kafka-console-consumer.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --consumer.config /etc/kafka/client/secrets/client.properties \
  --topic test-topic \
  --group test-group \
  --from-beginning

# List consumer groups
kafka-consumer-groups.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --list

# Describe consumer group
kafka-consumer-groups.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --describe \
  --group test-group
```

## Performance Testing

### Throughput Test

```bash
# Start a performance test pod
kubectl run kafka-perf-test --rm -it --restart=Never \
  --image=apache/kafka:4.0.0 \
  --namespace=kafka-lab \
  -- bash

# Inside the pod:

# Producer performance test
kafka-producer-perf-test.sh \
  --topic perf-test \
  --num-records 100000 \
  --record-size 1000 \
  --throughput -1 \
  --producer-props \
    bootstrap.servers=kafka-hs.kafka-lab.svc.cluster.local:9093 \
    security.protocol=SASL_SSL \
    sasl.mechanism=PLAIN \
    sasl.jaas.config='org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="admin-secret";' \
    ssl.truststore.location=/etc/kafka/client/secrets/truststore.p12 \
    ssl.truststore.password=changeit \
    ssl.keystore.location=/etc/kafka/client/secrets/keystore.p12 \
    ssl.keystore.password=changeit \
    acks=all

# Consumer performance test
kafka-consumer-perf-test.sh \
  --topic perf-test \
  --messages 100000 \
  --threads 1 \
  --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --consumer.config /etc/kafka/client/secrets/client.properties
```

### Latency Test

```bash
# End-to-end latency test
kafka-producer-perf-test.sh \
  --topic latency-test \
  --num-records 10000 \
  --record-size 100 \
  --throughput 100 \
  --producer-props \
    bootstrap.servers=kafka-hs.kafka-lab.svc.cluster.local:9093 \
    security.protocol=SASL_SSL \
    sasl.mechanism=PLAIN \
    sasl.jaas.config='org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="admin-secret";' \
    ssl.truststore.location=/etc/kafka/client/secrets/truststore.p12 \
    ssl.truststore.password=changeit \
    acks=all \
  --print-metrics
```

## Security Testing

### Authentication Test

```bash
# Test with correct credentials (should succeed)
kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --list

# Test with wrong credentials (should fail)
cat > /tmp/wrong-creds.properties <<EOF
security.protocol=SASL_SSL
sasl.mechanism=PLAIN
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username="wrong" password="wrong";
ssl.truststore.type=PKCS12
ssl.truststore.location=/etc/kafka/client/secrets/truststore.p12
ssl.truststore.password=changeit
EOF

kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /tmp/wrong-creds.properties \
  --list
# Expected: Authentication error
```

### TLS Verification

```bash
# Verify SSL connection
openssl s_client -connect kafka-0.kafka-hs.kafka-lab.svc.cluster.local:9093 \
  -showcerts < /dev/null

# Check certificate expiration
kubectl get secret kafka-tls-certificates -n kafka-lab -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | \
  openssl x509 -noout -dates
```

## High Availability Testing

### Pod Failure Recovery

```bash
# Delete a pod and verify it recovers
kubectl delete pod kafka-0 -n kafka-lab

# Wait for pod to be recreated
kubectl wait --for=condition=ready pod kafka-0 -n kafka-lab --timeout=300s

# Verify data is still accessible
kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --list
```

### Rolling Restart

```bash
# Perform a rolling restart
kubectl rollout restart statefulset kafka -n kafka-lab

# Monitor the restart
kubectl rollout status statefulset kafka -n kafka-lab

# Verify cluster is healthy
kubectl get pods -n kafka-lab
```

### Network Partition Simulation

```bash
# Create a network policy to simulate partition
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: kafka-partition-test
  namespace: kafka-lab
spec:
  podSelector:
    matchLabels:
      statefulset.kubernetes.io/pod-name: kafka-0
  policyTypes:
  - Ingress
  - Egress
  ingress: []
  egress: []
EOF

# Wait and observe behavior
sleep 30

# Remove the network policy
kubectl delete networkpolicy kafka-partition-test -n kafka-lab

# Verify cluster recovers
kubectl get pods -n kafka-lab
```

## Stress Testing

### Multi-Topic Test

```bash
# Create multiple topics
for i in {1..10}; do
  kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
    --command-config /etc/kafka/client/secrets/client.properties \
    --create \
    --topic stress-test-$i \
    --partitions 3 \
    --replication-factor 1
done

# Produce to all topics simultaneously
for i in {1..10}; do
  (kafka-producer-perf-test.sh \
    --topic stress-test-$i \
    --num-records 10000 \
    --record-size 1000 \
    --throughput -1 \
    --producer-props \
      bootstrap.servers=kafka-hs.kafka-lab.svc.cluster.local:9093 \
      security.protocol=SASL_SSL \
      sasl.mechanism=PLAIN \
      sasl.jaas.config='org.apache.kafka.common.security.plain.PlainLoginModule required username="admin" password="admin-secret";' \
      ssl.truststore.location=/etc/kafka/client/secrets/truststore.p12 \
      ssl.truststore.password=changeit \
      acks=all &)
done

# Wait for all to complete
wait

# Clean up
for i in {1..10}; do
  kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
    --command-config /etc/kafka/client/secrets/client.properties \
    --delete \
    --topic stress-test-$i
done
```

## Monitoring and Observability

### Check JMX Metrics (if enabled)

```bash
# Port-forward JMX port
kubectl port-forward kafka-0 9090:9090 -n kafka-lab

# Use JConsole or VisualVM to connect to localhost:9090
```

### Check Kafka Metrics

```bash
# Get broker metrics
kafka-broker-api-versions.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties

# Check metadata
kafka-metadata.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties
```

## Test Summary Script

Create a comprehensive test script:

```bash
#!/bin/bash
# save as: test-kafka.sh

set -e

NAMESPACE="kafka-lab"
BOOTSTRAP_SERVER="kafka-hs.kafka-lab.svc.cluster.local:9093"

echo "=== Kafka Cluster Test Suite ==="

echo "1. Testing pod health..."
kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=kafka-kraft-sasl-ssl

echo "2. Testing certificates..."
kubectl get certificates -n $NAMESPACE

echo "3. Testing topic creation..."
kubectl run kafka-test --rm -it --restart=Never \
  --image=apache/kafka:4.0.0 \
  --namespace=$NAMESPACE \
  -- kafka-topics.sh --bootstrap-server $BOOTSTRAP_SERVER \
    --command-config /etc/kafka/client/secrets/client.properties \
    --create --topic test-$(date +%s) --partitions 1 --replication-factor 1

echo "4. Running Helm tests..."
helm test kafka -n $NAMESPACE

echo "=== All tests completed ==="
```

## Cleanup After Testing

```bash
# Delete test topics
kafka-topics.sh --bootstrap-server kafka-hs.kafka-lab.svc.cluster.local:9093 \
  --command-config /etc/kafka/client/secrets/client.properties \
  --delete \
  --topic test-topic

# Remove test pods
kubectl delete pod kafka-client kafka-producer kafka-consumer -n kafka-lab --ignore-not-found

# Clear test data
# (Data will be removed when PVCs are deleted)
```

## Continuous Testing

For CI/CD pipelines, create an automated test suite:

```yaml
# .github/workflows/test-kafka-helm.yaml
name: Test Kafka Helm Chart

on:
  push:
    paths:
      - 'kaflocalhelm/kafka-kraft-sasl-ssl/**'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Kubernetes
        uses: helm/kind-action@v1
      
      - name: Install cert-manager
        run: |
          kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
          kubectl wait --for=condition=ready pod --all -n cert-manager --timeout=300s
      
      - name: Install Kafka
        run: |
          helm install kafka ./kaflocalhelm/kafka-kraft-sasl-ssl --namespace kafka-lab --create-namespace
          kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kafka-kraft-sasl-ssl -n kafka-lab --timeout=600s
      
      - name: Run tests
        run: helm test kafka -n kafka-lab
```

## Troubleshooting Test Failures

### Test Pod Issues

```bash
# Get test pod logs
kubectl logs -n kafka-lab kafka-test-connection

# Describe test pod
kubectl describe pod -n kafka-lab kafka-test-connection

# Check test pod events
kubectl get events -n kafka-lab --field-selector involvedObject.name=kafka-test-connection
```

### Connection Failures

```bash
# Verify service endpoints
kubectl get endpoints -n kafka-lab kafka-hs

# Test TCP connectivity
kubectl run netcat-test --rm -it --restart=Never --image=busybox -n kafka-lab -- sh -c "nc -zv kafka-hs.kafka-lab.svc.cluster.local 9093"

# Check DNS resolution
kubectl run dns-test --rm -it --restart=Never --image=busybox -n kafka-lab -- nslookup kafka-hs.kafka-lab.svc.cluster.local
```

## Performance Baseline

Expected performance for single-node deployment (resource-dependent):
- **Throughput**: 10,000+ messages/sec
- **Latency**: < 10ms (p99)
- **CPU Usage**: < 50% under normal load
- **Memory Usage**: < 2GB under normal load

Multi-node deployment should show improved throughput and availability.

## Conclusion

Regular testing ensures:
- Cluster stability
- Security compliance
- Performance requirements
- High availability
- Disaster recovery readiness

Run these tests periodically and after any configuration changes.
