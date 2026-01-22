---
layout: default
title: Installation
---

# Installation

This guide covers different ways to install Replizieren in your Kubernetes cluster.

## Prerequisites

- Kubernetes cluster v1.11.3+
- `kubectl` configured to communicate with your cluster
- Cluster-admin privileges (for RBAC setup)

## Install with Helm (Recommended)

The easiest way to install Replizieren is using Helm:

```bash
helm install replizieren oci://ghcr.io/kammerdiener-technologies/charts/replizieren \
  --version 0.1.0 \
  --namespace replizieren-system \
  --create-namespace
```

### Helm Configuration

You can customize the installation using values:

```bash
helm install replizieren oci://ghcr.io/kammerdiener-technologies/charts/replizieren \
  --version 0.1.0 \
  --namespace replizieren-system \
  --create-namespace \
  --set replicaCount=2 \
  --set resources.limits.memory=256Mi
```

Available values:

| Value | Default | Description |
|-------|---------|-------------|
| `replicaCount` | `1` | Number of replicas (uses leader election) |
| `image.repository` | `ghcr.io/kammerdiener-technologies/replizieren` | Image repository |
| `image.tag` | Chart appVersion | Image tag |
| `resources.limits.cpu` | `500m` | CPU limit |
| `resources.limits.memory` | `128Mi` | Memory limit |
| `resources.requests.cpu` | `10m` | CPU request |
| `resources.requests.memory` | `64Mi` | Memory request |
| `controller.leaderElect` | `true` | Enable leader election |

## Install with kubectl

Install using the manifest from a specific release:

```bash
# Install a specific version (recommended for production)
kubectl apply -f https://github.com/Kammerdiener-Technologies/replizieren/releases/download/v0.1.0/install.yaml
```

Or install the latest development version from main:

```bash
# Install latest (for development/testing)
kubectl apply -f https://raw.githubusercontent.com/Kammerdiener-Technologies/replizieren/main/dist/install.yaml
```

This will:
1. Create the `replizieren-system` namespace
2. Deploy the controller with appropriate RBAC permissions
3. Start watching for Secrets and ConfigMaps with replication annotations

### Verify Installation

```bash
# Check the controller is running
kubectl get pods -n replizieren-system

# Expected output:
# NAME                                      READY   STATUS    RESTARTS   AGE
# replizieren-controller-manager-xxx        1/1     Running   0          30s
```

## Install with Kustomize

For more control over the installation, use kustomize:

```bash
# Using kustomize with a specific version
kubectl apply -k https://github.com/Kammerdiener-Technologies/replizieren/config/default?ref=v0.1.0
```

Or clone and deploy:

```bash
git clone https://github.com/Kammerdiener-Technologies/replizieren.git
cd replizieren
make deploy IMG=ghcr.io/kammerdiener-technologies/replizieren:v0.1.0
```

## Build from Source

If you need to customize the operator or run a development version:

### 1. Clone the Repository

```bash
git clone https://github.com/Kammerdiener-Technologies/replizieren.git
cd replizieren
```

### 2. Build the Image

```bash
# Single architecture
make docker-build IMG=your-registry/replizieren:latest

# Multi-architecture (amd64 + arm64)
make docker-buildx IMG=your-registry/replizieren:latest
```

### 3. Push to Your Registry

```bash
make docker-push IMG=your-registry/replizieren:latest
```

### 4. Deploy

```bash
make deploy IMG=your-registry/replizieren:latest
```

## Configuration Options

### Resource Limits

The default deployment uses conservative resource limits:

```yaml
resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 10m
    memory: 64Mi
```

To customize, edit `config/manager/manager.yaml` before deploying, or patch after deployment:

```bash
kubectl patch deployment replizieren-controller-manager \
  -n replizieren-system \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/resources/limits/memory", "value": "256Mi"}]'
```

### Replicas

For high availability, you can increase replicas. The controller uses leader election, so only one instance is active at a time:

```bash
kubectl scale deployment replizieren-controller-manager \
  -n replizieren-system \
  --replicas=3
```

### Namespace Restriction

By default, Replizieren watches all namespaces. To restrict to specific namespaces, you would need to modify the controller code (feature planned for future releases).

## RBAC Permissions

Replizieren requires the following permissions:

| Resource | Verbs | Purpose |
|----------|-------|---------|
| secrets | get, list, watch, create, update, patch, delete | Replicate secrets |
| configmaps | get, list, watch, create, update, patch, delete | Replicate configmaps |
| namespaces | get, list, watch | Discover target namespaces |
| deployments | get, list, patch | Trigger rollouts |

The full ClusterRole is defined in `config/rbac/role.yaml`.

## Uninstalling

### Using Make

```bash
make undeploy
```

### Manual Uninstall

```bash
# Delete using the same manifest you installed with
kubectl delete -f https://github.com/Kammerdiener-Technologies/replizieren/releases/download/v0.1.0/install.yaml

# Or delete namespace (removes everything)
kubectl delete namespace replizieren-system
```

**Note:** Uninstalling Replizieren does NOT delete the replicated Secrets and ConfigMaps. They will remain in their target namespaces.

## Troubleshooting

### Controller Not Starting

Check the logs:

```bash
kubectl logs -n replizieren-system deployment/replizieren-controller-manager
```

### RBAC Errors

If you see permission denied errors, ensure you have cluster-admin privileges when installing:

```bash
kubectl auth can-i create clusterrole --all-namespaces
```

### Resources Not Replicating

1. Verify the annotation is correct:
   ```bash
   kubectl get secret my-secret -o jsonpath='{.metadata.annotations}'
   ```

2. Check controller logs for errors:
   ```bash
   kubectl logs -n replizieren-system deployment/replizieren-controller-manager -f
   ```

3. Ensure target namespace exists:
   ```bash
   kubectl get namespace target-namespace
   ```

## Next Steps

- [Usage Guide](usage) - Learn how to configure replication
- [Examples](examples) - See real-world use cases
