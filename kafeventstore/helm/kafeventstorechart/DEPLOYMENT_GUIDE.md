# Kafeventstore Deployment Guide

Comprehensive guide for deploying kafeventstore on different platforms with best practices.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Local Kubernetes Deployment](#local-kubernetes-deployment)
3. [AWS EKS Deployment](#aws-eks-deployment)
4. [Azure AKS Deployment](#azure-aks-deployment)
5. [GCP GKE Deployment](#gcp-gke-deployment)
6. [Using envsubst](#using-envsubst)
7. [Monitoring and Observability](#monitoring-and-observability)
8. [Security Best Practices](#security-best-practices)
9. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### General Requirements

- Kubernetes cluster 1.21+
- Helm 3.8+
- kubectl configured to access your cluster
- Container registry access (for custom images)

### Optional Tools

- [envsubst](https://github.com/a8m/envsubst) - For environment variable substitution
- [helm-diff](https://github.com/databus23/helm-diff) - For comparing releases

Install envsubst:
```bash
# Using Go
go install github.com/a8m/envsubst/cmd/envsubst@latest

# Using Homebrew (macOS)
brew install envsubst

# Using apt (Ubuntu/Debian)
apt-get install gettext-base
```

---

## Local Kubernetes Deployment

Best for development and testing.

### 1. Setup Local Kubernetes

Choose one:

#### Option A: Minikube
```bash
# Install minikube
brew install minikube  # macOS
# or download from https://minikube.sigs.k8s.io/

# Start cluster
minikube start --cpus=4 --memory=8192

# Enable metrics
minikube addons enable metrics-server
```

#### Option B: kind (Kubernetes in Docker)
```bash
# Install kind
brew install kind  # macOS
# or go install sigs.k8s.io/kind@latest

# Create cluster
cat <<EOF | kind create cluster --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOF
```

#### Option C: k3s (Lightweight Kubernetes)
```bash
# Install k3s
curl -sfL https://get.k3s.io | sh -

# Get kubeconfig
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
```

### 2. Deploy Kafka (Local)

```bash
# Using Strimzi Kafka Operator
kubectl create namespace kafka
kubectl apply -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka

# Wait for operator
kubectl wait deployment/strimzi-cluster-operator -n kafka --for=condition=available --timeout=300s

# Deploy Kafka cluster
cat <<EOF | kubectl apply -f -
apiVersion: kafka.strimzi.io/v1beta2
kind: Kafka
metadata:
  name: kafka-cluster
  namespace: kafka
spec:
  kafka:
    version: 3.6.0
    replicas: 1
    listeners:
      - name: plain
        port: 9092
        type: internal
        tls: false
    storage:
      type: ephemeral
  zookeeper:
    replicas: 1
    storage:
      type: ephemeral
  entityOperator:
    topicOperator: {}
    userOperator: {}
EOF

# Create topic
cat <<EOF | kubectl apply -f -
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaTopic
metadata:
  name: test-events
  namespace: kafka
  labels:
    strimzi.io/cluster: kafka-cluster
spec:
  partitions: 3
  replicas: 1
EOF
```

### 3. Configure Environment

```bash
# Copy example environment file
cp helm/kafeventstore/envs/.env.example helm/kafeventstore/envs/.env.dev

# Edit with your values
cat > helm/kafeventstore/envs/.env.dev <<EOF
KAFKA_BOOTSTRAP_SERVERS=kafka-cluster-kafka-bootstrap.kafka:9092
KAFKA_TOPICS=test-events
KAFKA_SECURITY_PROTOCOL=PLAINTEXT
STORAGE_BACKEND=file
IMAGE_REPOSITORY=kafeventstore
IMAGE_TAG=latest
EOF
```

### 4. Deploy Kafeventstore

Using deployment script with envsubst:
```bash
# Load environment
source helm/kafeventstore/envs/.env.dev

# Deploy
./helm/kafeventstore/deploy.sh dev local
```

Or using Helm directly:
```bash
helm install kafeventstore ./helm/kafeventstore \
  -f ./helm/kafeventstore/envs/values-local.yaml \
  --namespace default \
  --set config.kafka.bootstrapServers=kafka-cluster-kafka-bootstrap.kafka:9092
```

### 5. Verify Deployment

```bash
# Check pods
kubectl get pods

# Check logs
kubectl logs -f deployment/kafeventstore

# Port forward to access metrics
kubectl port-forward svc/kafeventstore 9090:9090 8080:8080

# Access metrics
curl http://localhost:9090/metrics

# Check health
curl http://localhost:8080/health
```

---

## AWS EKS Deployment

Production deployment on AWS using best practices.

### 1. Prerequisites

- AWS CLI configured
- eksctl or Terraform for EKS cluster
- IAM permissions to create roles and policies

### 2. Create EKS Cluster

```bash
# Using eksctl
eksctl create cluster \
  --name kafeventstore-cluster \
  --region us-east-1 \
  --node-type t3.large \
  --nodes 3 \
  --nodes-min 2 \
  --nodes-max 5 \
  --managed \
  --with-oidc

# Enable OIDC provider (if not done above)
eksctl utils associate-iam-oidc-provider \
  --cluster kafeventstore-cluster \
  --region us-east-1 \
  --approve
```

### 3. Setup IAM Roles for Service Accounts (IRSA)

```bash
# Get OIDC provider
OIDC_PROVIDER=$(aws eks describe-cluster \
  --name kafeventstore-cluster \
  --region us-east-1 \
  --query "cluster.identity.oidc.issuer" \
  --output text | sed -e "s/^https:\/\///")

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

# Create IAM policy for S3 access
cat > kafeventstore-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:DeleteObject"
      ],
      "Resource": [
        "arn:aws:s3:::your-events-bucket",
        "arn:aws:s3:::your-events-bucket/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt",
        "kms:GenerateDataKey"
      ],
      "Resource": "arn:aws:kms:us-east-1:${ACCOUNT_ID}:key/your-kms-key-id"
    }
  ]
}
EOF

aws iam create-policy \
  --policy-name KafeventstoreS3Policy \
  --policy-document file://kafeventstore-policy.json

# Create IAM role with trust policy
cat > trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::${ACCOUNT_ID}:oidc-provider/${OIDC_PROVIDER}"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "${OIDC_PROVIDER}:sub": "system:serviceaccount:kafeventstore:kafeventstore",
          "${OIDC_PROVIDER}:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
EOF

aws iam create-role \
  --role-name kafeventstore-role \
  --assume-role-policy-document file://trust-policy.json

aws iam attach-role-policy \
  --role-name kafeventstore-role \
  --policy-arn arn:aws:iam::${ACCOUNT_ID}:policy/KafeventstoreS3Policy
```

### 4. Create S3 Bucket

```bash
# Create bucket
aws s3 mb s3://your-events-bucket --region us-east-1

# Enable versioning (optional)
aws s3api put-bucket-versioning \
  --bucket your-events-bucket \
  --versioning-configuration Status=Enabled

# Enable encryption
aws s3api put-bucket-encryption \
  --bucket your-events-bucket \
  --server-side-encryption-configuration '{
    "Rules": [{
      "ApplyServerSideEncryptionByDefault": {
        "SSEAlgorithm": "aws:kms",
        "KMSMasterKeyID": "your-kms-key-id"
      }
    }]
  }'

# Configure lifecycle policy (optional)
aws s3api put-bucket-lifecycle-configuration \
  --bucket your-events-bucket \
  --lifecycle-configuration file://lifecycle-policy.json
```

### 5. Setup AWS MSK (optional)

```bash
# Create MSK cluster (basic example)
aws kafka create-cluster \
  --cluster-name kafeventstore-kafka \
  --kafka-version 3.6.0 \
  --number-of-broker-nodes 3 \
  --broker-node-group-info file://broker-info.json

# Get bootstrap servers
aws kafka get-bootstrap-brokers \
  --cluster-arn <cluster-arn>
```

### 6. Create Secrets

```bash
# Create namespace
kubectl create namespace kafeventstore

# Create Kafka credentials secret
kubectl create secret generic kafeventstore-kafka-creds \
  --from-literal=username='your-username' \
  --from-literal=password='your-password' \
  -n kafeventstore
```

### 7. Configure Environment

```bash
# Create environment file
cat > helm/kafeventstore/envs/.env.prod <<EOF
AWS_REGION=us-east-1
AWS_ROLE_ARN=arn:aws:iam::${ACCOUNT_ID}:role/kafeventstore-role
KAFKA_BOOTSTRAP_SERVERS=your-kafka-cluster:9092
KAFKA_TOPICS=events
S3_BUCKET=your-events-bucket
IMAGE_REPOSITORY=${ACCOUNT_ID}.dkr.ecr.us-east-1.amazonaws.com/kafeventstore
IMAGE_TAG=1.0.0
EOF
```

### 8. Deploy with envsubst

```bash
# Load environment
source helm/kafeventstore/envs/.env.prod

# Deploy using script
./helm/kafeventstore/deploy.sh prod aws

# Or manually with envsubst
envsubst < helm/kafeventstore/envs/values-aws.yaml | \
  helm install kafeventstore ./helm/kafeventstore \
    -f - \
    --namespace kafeventstore
```

### 9. Verify Deployment

```bash
# Check pods
kubectl get pods -n kafeventstore

# Check service account annotations
kubectl get sa kafeventstore -n kafeventstore -o yaml | grep eks.amazonaws.com

# Check logs
kubectl logs -f deployment/kafeventstore -n kafeventstore

# Test S3 access from pod
kubectl exec -it deployment/kafeventstore -n kafeventstore -- \
  aws s3 ls s3://your-events-bucket/
```

---

## Azure AKS Deployment

### 1. Create AKS Cluster with Workload Identity

```bash
# Set variables
RESOURCE_GROUP=kafeventstore-rg
CLUSTER_NAME=kafeventstore-aks
LOCATION=eastus

# Create resource group
az group create --name $RESOURCE_GROUP --location $LOCATION

# Create AKS cluster with Workload Identity
az aks create \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --node-count 3 \
  --node-vm-size Standard_D4s_v3 \
  --enable-managed-identity \
  --enable-oidc-issuer \
  --enable-workload-identity \
  --generate-ssh-keys

# Get credentials
az aks get-credentials \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME
```

### 2. Setup Azure Storage

```bash
# Create storage account
STORAGE_ACCOUNT=kafeventstorestorage

az storage account create \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --sku Standard_GRS \
  --encryption-services blob \
  --https-only true \
  --min-tls-version TLS1_2

# Create container
az storage container create \
  --name events \
  --account-name $STORAGE_ACCOUNT \
  --auth-mode login
```

### 3. Setup Workload Identity

```bash
# Get OIDC issuer
OIDC_ISSUER=$(az aks show \
  --name $CLUSTER_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "oidcIssuerProfile.issuerUrl" \
  --output tsv)

# Create managed identity
IDENTITY_NAME=kafeventstore-identity

az identity create \
  --resource-group $RESOURCE_GROUP \
  --name $IDENTITY_NAME

# Get identity client ID
CLIENT_ID=$(az identity show \
  --resource-group $RESOURCE_GROUP \
  --name $IDENTITY_NAME \
  --query clientId \
  --output tsv)

# Grant storage permissions
STORAGE_ID=$(az storage account show \
  --name $STORAGE_ACCOUNT \
  --resource-group $RESOURCE_GROUP \
  --query id \
  --output tsv)

az role assignment create \
  --role "Storage Blob Data Contributor" \
  --assignee $CLIENT_ID \
  --scope $STORAGE_ID

# Create federated credential
kubectl create namespace kafeventstore

az identity federated-credential create \
  --name kafeventstore-federated-credential \
  --identity-name $IDENTITY_NAME \
  --resource-group $RESOURCE_GROUP \
  --issuer $OIDC_ISSUER \
  --subject system:serviceaccount:kafeventstore:kafeventstore
```

### 4. Deploy

```bash
# Configure environment
cat > helm/kafeventstore/envs/.env.prod <<EOF
AZURE_CLIENT_ID=$CLIENT_ID
AZURE_TENANT_ID=$(az account show --query tenantId --output tsv)
AZURE_STORAGE_ACCOUNT=$STORAGE_ACCOUNT
KAFKA_BOOTSTRAP_SERVERS=your-kafka:9092
IMAGE_REPOSITORY=youracr.azurecr.io/kafeventstore
IMAGE_TAG=1.0.0
EOF

# Deploy
source helm/kafeventstore/envs/.env.prod
./helm/kafeventstore/deploy.sh prod azure
```

---

## GCP GKE Deployment

### 1. Create GKE Cluster

```bash
# Set variables
PROJECT_ID=your-project-id
CLUSTER_NAME=kafeventstore-cluster
REGION=us-central1

# Enable APIs
gcloud services enable container.googleapis.com
gcloud services enable iam.googleapis.com
gcloud services enable storage.googleapis.com

# Create cluster with Workload Identity
gcloud container clusters create $CLUSTER_NAME \
  --region $REGION \
  --num-nodes 1 \
  --machine-type n1-standard-4 \
  --workload-pool=$PROJECT_ID.svc.id.goog \
  --enable-autoscaling \
  --min-nodes 1 \
  --max-nodes 5

# Get credentials
gcloud container clusters get-credentials $CLUSTER_NAME --region $REGION
```

### 2. Setup GCS Bucket

```bash
# Create bucket
BUCKET_NAME=${PROJECT_ID}-events

gsutil mb -p $PROJECT_ID -l $REGION gs://$BUCKET_NAME/

# Enable versioning
gsutil versioning set on gs://$BUCKET_NAME/

# Set lifecycle policy
cat > lifecycle.json <<EOF
{
  "lifecycle": {
    "rule": [
      {
        "action": {"type": "Delete"},
        "condition": {"age": 365}
      }
    ]
  }
}
EOF

gsutil lifecycle set lifecycle.json gs://$BUCKET_NAME/
```

### 3. Setup Workload Identity

```bash
# Create GCP service account
GSA_NAME=kafeventstore

gcloud iam service-accounts create $GSA_NAME \
  --project=$PROJECT_ID

# Grant storage permissions
gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member="serviceAccount:${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role="roles/storage.objectAdmin"

# Create namespace
kubectl create namespace kafeventstore

# Bind K8s SA to GCP SA
gcloud iam service-accounts add-iam-policy-binding \
  ${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:${PROJECT_ID}.svc.id.goog[kafeventstore/kafeventstore]"
```

### 4. Deploy

```bash
# Configure environment
cat > helm/kafeventstore/envs/.env.prod <<EOF
GCP_PROJECT_ID=$PROJECT_ID
GCP_SERVICE_ACCOUNT=${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com
GCS_BUCKET=$BUCKET_NAME
KAFKA_BOOTSTRAP_SERVERS=your-kafka:9092
IMAGE_REPOSITORY=gcr.io/$PROJECT_ID/kafeventstore
IMAGE_TAG=1.0.0
EOF

# Deploy
source helm/kafeventstore/envs/.env.prod
./helm/kafeventstore/deploy.sh prod gcp
```

---

## Using envsubst

### Basic Usage

```bash
# Install envsubst
go install github.com/a8m/envsubst/cmd/envsubst@latest

# Set environment variables
export KAFKA_BOOTSTRAP_SERVERS="kafka:9092"
export S3_BUCKET="my-events-bucket"

# Process values file
envsubst < values-template.yaml > values-processed.yaml

# Deploy with processed values
helm install kafeventstore ./helm/kafeventstore -f values-processed.yaml
```

### Advanced: Template Values File

Create a template with environment variables:

```yaml
# values-template.yaml
config:
  kafka:
    bootstrapServers: "${KAFKA_BOOTSTRAP_SERVERS}"
    topics:
      - "${KAFKA_TOPIC:-events}"
  
  storage:
    backend: "${STORAGE_BACKEND}"
    s3:
      bucket: "${S3_BUCKET}"
      region: "${AWS_REGION:-us-east-1}"

image:
  repository: "${IMAGE_REPOSITORY}"
  tag: "${IMAGE_TAG:-latest}"
```

Use with deployment script:

```bash
# The deploy.sh script automatically uses envsubst if available
./helm/kafeventstore/deploy.sh prod aws
```

---

## Monitoring and Observability

### Prometheus Integration

```bash
# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# Deploy kafeventstore with ServiceMonitor
helm install kafeventstore ./helm/kafeventstore \
  -f values-aws.yaml \
  --set serviceMonitor.enabled=true \
  --set serviceMonitor.namespace=monitoring
```

### Grafana Dashboards

Access Grafana:
```bash
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80
```

Import dashboards for:
- Kafka consumer metrics
- Storage operation metrics
- Application performance

### Logging

#### Fluent Bit Integration

```bash
# Add Fluent Bit as sidecar
helm install kafeventstore ./helm/kafeventstore \
  --set sidecars[0].name=fluent-bit \
  --set sidecars[0].image=fluent/fluent-bit:2.0 \
  --set sidecars[0].volumeMounts[0].name=logs \
  --set sidecars[0].volumeMounts[0].mountPath=/logs
```

---

## Security Best Practices

### Pod Security

```yaml
# Use Pod Security Standards
apiVersion: v1
kind: Namespace
metadata:
  name: kafeventstore
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### Network Policies

Enable network policies in values:
```yaml
networkPolicy:
  enabled: true
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            name: monitoring
  egress:
    - to:
      - namespaceSelector: {}
      ports:
      - protocol: TCP
        port: 9092  # Kafka
```

### Secrets Management

#### AWS Secrets Manager
```bash
# Install External Secrets Operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets \
  --namespace external-secrets-system \
  --create-namespace
```

#### Azure Key Vault
```bash
# Install CSI Secret Store Driver
helm repo add csi-secrets-store-provider-azure https://azure.github.io/secrets-store-csi-driver-provider-azure/charts
helm install csi csi-secrets-store-provider-azure/csi-secrets-store-provider-azure
```

---

## Troubleshooting

### Common Issues

#### Pod Not Starting

```bash
# Check events
kubectl describe pod <pod-name> -n kafeventstore

# Check logs
kubectl logs <pod-name> -n kafeventstore

# Check previous logs if crashed
kubectl logs <pod-name> -n kafeventstore --previous
```

#### Storage Access Issues

```bash
# Verify service account annotations
kubectl get sa kafeventstore -n kafeventstore -o yaml

# Test from pod
kubectl exec -it <pod-name> -n kafeventstore -- /bin/sh

# For AWS
aws s3 ls s3://your-bucket/

# For Azure
az storage blob list --container events --account-name storage

# For GCP
gsutil ls gs://your-bucket/
```

#### Kafka Connection Issues

```bash
# Test Kafka connectivity
kubectl run kafka-test --rm -i --tty \
  --image confluentinc/cp-kafka:latest \
  --namespace kafeventstore \
  --command -- bash

# Inside pod
kafka-console-consumer \
  --bootstrap-server kafka:9092 \
  --topic events \
  --from-beginning
```

### Debug Mode

Enable debug logging:
```bash
helm upgrade kafeventstore ./helm/kafeventstore \
  --set config.logLevel=debug \
  --namespace kafeventstore
```

### Health Checks

```bash
# Port forward to pod
kubectl port-forward -n kafeventstore svc/kafeventstore 8080:8080 9090:9090

# Check health
curl http://localhost:8080/health

# Check metrics
curl http://localhost:9090/metrics
```

---

## Cleanup

### Remove Deployment

```bash
# Uninstall Helm release
helm uninstall kafeventstore --namespace kafeventstore

# Delete namespace
kubectl delete namespace kafeventstore
```

### AWS Cleanup

```bash
# Delete S3 bucket
aws s3 rb s3://your-events-bucket --force

# Delete IAM resources
aws iam detach-role-policy --role-name kafeventstore-role \
  --policy-arn arn:aws:iam::${ACCOUNT_ID}:policy/KafeventstoreS3Policy
aws iam delete-role --role-name kafeventstore-role
aws iam delete-policy --policy-arn arn:aws:iam::${ACCOUNT_ID}:policy/KafeventstoreS3Policy

# Delete EKS cluster
eksctl delete cluster --name kafeventstore-cluster --region us-east-1
```

### Azure Cleanup

```bash
# Delete resource group (includes all resources)
az group delete --name kafeventstore-rg --yes --no-wait
```

### GCP Cleanup

```bash
# Delete GCS bucket
gsutil rm -r gs://$BUCKET_NAME

# Delete service account
gcloud iam service-accounts delete ${GSA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com

# Delete GKE cluster
gcloud container clusters delete $CLUSTER_NAME --region $REGION --quiet
```

---

For more information, see the [main README](README.md) or visit https://github.com/jittakal/kafeventstore
