#!/bin/bash
# Deployment script for kafeventstore using envsubst
# Usage: ./deploy.sh [environment] [cloud-provider]
# Example: ./deploy.sh dev local
#          ./deploy.sh prod aws

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CHART_DIR="$SCRIPT_DIR"
ENVS_DIR="$CHART_DIR/envs"

# Default values
ENVIRONMENT="${1:-dev}"
CLOUD_PROVIDER="${2:-local}"
NAMESPACE="${NAMESPACE:-kafeventstore}"
RELEASE_NAME="${RELEASE_NAME:-kafeventstore}"
DRY_RUN="${DRY_RUN:-false}"

# Functions
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_requirements() {
    print_info "Checking requirements..."
    
    # Check if helm is installed
    if ! command -v helm &> /dev/null; then
        print_error "Helm is not installed. Please install Helm 3.8+"
        exit 1
    fi
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed"
        exit 1
    fi
    
    # Check if envsubst is installed (optional)
    if ! command -v envsubst &> /dev/null; then
        print_warn "envsubst is not installed. Environment variable substitution will be limited."
        print_warn "Install from: https://github.com/a8m/envsubst"
        USE_ENVSUBST=false
    else
        USE_ENVSUBST=true
        print_info "envsubst found: $(which envsubst)"
    fi
    
    # Check Helm version
    HELM_VERSION=$(helm version --short | grep -oE 'v[0-9]+\.[0-9]+' | sed 's/v//')
    REQUIRED_VERSION="3.8"
    if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$HELM_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
        print_error "Helm version $HELM_VERSION is too old. Required: $REQUIRED_VERSION+"
        exit 1
    fi
    
    print_info "All requirements met ✓"
}

load_environment() {
    print_info "Loading environment: $ENVIRONMENT ($CLOUD_PROVIDER)"
    
    # Determine values file
    VALUES_FILE="$ENVS_DIR/values-${CLOUD_PROVIDER}.yaml"
    
    if [ ! -f "$VALUES_FILE" ]; then
        print_error "Values file not found: $VALUES_FILE"
        exit 1
    fi
    
    # Load environment variables from .env file if exists
    ENV_FILE="$ENVS_DIR/.env.${ENVIRONMENT}"
    if [ -f "$ENV_FILE" ]; then
        print_info "Loading environment variables from: $ENV_FILE"
        set -a
        source "$ENV_FILE"
        set +a
    else
        print_warn "No environment file found at: $ENV_FILE"
    fi
    
    # Export values file path
    export VALUES_FILE
}

validate_cloud_config() {
    print_info "Validating cloud provider configuration..."
    
    case "$CLOUD_PROVIDER" in
        aws)
            if [ -z "$AWS_REGION" ]; then
                print_warn "AWS_REGION not set. Using default from values file."
            fi
            if [ -z "$AWS_ROLE_ARN" ] && [ "$ENVIRONMENT" != "dev" ]; then
                print_warn "AWS_ROLE_ARN not set. Make sure IRSA is configured."
            fi
            ;;
        azure)
            if [ -z "$AZURE_CLIENT_ID" ] && [ "$ENVIRONMENT" != "dev" ]; then
                print_warn "AZURE_CLIENT_ID not set. Make sure Workload Identity is configured."
            fi
            ;;
        gcp)
            if [ -z "$GCP_SERVICE_ACCOUNT" ] && [ "$ENVIRONMENT" != "dev" ]; then
                print_warn "GCP_SERVICE_ACCOUNT not set. Make sure Workload Identity is configured."
            fi
            ;;
        local)
            print_info "Local deployment - no cloud validation needed"
            ;;
        *)
            print_error "Unknown cloud provider: $CLOUD_PROVIDER"
            print_error "Supported: aws, azure, gcp, local"
            exit 1
            ;;
    esac
}

create_namespace() {
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        print_info "Namespace '$NAMESPACE' already exists"
    else
        print_info "Creating namespace: $NAMESPACE"
        kubectl create namespace "$NAMESPACE"
    fi
}

process_values() {
    print_info "Processing values file..."
    
    if [ "$USE_ENVSUBST" = true ]; then
        print_info "Using envsubst for environment variable substitution"
        PROCESSED_VALUES=$(mktemp)
        envsubst < "$VALUES_FILE" > "$PROCESSED_VALUES"
        export VALUES_FILE="$PROCESSED_VALUES"
    else
        print_warn "Skipping envsubst - using values file as-is"
    fi
}

