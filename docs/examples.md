---
layout: default
title: Examples
---

# Examples

This page provides real-world examples of using Replizieren for common Kubernetes scenarios.

## Docker Registry Credentials

Share private registry credentials across all namespaces so any workload can pull images.

### Create the Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: docker-registry-credentials
  namespace: default
  annotations:
    replizieren.dev/replicate-all: "true"
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: |
    eyJhdXRocyI6eyJnaGNyLmlvIjp7InVzZXJuYW1lIjoibXl1c2VyIiwicGFzc3dvcmQiOiJteXRva2VuIn19fQ==
```

### Use in a Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: production  # Any namespace works!
spec:
  template:
    spec:
      imagePullSecrets:
        - name: docker-registry-credentials
      containers:
        - name: app
          image: ghcr.io/myorg/myapp:latest
```

The secret is automatically available in `production` (and all other namespaces).

---

## TLS Certificates

Replicate TLS certificates from cert-manager to multiple ingress namespaces.

### Source Certificate

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: wildcard-example-com
  namespace: cert-manager
  annotations:
    replizieren.dev/replicate: "ingress-nginx, istio-system, default"
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTi...
  tls.key: LS0tLS1CRUdJTi...
```

### Use in Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-app
  namespace: default
spec:
  tls:
    - hosts:
        - app.example.com
      secretName: wildcard-example-com  # Automatically replicated here
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-app
                port:
                  number: 80
```

---

## Database Credentials with Auto-Rollout

Share database credentials and automatically restart applications when credentials rotate.

### The Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-credentials
  namespace: secrets
  annotations:
    replizieren.dev/replicate: "api, worker, scheduler"
    replizieren.dev/rollout-on-update: "true"
type: Opaque
data:
  POSTGRES_HOST: cG9zdGdyZXMuZGIuc3ZjLmNsdXN0ZXIubG9jYWw=
  POSTGRES_USER: YXBwdXNlcg==
  POSTGRES_PASSWORD: c3VwZXJzZWNyZXQ=
  POSTGRES_DB: bXlhcHA=
```

### The Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
  namespace: api
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: api
          image: myapp/api:latest
          envFrom:
            - secretRef:
                name: postgres-credentials
```

When you update `postgres-credentials`:
1. The new values are replicated to `api`, `worker`, and `scheduler` namespaces
2. Deployments using the secret are automatically restarted
3. New pods pick up the new credentials

### Rotating Credentials

```bash
# Generate new password
NEW_PASSWORD=$(openssl rand -base64 24)

# Update the secret
kubectl patch secret postgres-credentials -n secrets \
  --type='json' \
  -p="[{\"op\": \"replace\", \"path\": \"/data/POSTGRES_PASSWORD\", \"value\": \"$(echo -n $NEW_PASSWORD | base64)\"}]"

# Watch the rollout happen automatically
kubectl rollout status deployment/api-server -n api
```

---

## Feature Flags

Distribute feature flags across environments with automatic updates.

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: feature-flags
  namespace: platform
  annotations:
    replizieren.dev/replicate: "staging, production"
    replizieren.dev/rollout-on-update: "true"
data:
  features.json: |
    {
      "new_checkout_flow": true,
      "dark_mode": true,
      "beta_features": false,
      "maintenance_mode": false
    }
```

### Using in Your App

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: production
spec:
  template:
    spec:
      containers:
        - name: frontend
          image: myapp/frontend:latest
          volumeMounts:
            - name: features
              mountPath: /app/config
              readOnly: true
      volumes:
        - name: features
          configMap:
            name: feature-flags
```

### Toggle a Feature

```bash
# Enable beta features in production
kubectl patch configmap feature-flags -n platform \
  --type='json' \
  -p='[{"op": "replace", "path": "/data/features.json", "value": "{\"new_checkout_flow\": true, \"dark_mode\": true, \"beta_features\": true, \"maintenance_mode\": false}"}]'

# The frontend pods restart automatically with the new flags
```

---

## Shared Application Configuration

Maintain consistent configuration across multiple services.

### Shared Config

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: logging-config
  namespace: shared-config
  annotations:
    replizieren.dev/replicate-all: "true"
data:
  log_level: "info"
  log_format: "json"
  log_output: "stdout"
  sentry_dsn: "https://xxx@sentry.io/123"
```

### Using in Services

```yaml
# Service A
apiVersion: apps/v1
kind: Deployment
metadata:
  name: service-a
  namespace: team-a
spec:
  template:
    spec:
      containers:
        - name: service
          envFrom:
            - configMapRef:
                name: logging-config
---
# Service B (different namespace)
apiVersion: apps/v1
kind: Deployment
metadata:
  name: service-b
  namespace: team-b
spec:
  template:
    spec:
      containers:
        - name: service
          envFrom:
            - configMapRef:
                name: logging-config
```

Both services use the same logging configuration, managed from a single source.

---

## Multi-Environment Setup

Replicate to specific environments based on your promotion workflow.

### Development to Staging

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-keys
  namespace: development
  annotations:
    replizieren.dev/replicate: "staging"  # Only staging gets dev keys
type: Opaque
data:
  STRIPE_KEY: c2tfdGVzdF94eHg=
  SENDGRID_KEY: U0cueHh4
```

### Staging to Production (Separate Secret)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-keys
  namespace: production-secrets
  annotations:
    replizieren.dev/replicate: "production"
type: Opaque
data:
  STRIPE_KEY: c2tfbGl2ZV94eHg=  # Production keys
  SENDGRID_KEY: U0cueXl5
```

---

## Service Account Tokens

Share service account tokens for external service access.

### The Token Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: monitoring-token
  namespace: monitoring
  annotations:
    replizieren.dev/replicate: "app-a, app-b, app-c"
type: Opaque
data:
  token: ZXlKaGJHY2lPaUpTVXpJMU5pSXNJblI1Y0NJNklrcFhWQ0o5...
```

### Using in a Sidecar

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-with-monitoring
  namespace: app-a
spec:
  template:
    spec:
      containers:
        - name: app
          image: myapp:latest
        - name: monitoring-sidecar
          image: monitoring-agent:latest
          env:
            - name: MONITORING_TOKEN
              valueFrom:
                secretKeyRef:
                  name: monitoring-token
                  key: token
```

---

## Tips for Production

### 1. Label Your Replicated Resources

Add labels to track replicated resources:

```yaml
metadata:
  labels:
    replizieren.dev/managed: "true"
    replizieren.dev/source-namespace: "secrets"
```

### 2. Use GitOps

Store your annotated resources in Git and deploy with ArgoCD or Flux:

```
manifests/
├── secrets/
│   ├── docker-registry.yaml    # replicate-all: "true"
│   ├── tls-certs.yaml          # replicate: "ingress-nginx"
│   └── database-creds.yaml     # replicate: "api, worker"
└── configmaps/
    ├── feature-flags.yaml      # replicate: "staging, prod"
    └── logging-config.yaml     # replicate-all: "true"
```

### 3. Monitor Replication

Set up alerts for replication failures:

```bash
# Check for replication errors in logs
kubectl logs -n replizieren-system deployment/replizieren-controller-manager | grep -i error
```

## Next Steps

- [API Reference](api-reference) - Complete annotation documentation
- [Usage Guide](usage) - Detailed configuration options
