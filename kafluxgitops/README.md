# Kafka Lab - Flux GitOps Repository

This repository contains the Flux v2 GitOps configuration for deploying Kafka infrastructure and applications.

## Structure

Based on the official [flux2-kustomize-helm-example](https://github.com/fluxcd/flux2-kustomize-helm-example)
```
├── apps
│   ├── base
│   └── local-rancher
├── infrastructure
│   ├── sources
│   ├── configs
│   └── controllers
└── clusters
    └── local-rancher
        └── flux-system
```

## Repository Structure
```
./
├── apps/
│   ├── base/                      # Base application configurations
│   │   ├── kafeventconsumer/
│   │   └── kafeventproducer/
│   └── local-rancher/             # Local environment overrides
├── infrastructure/
│   ├── sources/                   # HelmRepository sources
│   ├── configs/                   # Common configs (namespaces, etc.)
│   └── controllers/               # Infrastructure controllers (Kafka)
└── clusters/
    └── local-rancher/             # Cluster-specific Flux configs
        ├── flux-system/
        ├── infrastructure.yaml
        └── apps.yaml
```

## Components

### Infrastructure
- **Namespace**: kafka-lab
- **Kafka Cluster**: kafka-kraft with KRaft mode and SASL/SSL

### Applications
- **kafeventconsumer**: Kafka event consumer
- **kafeventproducer**: Kafka event producer

## Prerequisites

- Kubernetes cluster (Rancher Desktop)
- Flux CLI v2.7.2+
- kubectl configured
- GitHub personal access token with repo permissions

## Bootstrap
```bash
export GITHUB_TOKEN=<your-token>
export GITHUB_USER=jittakal
export GITHUB_REPO=kafka-lab

flux bootstrap github \
  --owner=${GITHUB_USER} \
  --repository=${GITHUB_REPO} \
  --branch=main \
  --path=kafluxgitops/clusters/local-rancher \
  --personal
```

## Verify Installation
```bash
# Check Flux components
flux check

# Watch reconciliation
flux get kustomizations --watch

# Check HelmReleases
flux get helmreleases -A
```

## Deployment Order

1. **Infrastructure Sources** - HelmRepository definitions
2. **Infrastructure Configs** - Namespace creation
3. **Infrastructure Controllers** - Kafka cluster
4. **Applications** - Consumer and Producer apps (depends on Kafka)

## Making Changes

All changes are made through Git commits:
```bash
# Edit configuration
vi apps/local-rancher/kafeventconsumer-values.yaml

# Commit and push
git add .
git commit -m "Update consumer configuration"
git push

# Flux will automatically reconcile (or trigger manually)
flux reconcile kustomization apps --with-source
```