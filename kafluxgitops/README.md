# Kafka Lab - Flux GitOps Repository

This repository contains the Flux v2 GitOps configuration for deploying Kafka infrastructure and applications.

## Structure

Based on the official [flux2-kustomize-helm-example](https://github.com/fluxcd/flux2-kustomize-helm-example)
```
├── apps
│   ├── base
│   └── local-rancher
├── infrastructure
│   ├── configs
│   ├── controllers
│   └── platforms
└── clusters
    └── local-rancher
        └── flux-system
```

## Repository Structure
```
./
├── apps/
│   ├── base/                      # Base application configurations
│   │   ├── kafeventconsumer/      # Self-contained app with OCIRepository
│   │   │   ├── namespace.yaml
│   │   │   ├── repository.yaml    # OCIRepository definition
│   │   │   ├── release.yaml       # HelmRelease with inline values
│   │   │   └── kustomization.yaml
│   │   └── kafeventproducer/      # Self-contained app with OCIRepository
│   │       ├── namespace.yaml
│   │       ├── repository.yaml
│   │       ├── release.yaml
│   │       └── kustomization.yaml
│   └── local-rancher/             # Environment-specific overlays
│       ├── kustomization.yaml     # Kustomize patches
│       ├── kafeventconsumer-values.yaml
│       └── kafeventproducer-values.yaml
├── infrastructure/
│   ├── configs/                   # Common configurations
│   │   └── kustomization.yaml
│   ├── controllers/               # Infrastructure controllers
│   │   ├── cert-manager.yaml      # OCIRepository reference
│   │   └── kustomization.yaml
│   └── platforms/                 # Platform services (Kafka)
│       ├── kafka/
│       │   ├── namespace.yaml
│       │   ├── repository.yaml    # OCIRepository for Kafka
│       │   ├── release.yaml       # HelmRelease with chartRef
│       │   └── kustomization.yaml
│       └── kustomization.yaml
└── clusters/
    └── local-rancher/             # Cluster-specific Flux configs
        ├── flux-system/
        ├── infrastructure.yaml    # Orchestrates configs → controllers → platforms
        └── apps.yaml              # Depends on platforms
```

## Components

### Infrastructure
- **Namespaces**: 
  - `kafka` - Kafka platform service
  - `kafka-consumers` - Consumer applications
  - `kafka-producers` - Producer applications
- **Platform**: Kafka KRaft cluster with SASL/SSL (deployed via OCIRepository)

### Applications
- **kafeventconsumer**: Kafka event consumer (self-contained with OCIRepository)
- **kafeventproducer**: Kafka event producer (self-contained with OCIRepository)

## Key Patterns

### OCIRepository Pattern
All Helm charts are deployed using OCIRepository (not HelmRepository):
```yaml
apiVersion: source.toolkit.fluxcd.io/v1
kind: OCIRepository
metadata:
  name: kafeventconsumer
  namespace: kafka-consumers
spec:
  interval: 24h
  url: oci://registry-1.docker.io/jittakal/kafeventconsumer
  layerSelector:
    mediaType: "application/vnd.cncf.helm.chart.content.v1.tar+gzip"
    operation: copy
  ref:
    semver: "1.x"
```

### HelmRelease with chartRef
Using chartRef pattern for cleaner references:
```yaml
apiVersion: helm.toolkit.fluxcd.io/v2
kind: HelmRelease
metadata:
  name: kafeventconsumer
  namespace: kafka-consumers
spec:
  interval: 10m
  chartRef:
    kind: OCIRepository
    name: kafeventconsumer
  values:
    replicaCount: 0
```

### Kustomize Patches for Overrides
Environment-specific overrides using Kustomize patches (not ConfigMaps):
```yaml
# apps/local-rancher/kustomization.yaml
patches:
  - path: kafeventconsumer-values.yaml
    target:
      kind: HelmRelease
      name: kafeventconsumer
```

## Prerequisites

- Kubernetes cluster (Rancher Desktop)
- Flux CLI v2.7.2+
- kubectl configured
- GitHub personal access token with repo permissions

### GitHub fine-grained PAT

Bootstrap can be run with a GitHub fine-grained personal access token, for repositories that are created ahead of time.

The fine-grained PAT must be generated with the following permissions:

- Administration -> Access: Read-only
- Contents -> Access: Read and write
- Metadata -> Access: Read-only

Note: that Administration should be set to Access: Read and write when using bootstrap github --token-auth=false. Have used Read and write

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

1. **Infrastructure Configs** - Common configurations
2. **Infrastructure Controllers** - Infrastructure controllers (cert-manager)
3. **Infrastructure Platforms** - Platform services (Kafka KRaft cluster)
4. **Applications** - Consumer and Producer apps (depends on platforms)

Orchestrated via `clusters/local-rancher/infrastructure.yaml`:
```yaml
configs → controllers → platforms
```

And `clusters/local-rancher/apps.yaml`:
```yaml
apps (depends on platforms)
```

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