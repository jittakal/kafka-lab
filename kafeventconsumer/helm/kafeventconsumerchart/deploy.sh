#!/bin/bash

# Deployment script for kafeventconsumer
set -e

NAMESPACE=${NAMESPACE:-kafka-lab}
RELEASE_NAME=${RELEASE_NAME:-kafeventconsumer}
CHART_DIR="$(dirname "$0")"

echo "Deploying kafeventconsumer to namespace: $NAMESPACE"

# Check if namespace exists
if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
    echo "Creating namespace $NAMESPACE"
    kubectl create namespace "$NAMESPACE"
fi

# Install or upgrade the Helm chart
helm upgrade --install "$RELEASE_NAME" "$CHART_DIR" \
    --namespace "$NAMESPACE" \
    --wait \
    --timeout 5m

echo "Deployment complete!"
echo "Check status: kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=kafeventconsumer"
echo "View logs: kubectl logs -n $NAMESPACE -l app.kubernetes.io/name=kafeventconsumer -f"
echo "Port forward metrics: kubectl port-forward -n $NAMESPACE svc/$RELEASE_NAME 9090:9090"
