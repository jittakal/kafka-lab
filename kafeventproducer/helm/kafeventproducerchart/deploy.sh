#!/bin/bash

# Deployment script for kafeventproducer Helm chart
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CHART_DIR="$SCRIPT_DIR"
NAMESPACE="${NAMESPACE:-kafka-lab}"
RELEASE_NAME="${RELEASE_NAME:-producer}"

echo "╔════════════════════════════════════════════╗"
echo "║   kafeventproducer Helm Chart Deployer   ║"
echo "╚════════════════════════════════════════════╝"
echo ""

# Check prerequisites
echo "🔍 Checking prerequisites..."

if ! command -v helm &> /dev/null; then
    echo "❌ Helm is not installed. Please install Helm 3.8+"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl is not installed"
    exit 1
fi

echo "✅ Prerequisites check passed"
echo ""

# Validate Helm chart
echo "🔍 Validating Helm chart..."
helm lint "$CHART_DIR"
echo "✅ Chart validation passed"
echo ""

# Check if namespace exists
if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
    echo "📦 Creating namespace: $NAMESPACE"
    kubectl create namespace "$NAMESPACE"
fi

# Check if release exists
if helm list -n "$NAMESPACE" | grep -q "^$RELEASE_NAME"; then
    echo "🔄 Upgrading existing release: $RELEASE_NAME"
    helm upgrade "$RELEASE_NAME" "$CHART_DIR" \
        --namespace "$NAMESPACE" \
        --wait \
        --timeout 5m
else
    echo "🚀 Installing new release: $RELEASE_NAME"
    helm install "$RELEASE_NAME" "$CHART_DIR" \
        --namespace "$NAMESPACE" \
        --create-namespace \
        --wait \
        --timeout 5m
fi

echo ""
echo "✅ Deployment complete!"
echo ""

# Show deployment status
echo "📊 Deployment Status:"
echo "═══════════════════════"
kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/name=kafeventproducer"
echo ""

echo "📝 Useful Commands:"
echo "═══════════════════════"
echo "View logs:"
echo "  kubectl logs -f -l app.kubernetes.io/name=kafeventproducer -n $NAMESPACE"
echo ""
echo "Port-forward metrics:"
echo "  kubectl port-forward -n $NAMESPACE svc/$RELEASE_NAME 9090:9090"
echo ""
echo "Uninstall:"
echo "  helm uninstall $RELEASE_NAME -n $NAMESPACE"
echo ""
