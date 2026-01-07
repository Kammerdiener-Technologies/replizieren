---
layout: default
title: Contributing
---

# Contributing to Replizieren

Thank you for your interest in contributing to Replizieren! This document provides guidelines and information for contributors.

## Code of Conduct

Please be respectful and constructive in all interactions. We're all here to build something useful together.

## Getting Started

### Prerequisites

- Go 1.24+
- Docker 17.03+
- kubectl
- Access to a Kubernetes cluster (minikube, kind, or remote cluster)
- Make

### Setting Up Your Development Environment

1. **Fork the repository**

   Click the "Fork" button on GitHub to create your own copy.

2. **Clone your fork**

   ```bash
   git clone https://github.com/YOUR-USERNAME/replizieren.git
   cd replizieren
   ```

3. **Add upstream remote**

   ```bash
   git remote add upstream https://github.com/Kammerdiener-Technologies/replizieren.git
   ```

4. **Install dependencies**

   ```bash
   go mod download
   ```

5. **Set up envtest binaries**

   ```bash
   make setup-envtest
   ```

### Running the Project Locally

```bash
# Run the controller locally (connects to your current kubectl context)
make run

# In another terminal, create test resources
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
  namespace: default
  annotations:
    replizieren.dev/replicate: "kube-system"
type: Opaque
data:
  test: dGVzdA==
EOF
```

## Development Workflow

### Making Changes

1. **Create a feature branch**

   ```bash
   git checkout -b feature/my-feature
   ```

2. **Make your changes**

   Edit the code, add tests, update documentation as needed.

3. **Run tests**

   ```bash
   # Unit tests
   make test

   # Linting
   make lint

   # All checks
   make test lint
   ```

4. **Commit your changes**

   ```bash
   git add .
   git commit -m "feat: add my feature"
   ```

   We follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` - New features
   - `fix:` - Bug fixes
   - `docs:` - Documentation changes
   - `test:` - Test additions/changes
   - `refactor:` - Code refactoring
   - `chore:` - Maintenance tasks

5. **Push to your fork**

   ```bash
   git push origin feature/my-feature
   ```

6. **Create a Pull Request**

   Go to GitHub and create a PR from your branch to the main repository.

### Keeping Your Fork Updated

```bash
git fetch upstream
git checkout main
git merge upstream/main
git push origin main
```

## Project Structure

```
replizieren/
├── cmd/
│   └── main.go                    # Entry point
├── internal/
│   └── controller/
│       ├── secret_controller.go   # Secret reconciler
│       ├── configmapwatcher_controller.go  # ConfigMap reconciler
│       ├── replicator.go          # Shared helpers
│       └── *_test.go              # Tests
├── config/
│   ├── default/                   # Default kustomize overlay
│   ├── manager/                   # Controller deployment
│   └── rbac/                      # RBAC configuration
├── docs/                          # GitHub Pages documentation
└── test/
    └── e2e/                       # End-to-end tests
```

## Testing

### Unit Tests

Unit tests use Go's standard testing package and Ginkgo for BDD-style tests:

```bash
make test
```

### Adding Tests

When adding a new feature:
1. Add unit tests in `*_test.go` files
2. Use Ginkgo for integration tests that need envtest
3. Use standard Go tests for pure unit tests

Example test:

```go
func TestMyFunction(t *testing.T) {
    result := MyFunction("input")
    if result != "expected" {
        t.Errorf("expected 'expected', got '%s'", result)
    }
}
```

### E2E Tests

End-to-end tests run against a real cluster:

```bash
make test-e2e
```

## Code Style

### Go Code

- Follow standard Go conventions
- Run `go fmt` before committing
- Use meaningful variable names
- Add comments for exported functions

### Linting

We use golangci-lint:

```bash
make lint
```

Fix linting issues before submitting PRs.

## Documentation

### README

Keep the README updated with:
- New features
- Changed behavior
- Updated examples

### Code Comments

- Add godoc comments to exported functions
- Explain non-obvious logic with inline comments

### GitHub Pages Docs

Documentation lives in `/docs/`. To preview locally:

```bash
cd docs
bundle install
bundle exec jekyll serve
# Visit http://localhost:4000
```

## Pull Request Guidelines

### Before Submitting

- [ ] Tests pass (`make test`)
- [ ] Linting passes (`make lint`)
- [ ] Code is formatted (`go fmt ./...`)
- [ ] Documentation is updated if needed
- [ ] Commit messages follow Conventional Commits

### PR Description

Include:
- What the PR does
- Why it's needed
- How to test it
- Any breaking changes

### Review Process

1. Automated checks run (tests, lint)
2. Maintainers review the code
3. Address any feedback
4. PR is merged when approved

## Reporting Issues

### Bug Reports

Include:
- Kubernetes version
- Replizieren version
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs

### Feature Requests

Include:
- Use case description
- Proposed solution (if any)
- Alternatives considered

## Release Process

Releases are automated via GitHub Actions:

1. Maintainer creates a tag: `git tag v1.2.3`
2. Push the tag: `git push origin v1.2.3`
3. GitHub Actions builds and publishes:
   - Container image to GHCR
   - Release notes on GitHub

## Getting Help

- Open a GitHub issue for bugs/features
- Check existing issues before creating new ones
- Join discussions in existing PRs

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.

---

Thank you for contributing to Replizieren!