deploy() {
    print_info "Deploying kafeventstore..."
    print_info "  Release: $RELEASE_NAME"
    print_info "  Namespace: $NAMESPACE"
    print_info "  Environment: $ENVIRONMENT"
    print_info "  Cloud Provider: $CLOUD_PROVIDER"
    print_info "  Values File: $VALUES_FILE"
    
    # Build helm command
    HELM_CMD="helm upgrade --install $RELEASE_NAME $CHART_DIR"
    HELM_CMD="$HELM_CMD --namespace $NAMESPACE"
    HELM_CMD="$HELM_CMD --values $VALUES_FILE"
    HELM_CMD="$HELM_CMD --set global.environment=$ENVIRONMENT"
    HELM_CMD="$HELM_CMD --set global.cloudProvider=$CLOUD_PROVIDER"
    
    # Add cloud-specific overrides
    case "$CLOUD_PROVIDER" in
        aws)
            [ -n "$AWS_ROLE_ARN" ] && HELM_CMD="$HELM_CMD --set serviceAccount.annotations.eks\.amazonaws\.com/role-arn=$AWS_ROLE_ARN"
            [ -n "$S3_BUCKET" ] && HELM_CMD="$HELM_CMD --set config.storage.s3.bucket=$S3_BUCKET"
            ;;
        azure)
            [ -n "$AZURE_CLIENT_ID" ] && HELM_CMD="$HELM_CMD --set serviceAccount.annotations.azure\.workload\.identity/client-id=$AZURE_CLIENT_ID"
            [ -n "$AZURE_TENANT_ID" ] && HELM_CMD="$HELM_CMD --set serviceAccount.annotations.azure\.workload\.identity/tenant-id=$AZURE_TENANT_ID"
            [ -n "$AZURE_STORAGE_ACCOUNT" ] && HELM_CMD="$HELM_CMD --set config.storage.azure.accountName=$AZURE_STORAGE_ACCOUNT"
            ;;
        gcp)
            [ -n "$GCP_SERVICE_ACCOUNT" ] && HELM_CMD="$HELM_CMD --set serviceAccount.annotations.iam\.gke\.io/gcp-service-account=$GCP_SERVICE_ACCOUNT"
            [ -n "$GCS_BUCKET" ] && HELM_CMD="$HELM_CMD --set config.storage.gcs.bucket=$GCS_BUCKET"
            ;;
    esac
    
    # Add Kafka overrides
    [ -n "$KAFKA_BOOTSTRAP_SERVERS" ] && HELM_CMD="$HELM_CMD --set config.kafka.bootstrapServers=$KAFKA_BOOTSTRAP_SERVERS"
    [ -n "$KAFKA_TOPICS" ] && HELM_CMD="$HELM_CMD --set config.kafka.topics={$KAFKA_TOPICS}"
    
    # Add image overrides
    [ -n "$IMAGE_REPOSITORY" ] && HELM_CMD="$HELM_CMD --set image.repository=$IMAGE_REPOSITORY"
    [ -n "$IMAGE_TAG" ] && HELM_CMD="$HELM_CMD --set image.tag=$IMAGE_TAG"
    
    # Dry run or deploy
    if [ "$DRY_RUN" = "true" ]; then
        print_info "Dry run mode - generating manifests..."
        HELM_CMD="$HELM_CMD --dry-run --debug"
    fi
    
    print_info "Executing: $HELM_CMD"
    eval "$HELM_CMD"
    
    if [ $? -eq 0 ]; then
        if [ "$DRY_RUN" != "true" ]; then
            print_info "Deployment successful ✓"
            print_info ""
            print_info "To check deployment status:"
            print_info "  kubectl get pods -n $NAMESPACE"
            print_info "  kubectl logs -f deployment/$RELEASE_NAME -n $NAMESPACE"
            print_info ""
            print_info "To get service info:"
            print_info "  kubectl get svc -n $NAMESPACE"
        fi
    else
        print_error "Deployment failed ✗"
        exit 1
    fi
}

cleanup() {
    if [ -n "$PROCESSED_VALUES" ] && [ -f "$PROCESSED_VALUES" ]; then
        rm -f "$PROCESSED_VALUES"
    fi
}

# Main execution
trap cleanup EXIT

print_info "Kafeventstore Deployment Script"
print_info "================================"
echo ""

check_requirements
load_environment
validate_cloud_config
create_namespace
process_values
deploy

print_info ""
print_info "Deployment complete! 🎉"
