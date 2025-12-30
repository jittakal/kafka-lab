#!/bin/bash

################################################################################
# Kafka KRaft SASL_SSL Deployment Script
# 
# This script automates the deployment of Apache Kafka 4.x with KRaft mode
# and SASL_SSL security on Kubernetes.
################################################################################

set -e

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
NAMESPACE="kafka-lab"
RELEASE_NAME="kafka"
REPLICA_COUNT=1
CHART_PATH="$SCRIPT_DIR"
INSTALL_CERT_MANAGER=false
DRY_RUN=false
DEBUG=false

# Function to print colored messages
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deploy Apache Kafka 4.x with KRaft mode and SASL_SSL security.

OPTIONS:
    -n, --namespace NAME        Kubernetes namespace (default: kafka-lab)
    -r, --release NAME          Helm release name (default: kafka)
    -c, --replicas COUNT        Number of Kafka replicas (default: 1)
    -p, --chart-path PATH       Path to Helm chart (default: ./kaflocalhelm/kafka-kraft-sasl-ssl)
    --install-cert-manager      Install cert-manager if not present
    --dry-run                   Simulate deployment without applying changes
    --debug                     Enable debug output
    -h, --help                  Show this help message

EXAMPLES:
    # Deploy single-node Kafka
    $0

    # Deploy 3-node Kafka cluster
    $0 --replicas 3

    # Deploy with cert-manager installation
    $0 --install-cert-manager

    # Dry run to see what will be deployed
    $0 --dry-run

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -r|--release)
            RELEASE_NAME="$2"
            shift 2
            ;;
        -c|--replicas)
            REPLICA_COUNT="$2"
            shift 2
            ;;
        -p|--chart-path)
            CHART_PATH="$2"
            shift 2
            ;;
        --install-cert-manager)
            INSTALL_CERT_MANAGER=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --debug)
            DEBUG=true
            shift
            ;;
        -h|--help)
            show_usage
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Enable debug mode if requested
if [ "$DEBUG" = true ]; then
    set -x
fi

print_info "Starting Kafka deployment..."
echo ""
print_info "Configuration:"
echo "  Namespace: $NAMESPACE"
echo "  Release Name: $RELEASE_NAME"
echo "  Replicas: $REPLICA_COUNT"
echo "  Chart Path: $CHART_PATH"
echo "  Dry Run: $DRY_RUN"
echo ""

# Check prerequisites
print_info "Checking prerequisites..."

# Check if kubectl is installed
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed. Please install kubectl first."
    exit 1
fi
print_success "kubectl is installed"

# Check if helm is installed
if ! command -v helm &> /dev/null; then
    print_error "helm is not installed. Please install Helm 3.8+ first."
    exit 1
fi
print_success "helm is installed"

# Check Kubernetes connection
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
    exit 1
fi
print_success "Connected to Kubernetes cluster"

# Check if cert-manager is installed
print_info "Checking for cert-manager..."
if kubectl get namespace cert-manager &> /dev/null; then
    print_success "cert-manager namespace exists"
    
    if kubectl get pods -n cert-manager -l app.kubernetes.io/instance=cert-manager | grep -q "Running"; then
        print_success "cert-manager is running"
    else
        print_warning "cert-manager pods are not running"
        if [ "$INSTALL_CERT_MANAGER" = true ]; then
            print_info "Waiting for cert-manager to be ready..."
            kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=300s
        fi
    fi
else
    print_warning "cert-manager is not installed"
    
    if [ "$INSTALL_CERT_MANAGER" = true ]; then
        print_info "Installing cert-manager..."
        kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.3/cert-manager.yaml
        
        print_info "Waiting for cert-manager to be ready..."
        kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=300s
        print_success "cert-manager installed successfully"
    else
        print_error "cert-manager is required but not installed. Use --install-cert-manager flag to install it."
        exit 1
    fi
fi

# Check if chart exists
if [ ! -f "$CHART_PATH/Chart.yaml" ]; then
    print_error "Chart not found at: $CHART_PATH"
    exit 1
fi
print_success "Chart found at: $CHART_PATH"

# Create namespace if it doesn't exist
if kubectl get namespace $NAMESPACE &> /dev/null; then
    print_info "Namespace '$NAMESPACE' already exists"
else
    print_info "Creating namespace '$NAMESPACE'..."
    if [ "$DRY_RUN" = false ]; then
        kubectl create namespace $NAMESPACE
        print_success "Namespace created"
    else
        print_info "[DRY RUN] Would create namespace: $NAMESPACE"
    fi
fi

# Check if release already exists
if helm list -n $NAMESPACE | grep -q "^$RELEASE_NAME"; then
    print_warning "Release '$RELEASE_NAME' already exists in namespace '$NAMESPACE'"
    print_info "Upgrading existing release..."
    HELM_ACTION="upgrade"
else
    print_info "Installing new release..."
    HELM_ACTION="install"
fi

# Prepare Helm command
HELM_CMD="helm $HELM_ACTION $RELEASE_NAME $CHART_PATH"
HELM_CMD="$HELM_CMD --namespace $NAMESPACE"
HELM_CMD="$HELM_CMD --set replicaCount=$REPLICA_COUNT"

# Add additional flags for multi-node cluster
if [ "$REPLICA_COUNT" -gt 1 ]; then
    HELM_CMD="$HELM_CMD --set kafka.defaultReplicationFactor=$REPLICA_COUNT"
    HELM_CMD="$HELM_CMD --set kafka.offsetsTopicReplicationFactor=$REPLICA_COUNT"
    HELM_CMD="$HELM_CMD --set pdb.enabled=true"
    HELM_CMD="$HELM_CMD --set pdb.minAvailable=1"
fi

# Add dry-run flag if requested
if [ "$DRY_RUN" = true ]; then
    HELM_CMD="$HELM_CMD --dry-run --debug"
fi

# Execute Helm command
print_info "Executing Helm command..."
echo ""
eval $HELM_CMD

if [ "$DRY_RUN" = false ]; then
    echo ""
    print_success "Helm $HELM_ACTION completed successfully!"
    
    # Wait for pods to be ready
    print_info "Waiting for Kafka pods to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=kafka-kraft-sasl-ssl -n $NAMESPACE --timeout=600s
    
    print_success "All Kafka pods are ready!"
    
    # Display status
    echo ""
    print_info "Deployment Status:"
    kubectl get pods -n $NAMESPACE -l app.kubernetes.io/name=kafka-kraft-sasl-ssl
    
    echo ""
    print_info "Services:"
    kubectl get svc -n $NAMESPACE
    
    echo ""
    print_info "Persistent Volume Claims:"
    kubectl get pvc -n $NAMESPACE
    
    echo ""
    print_success "Kafka cluster deployed successfully!"
    echo ""
    print_info "To get started, run:"
    echo "  helm status $RELEASE_NAME -n $NAMESPACE"
    echo ""
    print_info "To test the deployment, run:"
    echo "  helm test $RELEASE_NAME -n $NAMESPACE"
    echo ""
    print_info "To connect to Kafka:"
    echo "  Bootstrap Server: $RELEASE_NAME-hs.$NAMESPACE.svc.cluster.local:9093"
    echo "  Username: admin"
    echo "  Password: admin-secret"
    echo ""
    print_warning "Remember to change the default credentials in production!"
else
    echo ""
    print_info "Dry run completed. No changes were made to the cluster."
fi
