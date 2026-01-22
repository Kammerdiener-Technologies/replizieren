---
layout: default
title: Usage
---

# Usage Guide

This guide explains how to use Replizieren to replicate Secrets and ConfigMaps across namespaces.

## Basic Concept

Replizieren works by watching for Secrets and ConfigMaps with specific annotations. When it finds a resource with the `replizieren.dev/replicate` annotation, it automatically creates or updates copies in the specified target namespaces.

## Annotations Reference

### replizieren.dev/replicate

Controls where the resource should be replicated.

| Value | Behavior |
|-------|----------|
| `"namespace"` | Replicate to a single namespace |
| `"ns1, ns2, ns3"` | Replicate to multiple namespaces (comma-separated) |
| `"true"` | Replicate to ALL namespaces (legacy, use `replicate-all` instead) |
| `"false"` | Explicitly disable replication |
| (empty/missing) | No replication |

### replizieren.dev/replicate-all

Controls replication to all namespaces. This is the preferred way to replicate to all namespaces.

| Value | Behavior |
|-------|----------|
| `"true"` | Replicate to ALL namespaces in the cluster |
| `"false"` | Disable "all namespaces" mode (allows `replicate: "true"` to target a namespace named "true") |
| (empty/missing) | Use `replicate` annotation behavior |

> **Recommendation:** Use `replicate-all: "true"` instead of `replicate: "true"` for replicating to all namespaces. This removes ambiguity if you have a namespace literally named "true".

### replizieren.dev/rollout-on-update

Controls whether Deployments should be restarted when the resource changes.

| Value | Behavior |
|-------|----------|
| `"true"` | Restart Deployments using this resource |
| `"false"` or (missing) | No automatic restarts |

## Replication Modes

### Single Namespace

Replicate to one specific namespace:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: database-credentials
  namespace: default
  annotations:
    replizieren.dev/replicate: "production"
type: Opaque
data:
  username: YWRtaW4=
  password: cGFzc3dvcmQ=
```

### Multiple Namespaces

Replicate to several specific namespaces:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
  namespace: default
  annotations:
    replizieren.dev/replicate: "staging, production, testing"
data:
  config.yaml: |
    log_level: info
```

**Note:** Whitespace around namespace names is automatically trimmed.

### All Namespaces

Replicate to every namespace in the cluster:

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
  .dockerconfigjson: eyJhdXRocyI6e319
```

**Important:**
- The source namespace is always excluded from replication targets to prevent conflicts.
- System namespaces (`kube-system`, `kube-public`, `kube-node-lease`) are automatically excluded.

### Automatic Replication to New Namespaces (v0.1.0+)

When you create a new namespace, Replizieren automatically replicates all resources that have `replicate-all: "true"` to the new namespace. No manual action required!

```bash
# Create a secret that replicates everywhere
kubectl create secret generic shared-secret \
  --from-literal=key=value \
  -n default

kubectl annotate secret shared-secret \
  replizieren.dev/replicate-all="true" \
  -n default

# Later, create a new namespace
kubectl create namespace my-new-app

# The secret is automatically replicated!
kubectl get secret shared-secret -n my-new-app
```

## Automatic Updates

When you update a source Secret or ConfigMap, Replizieren automatically updates all replicated copies:

```bash
# Update the source secret
kubectl patch secret my-secret -n default \
  --type='json' \
  -p='[{"op": "replace", "path": "/data/password", "value": "bmV3cGFzc3dvcmQ="}]'

# The change propagates automatically to all target namespaces
```

## Rollout Triggers

### How It Works

When `rollout-on-update` is enabled, Replizieren adds a timestamp annotation to Deployments that use the updated resource:

```yaml
spec:
  template:
    metadata:
      annotations:
        secret.restartedAt: "2024-01-15T10:30:00Z"
        # or for ConfigMaps:
        configmap.restartedAt: "2024-01-15T10:30:00Z"
```

This triggers Kubernetes to perform a rolling restart of the Deployment pods.

### Deployment Detection

Replizieren detects Deployments using a Secret or ConfigMap through:

1. **Volume Mounts**
   ```yaml
   spec:
     template:
       spec:
         volumes:
           - name: config
             secret:
               secretName: my-secret
   ```

2. **Environment Variables (envFrom)**
   ```yaml
   spec:
     template:
       spec:
         containers:
           - name: app
             envFrom:
               - secretRef:
                   name: my-secret
   ```

### Example with Rollout

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
  DATABASE_URL: postgres://localhost:5432/myapp
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: production
spec:
  template:
    spec:
      containers:
        - name: app
          envFrom:
            - configMapRef:
                name: app-config
```

When `app-config` is updated:
1. The ConfigMap is replicated to `production`
2. The `my-app` Deployment is automatically restarted

### Rollout Scope

Rollouts are triggered in:
- **Source namespace**: The namespace where the original resource lives
- **Target namespaces**: All namespaces where the resource is replicated

## Disabling Replication

### Temporarily Disable

Set the annotation to `"false"`:

```bash
kubectl annotate secret my-secret \
  replizieren.dev/replicate="false" \
  --overwrite
```

### Remove Replication

Remove the annotation entirely:

```bash
kubectl annotate secret my-secret \
  replizieren.dev/replicate-
```

**Note:** Removing replication does NOT delete the already-replicated copies. You must delete them manually if needed.

## Best Practices

### 1. Use Specific Namespaces When Possible

Instead of `replicate-all: "true"` (all namespaces), specify exactly which namespaces need the resource:

```yaml
annotations:
  replizieren.dev/replicate: "production, staging"
```

This improves security and reduces unnecessary resource copies.

### 2. Be Careful with Rollout Triggers

Enabling `rollout-on-update` means every change causes pod restarts. Consider:
- Only enable for resources that truly require restarts
- Test in non-production environments first
- Be aware of the impact on availability during updates

### 3. Monitor Replication

Check controller logs periodically:

```bash
kubectl logs -n replizieren-system deployment/replizieren-controller-manager
```

### 4. Naming Conventions

Use clear naming to indicate replicated resources:

```yaml
name: shared-database-credentials  # Clear it's shared
```

### 5. Document Your Replication Strategy

Keep track of which resources are replicated where, especially in large clusters.

## Limitations

1. **Namespace Must Exist** (for specific targets): When using `replicate: "ns1, ns2"`, target namespaces must exist. However, with `replicate-all: "true"`, new namespaces are automatically detected (v0.1.0+)
2. **No Cross-Cluster**: Replication only works within a single Kubernetes cluster
3. **No Selective Fields**: The entire resource is replicated; you cannot replicate only specific keys
4. **No Transformation**: Data is copied as-is; no templating or transformation is supported
5. **System Namespaces Excluded**: `kube-system`, `kube-public`, and `kube-node-lease` are always excluded from `replicate-all`

## Next Steps

- [Examples](examples) - See detailed examples for common use cases
- [API Reference](api-reference) - Complete annotation documentation
