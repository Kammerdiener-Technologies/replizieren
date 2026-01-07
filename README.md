# Replizieren

[![Test](https://github.com/Kammerdiener-Technologies/replizieren/actions/workflows/test.yml/badge.svg)](https://github.com/Kammerdiener-Technologies/replizieren/actions/workflows/test.yml)
[![Lint](https://github.com/Kammerdiener-Technologies/replizieren/actions/workflows/lint.yml/badge.svg)](https://github.com/Kammerdiener-Technologies/replizieren/actions/workflows/lint.yml)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

**Replizieren** (German for "replicate") is a Kubernetes operator that automatically replicates Secrets and ConfigMaps across namespaces. It also supports triggering rolling restarts of Deployments when the replicated resources change.

> **Documentation:** [https://replizieren.dev](https://replizieren.dev)

## Features

- **Secret Replication**: Automatically copy Secrets to one or more target namespaces
- **ConfigMap Replication**: Automatically copy ConfigMaps to one or more target namespaces
- **Flexible Targeting**: Replicate to specific namespaces, multiple namespaces, or all namespaces
- **Rollout Triggers**: Optionally restart Deployments when Secrets/ConfigMaps are updated
- **Lightweight**: Single controller handles both Secrets and ConfigMaps

## Quick Start

### Installation

```bash
# Using the pre-built image from GitHub Container Registry
kubectl apply -f https://raw.githubusercontent.com/Kammerdiener-Technologies/replizieren/main/dist/install.yaml
```

Or deploy with a specific version:

```bash
# Deploy using kustomize
make deploy IMG=ghcr.io/kammerdiener-technologies/replizieren:v1.0.0
```

### Basic Usage

Add the `replizieren.dev/replicate` annotation to any Secret or ConfigMap:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: source-namespace
  annotations:
    replizieren.dev/replicate: "target-namespace"
type: Opaque
data:
  password: cGFzc3dvcmQxMjM=
```

The Secret will be automatically replicated to `target-namespace`.

## Annotations

| Annotation | Values | Description |
|------------|--------|-------------|
| `replizieren.dev/replicate` | `"namespace"` | Replicate to a single namespace |
| `replizieren.dev/replicate` | `"ns1,ns2,ns3"` | Replicate to multiple namespaces |
| `replizieren.dev/replicate` | `"true"` | Replicate to all namespaces |
| `replizieren.dev/replicate` | `"false"` or empty | Disable replication |
| `replizieren.dev/rollout-on-update` | `"true"` | Restart Deployments using this resource when it changes |

## Examples

### Replicate to Multiple Namespaces

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
  annotations:
    replizieren.dev/replicate: "staging, production, testing"
data:
  app.conf: |
    setting=value
```

### Replicate to All Namespaces

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: registry-credentials
  namespace: default
  annotations:
    replizieren.dev/replicate: "true"
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: ...
```

### Trigger Deployment Rollout on Update

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
  annotations:
    replizieren.dev/replicate: "production"
    replizieren.dev/rollout-on-update: "true"
data:
  config.yaml: |
    database_url: postgres://...
```

When `app-config` is updated, any Deployment in `production` (or `default`) that uses this ConfigMap will be restarted.

## How It Works

1. **Watch**: The operator watches for changes to Secrets and ConfigMaps across all namespaces
2. **Parse**: When a resource changes, it reads the `replizieren.dev/replicate` annotation
3. **Replicate**: Creates or updates copies in the target namespaces
4. **Rollout** (optional): If `rollout-on-update` is enabled, patches Deployments with a timestamp annotation to trigger a rolling restart

### Deployment Detection

The operator detects Deployments using a Secret or ConfigMap by checking:
- **Volume mounts**: `spec.template.spec.volumes[].secret` or `spec.template.spec.volumes[].configMap`
- **Environment variables**: `spec.template.spec.containers[].envFrom[].secretRef` or `spec.template.spec.containers[].envFrom[].configMapRef`

## Installation Options

### From GitHub Container Registry (Recommended)

```bash
kubectl apply -f https://raw.githubusercontent.com/Kammerdiener-Technologies/replizieren/main/dist/install.yaml
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/Kammerdiener-Technologies/replizieren.git
cd replizieren

# Build and push to your registry
make docker-build docker-push IMG=your-registry/replizieren:latest

# Deploy to cluster
make deploy IMG=your-registry/replizieren:latest
```

### Uninstall

```bash
make undeploy
```

## Configuration

The operator runs with minimal configuration. It uses:
- Leader election for high availability
- Health probes on port 8081
- Restricted Pod Security Standards

### Resource Requirements

Default resource limits:
```yaml
resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 10m
    memory: 64Mi
```

## Development

### Prerequisites

- Go 1.24+
- Docker 17.03+
- kubectl v1.11.3+
- Access to a Kubernetes cluster

### Running Locally

```bash
# Install CRDs (if any)
make install

# Run the controller locally
make run
```

### Running Tests

```bash
# Unit tests
make test

# E2E tests (requires cluster)
make test-e2e

# Linting
make lint
```

### Building

```bash
# Build binary
make build

# Build container image
make docker-build IMG=your-registry/replizieren:latest

# Build multi-arch image
make docker-buildx IMG=your-registry/replizieren:latest
```

## Architecture

```
replizieren/
├── cmd/main.go                 # Entry point
├── internal/controller/
│   ├── secret_controller.go    # Secret replication logic
│   ├── configmapwatcher_controller.go  # ConfigMap replication logic
│   └── replicator.go           # Shared helpers
└── config/
    ├── manager/                # Deployment manifests
    ├── rbac/                   # RBAC configuration
    └── default/                # Kustomize overlays
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
