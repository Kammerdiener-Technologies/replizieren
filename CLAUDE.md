# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Replizieren is a Kubernetes operator that replicates Secrets and ConfigMaps across namespaces and triggers deployment rollouts when these resources change. Built with Kubebuilder v4.6.0 and controller-runtime v0.21.0.

## Common Commands

```bash
# Build
make build                    # Build manager binary to bin/manager

# Test
make test                     # Run unit tests with coverage
make test-e2e                 # Run E2E tests (creates/destroys Kind cluster)
go test ./internal/controller/... -v -run TestSecretReplication  # Run specific test

# Lint
make lint                     # Run golangci-lint
make lint-fix                 # Run golangci-lint with auto-fix

# Run locally
make run                      # Run controller against current kubeconfig cluster

# Code generation (run after modifying RBAC markers or API types)
make manifests                # Generate RBAC and CRD manifests from +kubebuilder markers
make generate                 # Generate DeepCopy methods

# Deploy to cluster
make docker-build IMG=<registry>/replizieren:tag
make docker-push IMG=<registry>/replizieren:tag
make deploy IMG=<registry>/replizieren:tag
```

## Architecture

### Controllers (internal/controller/)

Two reconcilers watch Kubernetes resources and act on annotation changes:

- **SecretReconciler** (`secret_controller.go`): Watches Secrets
- **ConfigMapWatcherReconciler** (`configmapwatcher_controller.go`): Watches ConfigMaps

Both support these annotations:
- `replizieren.dev/replicate`: `"true"` (all namespaces) or comma-separated list (e.g., `"ns1,ns2"`)
- `replizieren.dev/rollout-on-update`: `"true"` to restart deployments using the resource

### Key Flow

1. Resource with annotation created/updated
2. Controller determines target namespaces
3. Replicates resource to each target namespace
4. If rollout-on-update is set, patches deployments with `restartedAt` timestamp annotation

### Entry Point

`cmd/main.go` initializes the controller-runtime Manager, sets up TLS, metrics, health checks, and registers both reconcilers.

## Testing

- **Unit tests**: Use Ginkgo/Gomega with envtest (fake Kubernetes API)
- **E2E tests**: Use Kind clusters via `test/e2e/`
- Test helpers in `internal/controller/test_util.go`

## RBAC

RBAC markers are in the controller files (e.g., `// +kubebuilder:rbac:groups=core,resources=secrets,...`). Run `make manifests` after changing them to regenerate `config/rbac/role.yaml`.
