# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

FSM (Flomesh Service Mesh) is a lightweight, SMI-native service mesh for Kubernetes written in Go. Key characteristics:
- **Data Plane**: Uses [Flomesh Pipy](https://github.com/flomesh-io/pipy) proxy instead of Envoy
- **Traffic Interception**: eBPF-based (not iptables-based)
- **Traffic Management**: Supports SMI Spec and Gateway API
- **Multi-cluster**: Service discovery and interconnectivity across clusters

## Build Commands

```bash
# Setup environment
cp .env.example .env

# Build
make build              # Build all FSM binaries with release ldflags
make build-fsm          # Build the fsm CLI tool only
make docker-build       # Build and push all Docker images
make kind-up            # Start local Kind cluster with registry

# Testing
make go-test            # Run unit tests
make go-test-coverage   # Run tests with HTML coverage report
make test-e2e           # Run end-to-end tests (Ginkgo/Gomega)
go test ./pkg/...       # Run tests for specific package

# Code Quality
make go-lint            # Run golangci-lint (via Docker)
make go-fmt             # Format Go code
make go-mod-tidy        # Tidy Go modules
make check-mocks        # Generate and verify mocks
make check-codegen      # Verify code generation
make chart-checks       # Verify Helm chart docs and lint
```

## Key Environment Variables

```bash
CTR_REGISTRY           # Container registry URL (required)
CTR_TAG                # Image tag (default: latest)
K8S_NAMESPACE          # FSM installation namespace (default: fsm-system)
CERT_MANAGER           # Certificate manager: tresor, vault, or cert-manager (default: tresor)
```

## Architecture

### Main Entry Points (cmd/)

| Binary | Purpose |
|--------|---------|
| `fsm-controller` | Main control plane |
| `fsm-injector` | Sidecar injection webhook |
| `fsm-bootstrap` | Bootstrap controller |
| `fsm-gateway` | Gateway API controller |
| `fsm-connector` | Multi-cluster connector |
| `fsm-ingress` | Ingress controller |
| `fsm-interceptor` | Traffic interceptor |
| `fsm` (cli) | CLI tool for managing FSM |

### Core Packages (pkg/)

- **catalog/**: Mesh Catalog - central component that collects inputs from all other components and dispatches configuration to proxies
- **certificate/**: Pluggable certificate manager interface with implementations for tresor (native), Vault, and cert-manager
- **endpoint/**: Endpoints providers that introspect compute platforms to retrieve IP addresses
- **smi/**: Wrapper around SMI Spec SDK for traffic policies
- **policy/**: Policy enforcement
- **injector/**: Sidecar injection logic
- **gateway/**: Gateway API implementation
- **trafficpolicy/**: Traffic policy types

### Design Patterns

1. **Mesh Catalog Pattern**: The catalog is the heart of FSM - it combines outputs from all other components and dispatches to proxies
2. **Pluggable Interfaces**: Certificate manager, endpoints providers use interfaces allowing different implementations
3. **Stateless Control Plane**: Proxy control plane is stateless, handles gRPC connections from sidecar proxies
4. **Kubernetes Controllers**: Use controller-runtime for reconcile loops on CRDs

### Supported Specifications

- **SMI Spec**: TrafficTarget, HTTPRouteGroup, TCPRoute, TrafficSplit
- **Gateway API**: GatewayClass, Gateway, HTTPRoute, GRPCRoute, TLSRoute, TCPRoute, UDPRoute, ReferenceGrant

## Code Conventions

- **Import prefix**: `github.com/flomesh-io/fsm`
- **Code formatting**: `goimports` (enforced by CI)
- **Mocking**: GoMock with `mockgen`, mocks prefixed with `mock_` and suffixed with `_generated`
- **Test coverage target**: 80% minimum
- **Test files**: Named `*_test.go` alongside implementation files

## Key Files

- `DESIGN.md` - High-level architecture and design decisions
- `charts/fsm/` - Main Helm chart for FSM control plane
- `tests/e2e/` - Ginkgo/Gomega end-to-end tests
- `mockspec/rules` - Mock generation rules
- `.golangci.yml` - Linter configuration

## Useful Locations

- CRDs: `cmd/fsm-bootstrap/crds/`
- Debug endpoints: `pkg/debugger/`
- eBPF programs: `bpf/`
- Demo app: `demo/` (Bookstore demo)
