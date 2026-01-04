# Helm Chart Testing and Validation Guide

## Prerequisites

- Helm 3.8+
- kubectl access to a Kubernetes cluster
- [helm-unittest](https://github.com/helm-unittest/helm-unittest) plugin (optional)

## Quick Test

```bash
# Run Helm tests
helm test kafeventstore --namespace kafeventstore

# Check test results
kubectl get pods -n kafeventstore | grep test
kubectl logs -n kafeventstore kafeventstore-test-connection
```

## Validation

### 1. Lint the Chart

```bash
# Basic lint
helm lint ./helm/kafeventstore

# Lint with values file
helm lint ./helm/kafeventstore -f ./helm/kafeventstore/envs/values-local.yaml

# Lint for all environments
for env in local aws azure gcp; do
  echo "Linting for $env..."
  helm lint ./helm/kafeventstore -f ./helm/kafeventstore/envs/values-${env}.yaml
done
```

### 2. Template Validation

```bash
# Generate templates
helm template kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml \
  --debug

# Validate generated manifests
helm template kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml | \
  kubectl apply --dry-run=client -f -

# Check for specific resources
helm template kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-aws.yaml | \
  grep -A 10 "kind: Deployment"
```

### 3. Dry Run Installation

```bash
# Dry run
helm install kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml \
  --namespace kafeventstore \
  --create-namespace \
  --dry-run --debug
```

### 4. Install and Test

```bash
# Install
helm install kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml \
  --namespace kafeventstore \
  --create-namespace \
  --wait --timeout 5m

# Run tests
helm test kafeventstore --namespace kafeventstore

# Check deployment
kubectl get all -n kafeventstore
```

## Unit Testing with helm-unittest

### Install Plugin

```bash
helm plugin install https://github.com/helm-unittest/helm-unittest.git
```

### Create Test File

```yaml
# tests/deployment_test.yaml
suite: test deployment
templates:
  - deployment.yaml
tests:
  - it: should create deployment
    asserts:
      - isKind:
          of: Deployment
      - equal:
          path: metadata.name
          value: RELEASE-NAME-kafeventstore
  
  - it: should set correct labels
    asserts:
      - isSubset:
          path: metadata.labels
          content:
            app.kubernetes.io/name: kafeventstore
  
  - it: should use correct image
    set:
      image.repository: custom-repo
      image.tag: 1.0.0
    asserts:
      - equal:
          path: spec.template.spec.containers[0].image
          value: custom-repo:1.0.0
```

### Run Unit Tests

```bash
helm unittest ./helm/kafeventstore
```

## End-to-End Testing

### 1. Deploy Test Environment

```bash
# Create test namespace
kubectl create namespace kafeventstore-test

# Deploy
helm install kafeventstore-test ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml \
  --namespace kafeventstore-test
```

### 2. Produce Test Events

```bash
# Port forward to Kafka
kubectl port-forward -n kafka svc/kafka-cluster-kafka-bootstrap 9092:9092

# Produce test event
cat <<EOF | kafkacat -P -b localhost:9092 -t test-events
{
  "specversion": "1.0",
  "type": "com.example.event",
  "source": "/test",
  "id": "test-001",
  "time": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "datacontenttype": "application/json",
  "data": {
    "message": "Test event for kafeventstore"
  }
}
EOF
```

### 3. Verify Storage

```bash
# Check pod logs
kubectl logs -f -n kafeventstore-test deployment/kafeventstore-test

# For file storage - check PVC
kubectl exec -it -n kafeventstore-test deployment/kafeventstore-test -- \
  ls -la /data/events/

# For S3 - check bucket
kubectl exec -it -n kafeventstore-test deployment/kafeventstore-test -- \
  aws s3 ls s3://your-bucket/events/
```

### 4. Check Metrics

```bash
# Port forward to metrics endpoint
kubectl port-forward -n kafeventstore-test svc/kafeventstore-test 9090:9090

# Get metrics
curl http://localhost:9090/metrics | grep kafka_consumer
curl http://localhost:9090/metrics | grep storage_operations
```

### 5. Cleanup Test Environment

```bash
helm uninstall kafeventstore-test --namespace kafeventstore-test
kubectl delete namespace kafeventstore-test
```

## Performance Testing

### Load Test with k6

```javascript
// load-test.js
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 100 }, // ramp up
    { duration: '5m', target: 100 }, // stay at 100
    { duration: '2m', target: 0 },   // ramp down
  ],
};

export default function () {
  let res = http.get('http://kafeventstore:8080/health');
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
}
```

Run test:
```bash
k6 run load-test.js
```

## Upgrade Testing

```bash
# Install version 1.0.0
helm install kafeventstore ./helm/kafeventstore \
  --namespace kafeventstore \
  --set image.tag=1.0.0

# Upgrade to 1.1.0
helm upgrade kafeventstore ./helm/kafeventstore \
  --namespace kafeventstore \
  --set image.tag=1.1.0

# Rollback if needed
helm rollback kafeventstore --namespace kafeventstore
```

## Security Testing

### Pod Security

```bash
# Check pod security context
kubectl get pod -n kafeventstore -o jsonpath='{.items[0].spec.securityContext}'

# Check container security context
kubectl get pod -n kafeventstore -o jsonpath='{.items[0].spec.containers[0].securityContext}'
```

### Network Policy Testing

```bash
# Install test pod in different namespace
kubectl run test-pod --image=nicolaka/netshoot -n default --rm -it -- /bin/bash

# Try to connect to kafeventstore
curl kafeventstore.kafeventstore:8080/health
```

## Checklist

- [ ] Helm lint passes
- [ ] Template generation successful
- [ ] Dry run installation works
- [ ] Actual installation succeeds
- [ ] Helm tests pass
- [ ] Pods are running
- [ ] Health checks pass
- [ ] Metrics accessible
- [ ] Can consume from Kafka
- [ ] Can write to storage
- [ ] Logs show no errors
- [ ] Resource limits respected
- [ ] Security context applied
- [ ] Network policies work
- [ ] Service account configured correctly
- [ ] Secrets mounted properly

## Troubleshooting Test Failures

### Test Pod Fails

```bash
# Check test pod logs
kubectl logs -n kafeventstore kafeventstore-test-connection

# Describe test pod
kubectl describe pod -n kafeventstore kafeventstore-test-connection

# Manual test
kubectl run debug --rm -it --image=busybox:1.36 -n kafeventstore -- sh
wget --spider http://kafeventstore:8080/health
```

### Deployment Issues

```bash
# Check deployment events
kubectl describe deployment kafeventstore -n kafeventstore

# Check pod events
kubectl describe pod -n kafeventstore -l app.kubernetes.io/name=kafeventstore

# Check replica set
kubectl get rs -n kafeventstore
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Helm Chart Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Helm
        uses: azure/setup-helm@v3
        with:
          version: '3.12.0'
      
      - name: Lint chart
        run: helm lint ./helm/kafeventstore
      
      - name: Template chart
        run: |
          for env in local aws azure gcp; do
            helm template kafeventstore ./helm/kafeventstore \
              -f ./helm/kafeventstore/envs/values-${env}.yaml
          done
      
      - name: Install unittest plugin
        run: helm plugin install https://github.com/helm-unittest/helm-unittest.git
      
      - name: Run unit tests
        run: helm unittest ./helm/kafeventstore
```

---

For more information, see [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)
