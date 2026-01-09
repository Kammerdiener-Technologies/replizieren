---
layout: default
title: Home
---

# Replizieren

**Replizieren** (German for "replicate") is a Kubernetes operator that automatically replicates Secrets and ConfigMaps across namespaces. It also supports triggering rolling restarts of Deployments when the replicated resources change.

## Why Replizieren?

In Kubernetes, Secrets and ConfigMaps are namespace-scoped resources. This means if you have a TLS certificate, Docker registry credentials, or shared configuration that needs to be available in multiple namespaces, you typically have to:

1. Manually copy resources to each namespace
2. Keep them in sync when they change
3. Restart applications when configurations update

**Replizieren solves all of these problems automatically.**

## Key Features

### Automatic Replication

Simply add an annotation to your Secret or ConfigMap, and Replizieren will automatically create and maintain copies in your target namespaces.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: registry-credentials
  annotations:
    replizieren.dev/replicate-all: "true"  # Replicate to ALL namespaces
```

### Flexible Targeting

- **Single namespace**: `replizieren.dev/replicate: "production"`
- **Multiple namespaces**: `replizieren.dev/replicate: "staging, production, testing"`
- **All namespaces**: `replizieren.dev/replicate-all: "true"`

### Automatic Rollout Triggers

When a Secret or ConfigMap changes, you often need to restart the Deployments that use it. Replizieren can do this automatically:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  annotations:
    replizieren.dev/replicate: "production"
    replizieren.dev/rollout-on-update: "true"
```

## Quick Start

### 1. Install Replizieren

```bash
# Install a specific version (recommended)
kubectl apply -f https://github.com/Kammerdiener-Technologies/replizieren/releases/download/v0.0.1/install.yaml

# Or install the latest development version
kubectl apply -f https://raw.githubusercontent.com/Kammerdiener-Technologies/replizieren/main/dist/install.yaml
```

### 2. Annotate Your Resources

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: default
  annotations:
    replizieren.dev/replicate: "target-namespace"
type: Opaque
data:
  password: cGFzc3dvcmQxMjM=
```

### 3. Watch It Work

The Secret will automatically appear in `target-namespace`:

```bash
kubectl get secret my-secret -n target-namespace
```

## Use Cases

### Docker Registry Credentials

Share `imagePullSecrets` across all namespaces so any namespace can pull from your private registry:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: docker-registry
  namespace: default
  annotations:
    replizieren.dev/replicate-all: "true"
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: ...
```

### TLS Certificates

Replicate TLS certificates from cert-manager to multiple ingress namespaces:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: wildcard-tls
  namespace: cert-manager
  annotations:
    replizieren.dev/replicate: "ingress-nginx, istio-system"
type: kubernetes.io/tls
data:
  tls.crt: ...
  tls.key: ...
```

### Shared Configuration

Share common configuration across environments with automatic rollouts:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: feature-flags
  namespace: default
  annotations:
    replizieren.dev/replicate: "staging, production"
    replizieren.dev/rollout-on-update: "true"
data:
  flags.json: |
    {"new_feature": true}
```

## Compatibility

Replizieren is tested against multiple Kubernetes versions:

| Kubernetes | Status |
|------------|--------|
| 1.32.x | Tested |
| 1.31.x | Tested |
| 1.30.x | Tested |
| 1.29.x | Tested |
| 1.28.x | Tested |

See [API Reference](api-reference#compatibility) for full compatibility details.

## Next Steps

- [Installation Guide](installation) - Detailed installation options
- [Usage Guide](usage) - Complete configuration reference
- [Examples](examples) - Real-world usage patterns
- [API Reference](api-reference) - Annotation documentation

## Support

If you find Replizieren useful, consider supporting its development:

<a href="https://github.com/sponsors/Kammerdiener-Technologies" target="_blank"><img src="https://img.shields.io/badge/Sponsor-GitHub-ea4aaa.svg?style=for-the-badge&logo=github-sponsors" alt="GitHub Sponsors"></a>
<a href="https://buymeacoffee.com/kammerdiener" target="_blank"><img src="https://img.shields.io/badge/Buy%20Me%20A%20Coffee-support-yellow.svg?style=for-the-badge&logo=buy-me-a-coffee&logoColor=white" alt="Buy Me A Coffee"></a>

## License

Replizieren is open source and available under the [Apache 2.0 License](https://github.com/Kammerdiener-Technologies/replizieren/blob/main/LICENSE).
