# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

FSM (Flomesh Service Mesh) is a lightweight, SMI-native service mesh for Kubernetes written in Go.
- **Language**: Go 1.25+
- **Data Plane**: Uses [Flomesh Pipy](https://github.com/flomesh-io/pipy) proxy instead of Envoy
- **Traffic Interception**: eBPF-based (not iptables-based)
- **Traffic Management**: Supports SMI Spec and Gateway API
- **Multi-cluster**: Service discovery and interconnectivity across clusters

## Build Commands

```bash
# Setup environment
cp .env.example .env
source .env

# Build
make build              # Build all FSM binaries with release ldflags
make build-fsm          # Build the fsm CLI tool only
make docker-build       # Build and push all Docker images
make kind-up            # Start local Kind cluster with registry
make k3d-up             # Start local K3d cluster with registry (default for e2e)

# Testing
make go-test            # Run unit tests
make go-test-coverage   # Run tests with HTML coverage report
make test-e2e           # Run end-to-end tests (Ginkgo/Gomega, default: K3dCluster)
go test ./pkg/...       # Run tests for specific package

# Code Quality
make go-lint            # Run golangci-lint (via Docker)
make go-fmt             # Format Go code
make go-mod-tidy        # Tidy Go modules
make check-mocks        # Generate and verify mocks
make check-codegen      # Verify code generation
make chart-checks       # Verify Helm chart docs and lint
make manifests          # Generate CRDs
```

## Key Environment Variables

```bash
CTR_REGISTRY           # Container registry URL (required)
CTR_TAG                # Image tag (default: latest)
K8S_NAMESPACE          # FSM installation namespace (default: fsm-system)
CERT_MANAGER           # Certificate manager: tresor, vault, or cert-manager (default: tresor)
MESH_NAME              # Mesh identifier (default: fsm)
```

## Architecture

### Main Entry Points (cmd/)

| Binary | Purpose |
|--------|---------|
| `fsm-controller` | Main control plane - programs sidecar proxies |
| `fsm-injector` | Sidecar injection webhook |
| `fsm-bootstrap` | Bootstrap controller |
| `fsm-gateway` | Gateway API controller |
| `fsm-connector` | Multi-cluster connector |
| `fsm-ingress` | Ingress controller |
| `fsm-preinstall` | Pre-install components |
| `fsm-xnetmgmt` | External network management |
| `fsm-healthcheck` | Health check server |
| `fsm` (cli) | CLI tool for managing FSM |

### Core Packages (pkg/)

- **catalog/**: Mesh Catalog - central component combining all component outputs and dispatching to proxies
- **certificate/**: Pluggable certificate manager with tresor (native), Vault, and cert-manager implementations
- **endpoint/**: Endpoints providers for Kubernetes and other platforms
- **smi/**: Wrapper around SMI Spec SDK for traffic policies
- **policy/**: Policy enforcement (TrafficTarget, HTTPRouteGroup, etc.)
- **injector/**: Sidecar injection webhook logic
- **gateway/**: Gateway API implementation
- **trafficpolicy/**: Traffic policy types and conversions
- **ingress/**: Ingress controller implementation
- **multicluster/**: Multi-cluster connectivity
- **connector/**: Cluster connector functionality
- **sidecar/**: Sidecar proxy management (v1 providers for Pipy)
- **repo/**: Repository for proxy configuration
- **providers/**: FSM and kube providers
- **k8s/**: Kubernetes utilities and informers
- **controllers/**: Various Kubernetes controllers (flb, gateway, ingress, namespacedingress, etc.)
- **webhook/**: Webhook implementations
- **reconciler/**: Kubernetes reconcilers
- **messaging/**: Pub/sub for internal events
- **configurator/**: Mesh configuration management

### Design Patterns

1. **Mesh Catalog Pattern**: Central component combining all component outputs
2. **Pluggable Interfaces**: Certificate manager, endpoints providers use interfaces for different implementations
3. **Stateless Control Plane**: Proxy control plane is stateless, handles gRPC from sidecar proxies
4. **Kubernetes Controllers**: Use controller-runtime for reconcile loops on CRDs
5. **Pub/Sub Messaging**: Internal event communication via pub/sub
6. **Webhook Pattern**: Mutating and validating webhooks for injection and policy

### Supported Specifications

**SMI Spec:**
- TrafficTarget, HTTPRouteGroup, TCPRoute, TrafficSplit

**Gateway API:**
- GatewayClass, Gateway, HTTPRoute, GRPCRoute
- TLSRoute, TCPRoute, UDPRoute, ReferenceGrant

## Code Conventions

- **Import prefix**: `github.com/flomesh-io/fsm`
- **Code formatting**: `goimports` (enforced by CI)
- **Mocking**: GoMock with `mockgen`, mocks prefixed with `mock_` and suffixed with `_generated`
- **Test coverage target**: 80% minimum
- **Test files**: Named `*_test.go` alongside implementation files
- **Generated files**: `zz_generated.deepcopy.go`, `mock_*_generated.go` excluded from coverage

## Tool Versions (from Makefile)

- controller-gen: v0.18.0
- kustomize: v5.4.3
- helm: v3.18.6
- Go: 1.25+

## Key Files

- `DESIGN.md` - High-level architecture and design decisions
- `charts/fsm/` - Main Helm chart for FSM control plane
- `tests/e2e/` - Ginkgo/Gomega end-to-end tests
- `tests/framework/` - Test framework utilities
- `mockspec/rules` - Mock generation rules
- `.golangci.yml` - Linter configuration
- `.goreleaser.yaml` - Release configuration

## Useful Locations

- CRDs: `cmd/fsm-bootstrap/crds/`
- Debug endpoints: `pkg/debugger/`
- eBPF programs: `bpf/`
- Demo app: `demo/` (Bookstore demo)
- Go tools: `tools.go`
