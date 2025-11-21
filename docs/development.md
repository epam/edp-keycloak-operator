# Development Guide

This document provides guidance for developers working on the EDP Keycloak Operator project.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Project Structure](#project-structure)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Building and Deployment](#building-and-deployment)
- [Running and Debugging Locally](#running-and-debugging-locally)
- [Common Development Tasks](#common-development-tasks)
- [Getting Help](#getting-help)

## Prerequisites

Before starting development, ensure you have the following tools installed:

### Required Tools

| Tool | Purpose |
|------|---------|
| Go (1.24+) | Programming language |
| Docker/Podman | Container runtime |
| kubectl | Kubernetes CLI |
| [kind](https://github.com/kubernetes-sigs/kind) | Local Kubernetes cluster |
| Helm | Package manager |
| make | Build automation |

### Optional Tools

| Tool | Purpose |
|------|---------|
| [operator-sdk](https://sdk.operatorframework.io/docs/) | Operator development framework |
| [kuttl](https://github.com/kudobuilder/kuttl) | E2E testing |

**Note**: The following tools are automatically installed by the Makefile when needed:
- `golangci-lint` - Code linting
- `mockery` - Mock generation
- `controller-gen` - Code generation
- `kustomize` - Kubernetes manifest management
- `helm-docs` - Helm documentation generation
- `crdoc` - CRD documentation generation

## Quick Start

Get started with development:

```bash
# 1. Clone the repository
git clone https://github.com/epam/edp-keycloak-operator.git
cd edp-keycloak-operator

# 2. Start local Kubernetes cluster
make start-kind

# 3. Install CRDs
make install

# 4. Start test Keycloak instance
make start-keycloak

# 5. Run tests
TEST_KEYCLOAK_URL=http://localhost:8086 make test

# 6. Build the operator
make build
```

## Project Structure

```
edp-keycloak-operator/
├── api/                         # API definitions
│   ├── v1/                      # v1 API version
│   └── v1alpha1/                # v1alpha1 API version
├── bin/                         # Downloaded development tools
├── bundle/                      # OLM bundle manifests
├── cmd/                         # Main application entry point
├── config/                      # Kubernetes manifests
├── deploy-templates/            # Helm chart templates
├── docs/                        # Documentation
├── hack/                        # Development scripts
├── internal/                    # Private application code
│   └── controller/              # Controller implementations
├── pkg/                         # Public library code
│   ├── client/                  # Keycloak client
│   ├── secretref/               # Secret reference utilities
│   └── util/                    # Utility functions
└── tests/                       # E2E test files
```

## Development Workflow

### Feature Development

```bash
# 1. Create feature branch
git checkout -b feature/new-feature

# 2. Make changes
# Edit code files

# 3. Generate code
make manifests
make generate
make mocks

# 4. Run tests
make test

# 5. Run linter
make lint-fix

# 6. Commit (project requires one commit per PR for clean history)
git add .
git commit -m "feat: add new feature (#issue_number)"

# For subsequent changes, amend the existing commit:
git add .
git commit --amend --no-edit

# Push changes (--force-with-lease is safer than --force as it prevents overwriting others' work)
git push --force-with-lease
```

## Testing

The project uses `make test` as the main testing command, which intelligently runs different test types based on environment variables.

### Unit Tests

Unit tests run by default and don't require external dependencies:

```bash
# Run unit tests only (integration tests will be skipped)
make test

# Run tests for specific package
go test ./pkg/client/keycloak/...

# Run tests with verbose output
go test -v ./...
```

### Integration Tests

Integration tests run automatically when `TEST_KEYCLOAK_URL` is specified. They require:

- **Keycloak instance** - Provides the Keycloak server for testing
- **envtest** - Kubernetes API server for testing controller logic
- **Ginkgo** - BDD testing framework (github.com/onsi/ginkgo)
- **Gomega** - Ginkgo's preferred matcher library (github.com/onsi/gomega)

```bash
# Start test Keycloak instance
make start-keycloak

# Run unit tests + integration tests
TEST_KEYCLOAK_URL="http://localhost:8086" make test
```

**How it works**: The `make test` command checks if `TEST_KEYCLOAK_URL` is set:
- **Without `TEST_KEYCLOAK_URL`**: Runs only unit tests, shows warning that integration tests are skipped
- **With `TEST_KEYCLOAK_URL`**: Runs both unit tests and integration tests

### End-to-End Tests

End-to-end tests require a Kubernetes cluster and use the following tools:

- **kind** - Local Kubernetes cluster for testing
- **kuttl** - Kubernetes testing framework for e2e tests
- **Docker/Podman** - Container runtime for building and loading test images

```bash
# Start kind cluster
make start-kind

# Run e2e tests
make e2e

# Clean up resources
make delete-kind
```

## Building and Deployment

### Local Build

```bash
# Build binary
make build

# Build container image
docker build -t keycloak-operator:latest .
```

### Helm Chart Development

```bash
# Generate Helm documentation
make helm-docs

# Validate Helm chart
helm lint deploy-templates/
```

### Bundle Generation

```bash
# Generate a new bundle version
VERSION=1.29.0 CHANNELS="stable" DEFAULT_CHANNEL=stable make bundle
```

## Running and Debugging Locally

This section covers how to run and debug the operator locally during development.

### Prerequisites

Before running the operator locally, ensure you have:

1. **Kubernetes Cluster**: The operator needs to connect to a Kubernetes cluster using the current active context from your kubeconfig.

2. **Keycloak Instance**: A running Keycloak server for the operator to manage.

### Step-by-Step Setup

#### 1. Start Local Kubernetes Cluster

```bash
# Start a local kind cluster
make start-kind
```

#### 2. Install CRDs

```bash
# Install Custom Resource Definitions to the cluster
make install
```

#### 3. Start Keycloak Instance

```bash
# Start a local Keycloak server (runs on port 8086)
make start-keycloak
```

#### 4. Configure Environment Variables

The operator requires the following environment variables:

- `WATCH_NAMESPACE`: Namespace where the operator will reconcile resources (empty string means all namespaces)
- `OPERATOR_NAMESPACE`: Namespace used for cluster resources to get secrets with credentials

```bash
# Example: Run operator watching all namespaces, using default namespace for secrets
export WATCH_NAMESPACE=""
export OPERATOR_NAMESPACE="default"

# Or watch only a specific namespace
export WATCH_NAMESPACE="default"
export OPERATOR_NAMESPACE="default"
```

#### 5. Run the Operator

##### Option A: Command Line

```bash
# Set environment variables and run
WATCH_NAMESPACE="" OPERATOR_NAMESPACE="default" go run cmd/main.go
```

##### Option B: VS Code Debug

Use the preconfigured VS Code launch configuration named `local`:

1. Open VS Code in the project root
2. Go to Run and Debug (Ctrl/Cmd + Shift + D)
3. Select "local" configuration
4. Press F5 to start debugging

#### 6. Create Test Resources

Create a Keycloak resource and secret for testing:

```yaml
apiVersion: v1.edp.epam.com/v1
kind: Keycloak
metadata:
  name: keycloak-sample
spec:
  secret: keycloak-access
  url: http://host.docker.internal:8086  # For Docker Desktop
  # Alternative URL:
  # url: http://192.168.0.146:8086  # Use your local IP

---
apiVersion: v1
kind: Secret
metadata:
  name: keycloak-access
data:
  username: YWRtaW4=  # admin (base64 encoded)
  password: YWRtaW4=  # admin (base64 encoded)
```
Apply the resources:
```bash
kubectl apply -f keycloak-sample.yaml
```

#### 7. Verify Setup

Check that the Keycloak resource is properly reconciled:

```bash
# Check Keycloak resource status
kubectl get keycloak keycloak-sample

# View detailed status
kubectl describe keycloak keycloak-sample
```

The status should show `connected: true` when the operator successfully connects to Keycloak.

### Debugging Tips

#### Getting the Local IP Address

If `host.docker.internal` doesn't work, get your local IP address:

```bash
# macOS/Linux
ipconfig getifaddr $(route get default | awk '/interface: / {print $2}')

# Alternative for macOS
ifconfig | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -1
```

#### Common Issues

1. **Connection Refused**: Ensure Keycloak is running on the expected port (8086)
2. **CRD Not Found**: Run `make install` to install CRDs
3. **Permission Denied**: Ensure your kubeconfig has proper permissions
4. **Namespace Issues**: Verify `OPERATOR_NAMESPACE` exists and contains the secret

#### Logs and Debugging

- The operator logs will appear in your terminal or VS Code debug console
- Use `kubectl logs` to check other cluster resources
- Set log level with `--zap-log-level=debug` for more verbose output

## Common Development Tasks

This section covers typical development workflows for extending the operator.

### Adding a New Field to Existing Resources

When adding new functionality to existing Custom Resources like `KeycloakClient`:

#### Example: Adding a field to KeycloakClient

```bash
# 1. Add the new field to the resource spec
# Edit api/v1/keycloakclient_types.go
# Add your field to KeycloakClientSpec struct

# 2. Generate code and manifests
make generate  # Updates deepcopy methods
make manifests # Updates CRD YAML files

# 3. Implement business logic
# Edit files in internal/controller/keycloakclient/chain/
# Map your new field to the corresponding Keycloak client configuration

# 4. Update Keycloak client methods (if needed)
# Edit files in pkg/client/keycloak/
# Add new methods to interact with Keycloak API

# 5. Generate mocks for testing
make mocks

# 6. Write comprehensive tests
# Unit tests: Add to relevant *_test.go files
# Integration tests: Add to keycloakclient_controller_integration_test.go

# 7. Run linter and fix issues
make lint-fix

# 8. Test your changes
TEST_KEYCLOAK_URL="http://localhost:8086" make test

# 9. Commit and push
git add .
git commit -m "feat: Add new field to KeycloakClient (#issue_number)"
git push --force-with-lease
```

#### Key Files to Modify:
- **API Definition**: `api/v1/keycloakclient_types.go` - Add the field to the spec
- **Controller Logic**: `internal/controller/keycloakclient/chain/` - Implement the business logic
- **Keycloak Client**: `pkg/client/keycloak/` - Add API methods if needed
- **Tests**: Write both unit and integration tests

### Adding a New Custom Resource and Controller

For completely new functionality, create new Custom Resources:

```bash
# 1. Use operator-sdk to scaffold the new resource
operator-sdk create api \
  --group v1.edp.epam.com \
  --version v1 \
  --kind KeycloakSomeResource \
  --resource \
  --controller

# 2. Customize the generated files
# Edit api/v1/keycloaksomeresource_types.go - Define your resource spec and status
# Edit internal/controller/keycloaksomeresource/ - Implement controller logic

# 3. Generate code and manifests
make generate
make manifests

# 4. Add RBAC permissions
# Add kubebuilder RBAC markers to your controller file:
# // +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloaksomeresources,verbs=get;list;watch;create;update;patch;delete
# // +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloaksomeresources/status,verbs=get;update;patch
# // +kubebuilder:rbac:groups=v1.edp.epam.com,namespace=placeholder,resources=keycloaksomeresources/finalizers,verbs=update
# Run 'make manifests' to generate config/rbac/ files
# Manually update deploy-templates/templates/ to align with generated RBAC

# 5. Implement Keycloak integration
# Add methods to pkg/client/keycloak/ for your resource

# 6. Generate mocks and write tests
make mocks
# Write unit tests for controller logic
# Write integration tests following existing patterns

# 7. Add CRD examples
# Create example YAML in deploy-templates/_crd_examples/

# 8. Test and validate
make lint-fix
TEST_KEYCLOAK_URL="http://localhost:8086" make test

# 9. Commit your work
git add .
git commit -m "feat: Add KeycloakSomeResource controller (#issue_number)"
git push --force-with-lease
```

#### Reference Documentation:
- **Operator SDK**: [Building Operators with Go](https://sdk.operatorframework.io/docs/building-operators/golang/quickstart/)
- **Controller Runtime**: [Kubebuilder Book](https://book.kubebuilder.io/)
- **RBAC Markers**: [Controller Runtime RBAC](https://book.kubebuilder.io/reference/markers/rbac.html)

### Modifying Existing Controller Logic

When updating business logic for existing resources:

```bash
# 1. Identify the chain element to modify
# Controllers use a chain pattern in internal/controller/*/chain/

# 2. Make your changes
# Edit the appropriate chain file (e.g., put_client.go, put_client_role.go)

# 3. Update tests
# Modify existing unit tests
# Add new test cases for your changes

# 4. Test thoroughly
make lint-fix
TEST_KEYCLOAK_URL="http://localhost:8086" make test

# 5. Commit changes
git add .
git commit -m "fix: Update client role logic (#issue_number)"
git push --force-with-lease
```

### Best Practices

1. **Always run code generation**: Use `make generate` and `make manifests` after API changes
2. **Use RBAC markers**: Add kubebuilder RBAC markers to controllers, don't edit `config/rbac/` manually
3. **Sync Helm RBAC**: After generating RBAC with `make manifests`, manually align `deploy-templates/templates/` RBAC
4. **Test thoroughly**: Write both unit and integration tests for new functionality
5. **Follow naming conventions**: Use consistent naming patterns for fields and methods
6. **Document your changes**: Add comments to complex logic and update examples
7. **Validate with real Keycloak**: Test against actual Keycloak instance, not just mocks

## Getting Help

- **Issues**: Create GitHub issues for bugs and feature requests
- **Code Examples**: Review test files for usage patterns and CRD examples in `deploy-templates/_crd_examples/` 