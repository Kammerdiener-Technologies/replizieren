---
layout: default
title: API Reference
---

# API Reference

Complete reference for Replizieren annotations and behavior.

## Annotations

All Replizieren annotations use the `replizieren.dev/` prefix.

### replizieren.dev/replicate

**Type:** String
**Required:** Yes (for replication to occur)
**Applies to:** Secrets, ConfigMaps

Controls whether and where a resource should be replicated.

#### Values

| Value | Description |
|-------|-------------|
| `"namespace-name"` | Replicate to a single specific namespace |
| `"ns1, ns2, ns3"` | Replicate to multiple namespaces (comma-separated) |
| `"true"` | Replicate to all namespaces in the cluster |
| `"false"` | Explicitly disable replication |
| `""` (empty) | No replication (same as missing annotation) |

#### Behavior

- **Whitespace handling:** Spaces around namespace names are automatically trimmed
- **Empty entries:** Empty entries in comma-separated lists are ignored (`"ns1,,ns2"` = `"ns1, ns2"`)
- **Source exclusion:** The source namespace is always excluded from targets
- **Non-existent namespaces:** Replication to non-existent namespaces will log an error but continue processing other targets

#### Examples

```yaml
# Single namespace
annotations:
  replizieren.dev/replicate: "production"

# Multiple namespaces
annotations:
  replizieren.dev/replicate: "staging, production, testing"

# All namespaces
annotations:
  replizieren.dev/replicate: "true"

# Disabled
annotations:
  replizieren.dev/replicate: "false"
```

---

### replizieren.dev/rollout-on-update

**Type:** String (boolean)
**Required:** No
**Default:** `"false"`
**Applies to:** Secrets, ConfigMaps

Controls whether Deployments should be restarted when the resource is updated.

#### Values

| Value | Description |
|-------|-------------|
| `"true"` | Trigger rolling restart of affected Deployments |
| `"false"` | No automatic restarts (default) |
| (missing) | Same as `"false"` |

#### Behavior

When enabled and the resource is updated:

1. **Detection:** Finds all Deployments in affected namespaces that use the resource
2. **Annotation:** Adds/updates a timestamp annotation on the Pod template
3. **Restart:** Kubernetes performs a rolling restart due to the template change

##### Affected Namespaces

Rollouts are triggered in:
- The **source namespace** (where the original resource lives)
- All **target namespaces** (where the resource is replicated)

##### Detection Methods

A Deployment is considered to "use" a Secret if:
- It has a volume with `secret.secretName` matching the Secret name
- It has a container with `envFrom[].secretRef.name` matching the Secret name

A Deployment is considered to "use" a ConfigMap if:
- It has a volume with `configMap.name` matching the ConfigMap name
- It has a container with `envFrom[].configMapRef.name` matching the ConfigMap name

##### Annotations Added

For Secrets:
```yaml
spec:
  template:
    metadata:
      annotations:
        secret.restartedAt: "2024-01-15T10:30:00Z"
```

For ConfigMaps:
```yaml
spec:
  template:
    metadata:
      annotations:
        configmap.restartedAt: "2024-01-15T10:30:00Z"
```

#### Examples

```yaml
# Enable rollout
annotations:
  replizieren.dev/replicate: "production"
  replizieren.dev/rollout-on-update: "true"

# Replicate without rollout
annotations:
  replizieren.dev/replicate: "production"
  replizieren.dev/rollout-on-update: "false"
```

---

## Supported Resources

### Secrets

All Secret types are supported:

| Type | Supported |
|------|-----------|
| `Opaque` | Yes |
| `kubernetes.io/tls` | Yes |
| `kubernetes.io/dockerconfigjson` | Yes |
| `kubernetes.io/dockercfg` | Yes |
| `kubernetes.io/basic-auth` | Yes |
| `kubernetes.io/ssh-auth` | Yes |
| `kubernetes.io/service-account-token` | Yes |
| `bootstrap.kubernetes.io/token` | Yes |

### ConfigMaps

All ConfigMaps are supported, including those with:
- `data` (string key-value pairs)
- `binaryData` (binary data)

---

## Replicated Resource Properties

When a resource is replicated, the following properties are preserved:

| Property | Preserved | Notes |
|----------|-----------|-------|
| `metadata.name` | Yes | Same name in target namespace |
| `metadata.labels` | Yes | All labels copied |
| `metadata.annotations` | Yes | All annotations copied (including replication annotations) |
| `data` | Yes | All data copied |
| `binaryData` | Yes | All binary data copied |
| `type` | Yes | Secret type preserved |
| `stringData` | No | Converted to `data` by Kubernetes |

The following are NOT preserved (set by Kubernetes):
- `metadata.namespace` (set to target namespace)
- `metadata.uid`
- `metadata.resourceVersion`
- `metadata.creationTimestamp`
- `metadata.ownerReferences`

---

## Controller Behavior

### Reconciliation

The controller reconciles on:
- Resource creation
- Resource update
- Resource deletion (no action taken on replicated copies)

### Error Handling

| Scenario | Behavior |
|----------|----------|
| Target namespace doesn't exist | Error logged, continues with other targets |
| Permission denied | Error logged, continues with other targets |
| Resource conflict | Retries with exponential backoff |
| Network error | Retries with exponential backoff |

### Leader Election

The controller uses leader election for high availability. Only one instance is active at a time, ensuring no duplicate processing.

### Rate Limiting

The controller uses Kubernetes' standard rate limiting:
- Initial retry: 5ms
- Max retry: 1000s
- Exponential backoff between retries

---

## RBAC Requirements

The controller requires the following ClusterRole permissions:

```yaml
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "patch"]
```

---

## Metrics

The controller exposes standard controller-runtime metrics on port 8080 (configurable):

| Metric | Description |
|--------|-------------|
| `controller_runtime_reconcile_total` | Total reconciliations |
| `controller_runtime_reconcile_errors_total` | Failed reconciliations |
| `controller_runtime_reconcile_time_seconds` | Reconciliation duration |
| `workqueue_depth` | Current queue depth |
| `workqueue_adds_total` | Total items added to queue |

---

## Health Endpoints

| Endpoint | Port | Purpose |
|----------|------|---------|
| `/healthz` | 8081 | Liveness probe |
| `/readyz` | 8081 | Readiness probe |

---

## Environment Variables

The controller does not require any environment variables. All configuration is done through command-line flags set in the Deployment manifest.

### Command-Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--leader-elect` | false | Enable leader election |
| `--health-probe-bind-address` | `:8081` | Health probe bind address |
| `--metrics-bind-address` | `:8080` | Metrics bind address |

---

## Compatibility

| Kubernetes Version | Support |
|-------------------|---------|
| 1.24+ | Fully tested |
| 1.20-1.23 | Should work |
| < 1.20 | Not tested |

| Go Version | Support |
|------------|---------|
| 1.24+ | Required |
