# Flomesh Service Mesh Development Guide

Welcome to the Flomesh Service Mesh development guide!
Thank you for joining us on a journey to build an SMI-native lightweight service mesh. The first of our [core principles](https://github.com/flomesh-io/fsm#core-principles) is to create a system, which is "simple to understand and contribute to." We hope that you would find the source code easy to understand. If not - we invite you to help us fulfill this principle. There is no PR too small!

To understand _what_ Flomesh Service Mesh does - take it for a spin and kick the tires. Install it on your Kubernetes cluster by following [the getting started guide](https://docs.flomesh.io/docs/getting_started/).

To get a deeper understanding of how FSM functions - take a look at the detailed [software design](/DESIGN.md).

When you are ready to jump in - [fork the repo](https://docs.github.com/en/github/getting-started-with-github/fork-a-repo) and then [clone it](https://docs.github.com/en/github/creating-cloning-and-archiving-repositories/cloning-a-repository) on your workstation.

The directories in the cloned repo will be structured approximately like this:

<details>
  <summary>Click to expand directory structure</summary>
This in a non-exhaustive list of the directories in the FSM repo. It is provided
as a birds-eye view of where the different components are located.

- `charts/` - contains FSM Helm chart
- `ci/` - tools and scripts for the continuous integration system
- `cmd/` - FSM command line tools
- `crd/` - Custom Resource Definitions needed by FSM
- `demo/` - scripts and Kubernetes resources needed to run the Bookstore demonstration of Flomesh Service Mesh
- `docs/` - FSM documentation
- `pkg/` -
  - `catalog/` - Mesh Catalog component is the central piece of FSM, which collects inputs from all other components and dispatches configuration to the proxy control plane
  - `certificate/` - contains multiple implementations of 1st and 3rd party certificate issuers, as well as PEM and x509 certificate management tools
    - `providers/` -
      - `keyvault/` - implements integration with Azure Key Vault
      - `vault/` - implements integration with Hashicorp Vault
      - `tresor/` - FSM native certificate issuer
  - `debugger/` - web server and tools used to debug the service mesh and the controller
  - `endpoint/` - Endpoints are components capable of introspecting the participating compute platforms; these retrieve the IP addresses of the compute backing the services in the mesh. This directory contains integrations with supported compute providers.
    - `providers/` -
      - `azure/` - integrates with Azure
      - `kube/` - Kubernetes tools and informers integrations
  - `envoy/` - packages needed to translate SMI into xDS
    - `ads/` - Aggregated Discovery Service related tools
    - `cds/` - Cluster Discovery Service related tools
    - `cla/` - Cluster Load Assignment components
    - `eds/` - Endpoint Discovery Service tools
    - `lds/` - Listener Discovery Service tools
    - `rds/` - Route Discovery Service tools
    - `sds/` - Secret Discovery service related tools
  - `health/` - FSM controller liveness and readiness probe handlers
  - `ingress/` - package mutating the service mesh in response to the application of an Ingress Kubernetes resource
  - `injector/` - sidecar injection webhook and related tools
  - `kubernetes/` - Kubernetes event handlers and helpers
  - `logger/` - logging facilities
  - `metricsstore/` - FSM controller system metrics tools
  - `namespace/` - package with tools handling a service mesh spanning multiple Kubernetes namespaces.
  - `service/` - tools needed for easier handling of Kubernetes services
  - `signals/` - operating system signal handlers
  - `smi/` - SMI client, informer, caches and tools
  - `tests/` - test fixtures and other functions to make unit testing easier
  - `trafficpolicy/` - SMI related types
- `wasm/` - Source for a WebAssembly-based Envoy extension
</details>

The Flomesh Service Mesh controller is written in Go.
It relies on the [SMI Spec](https://github.com/servicemeshinterface/smi-spec/).
FSM leverages [Envoy proxy](https://github.com/envoyproxy/envoy) as a data plane and Envoy's [XDS v3](https://www.envoyproxy.io/docs/envoy/latest/api-v3/api) protocol, which is offered in Go by [go-control-plane](https://github.com/envoyproxy/go-control-plane).

## Get Go-ing

This Flomesh Service Mesh project uses [Go v1.19.0+](https://golang.org/). If you are not familiar with Go, spend some time with the excellent [Tour of Go](https://tour.golang.org/).

## Get the dependencies

The FSM packages rely on many external Go libraries.

Take a peek at the `go.mod` file in the root of this repository to see all dependencies.

Run `go get -d ./...` to download all required Go packages.

Also the project requires Docker. See how to [install Docker](https://docs.docker.com/engine/install/).

#### Makefile

Many of the operations within the FSM repo have GNU Makefile targets.
More notable:

- `make docker-build` builds and pushes all Docker images
- `make go-test` to run unit tests
- `make go-test-coverage` - run unit tests and output unit test coverage
- `make go-lint` runs golangci-lint
- `make go-fmt` - same as `go fmt ./...`
- `make go-vet` - same as `go vet ./...`

## Create Environment Variables

The FSM demos and examples rely on environment variables to make it usable on your localhost. The root of the FSM repository contains a file named `.env.example`. Copy the contents of this file into `.env`

```bash
cat .env.example > .env
```

The various environment variables are documented in the `.env.example` file itself. Modify the variables in `.env` to suite your environment.

Some of the scripts and build targets available expect an accessible container registry where to push the `fsm-controller` and `init` docker images once compiled. The location and credential options for the container registry can be specified as environment variables declared in `.env`, as well as the target namespace where `fsm-controller` will be installed on.

Additionally, if using `demo/` scripts to deploy FSM's provided demo on your own K8s cluster, the same container registry configured in `.env` will be used to pull FSM images on your K8s cluster.

```console
$ # K8S_NAMESPACE is the Namespace the control plane will be installed into
$ export K8S_NAMESPACE=fsm-system

$ # CTR_REGISTRY is the URL of the container registry to use
$ export CTR_REGISTRY=<your registry>

$ # If no authentication to push to the container registry is required, the following steps may be skipped.

$ # For Azure Container Registry (ACR), the following command may be used: az acr credential show -n <your_registry_name> --query "passwords[0].value" | tr -d '"'
$ export CTR_REGISTRY_PASSWORD=<your password>

$ # Create docker secret in Kubernetes Namespace using following script:
$ ./scripts/create-container-registry-creds.sh "$K8S_NAMESPACE"

```

(NOTE: these requirements are true for automatic demo deployment using the available demo scripts; [#1416](https://github.com/flomesh-io/fsm/issues/1416) tracks an improvement to not strictly require these and use upstream images from official dockerhub registry if a user does not want/need changes on the code)

## Build and push FSM images

For development and/or testing locally compiled builds, pushing the local image to a container registry is still required. Several Makefile targets are available.

### Examples

Build and push all images:

```console
make docker-build
```

Build and push all images to a specific registry with a specific tag:

```console
make docker-build CTR_REGISTRY=myregistry CTR_TAG=mytag
```

Build all images and load them into the current docker instance, but do not push:

```console
make docker-build DOCKER_BUILDX_OUTPUT=type=docker
```

Build and push only the fsm-controller image. Similar targets exist for all FSM and demo images:

```console
make docker-build-fsm-controller
```

Build and push a particular image for multiple architectures:

```console
make docker-build-fsm-bootstrap DOCKER_BUILDX_PLATFORM=linux/amd64,linux/arm64
```

## Code Formatting

All Go source code is formatted with `goimports`. The version of `goimports`
used by this project is specified in `go.mod`. To ensure you have the same
version installed, run `go install -mod=readonly golang.org/x/tools/cmd/goimports`. It's recommended that you set your IDE or
other development tools to use `goimports`. Formatting is checked during CI by
the `bin/fmt` script.

## Testing your changes

The FSM repo has a few layers of tests:

- Unit tests
- End-to-end (e2e) tests
- Simulations

For tests in the FSM repo we have chosen to leverage the following:

- [Go testing](https://golang.org/pkg/testing/) for unit tests
- [Gomega](https://onsi.github.io/gomega/) and [Ginkgo](https://onsi.github.io/ginkgo/) frameworks for e2e tests

We follow Go's convention and add unit tests for the respective functions in files with the `_test.go` suffix. So if a function lives in a file `foo.go` we will write a unit test for it in the file `foo_test.go`.

Refer to a [unit test](/pkg/catalog/inbound_traffic_policies_test.go) and [e2e test](/tests/e2e/e2e_egress_policy_test.go) example should you need a starting point.

#### Unit Tests

The most rudimentary tests are the unit tests. We strive for test coverage above 80% where
this is pragmatic and possible.
Each newly added function should be accompanied by a unit test. Ideally, while working
on the FSM repository, we practice
[test-driven development](https://en.wikipedia.org/wiki/Test-driven_development),
and each change would be accompanied by a unit test.

To run all unit tests you can use the following `Makefile` target:

```bash
make go-test-coverage
```

You can run the tests exclusively for the package you are working on. For example the following command will
run only the tests in the package implementing FSM's
[Hashicorp Vault](https://www.vaultproject.io/) integration:

```bash
go test ./pkg/certificate/providers/vault/...
```

You can check the unit test coverage by using the `-cover` option:

```bash
go test -cover ./pkg/certificate/providers/vault/...
```

We have a dedicated tool for in-depth analysis of the unit-test code coverage:

```bash
./scripts/test-w-coverage.sh
```

Running the [test-w-coverage.sh](/scripts/test-w-coverage.sh) script will create
an HTML file with in-depth analysis of unit-test coverage per package, per
function, and it will even show lines of code that need work. Open the HTML
file this tool generates to understand how to improve test coverage:

```
open ./coverage/index.html
```

Once the file loads in your browser, scroll to the package you worked on to see current test coverage:

![package coverage](https://docs.flomesh.io/docs/images/unit-test-coverage-1.png)

Our overall guiding principle is to maintain unit-test coverage at or above 80%.

To understand which particular functions need more testing - scroll further in the report:

![per function](https://docs.flomesh.io/docs/images/unit-test-coverage-2.png)

And if you are wondering why a function, which we have written a test for, is not 100% covered,
you will find the per-function analysis useful. This will show you code paths that are not tested.

![per function](https://docs.flomesh.io/docs/images/unit-test-coverage-3.png)

##### Mocking

FSM uses the [GoMock](https://github.com/golang/mock) mocking framework to mock interfaces in unit tests.
GoMock's `mockgen` tool is used to autogenerate mocks from interfaces.

As an example, to create a mock client for the `Configurator` interface defined in the [configurator](/pkg/configurator) package, add the corresponding mock generation rule in the [rules file](/mockspec/rules) and generate the mocks using the command `make check-mocks` from the root of the FSM repo.

_Note: Autogenerated mock file names must be prefixed with `mock_`and suffixed with`_generated` as seen above. These files are excluded from code coverage reports._

When a mocked interface is changed, the autogenerated mock code must be regenerated.
More details can be found in [GoMock's documentation](https://github.com/golang/mock/blob/master/README.md).

#### Integration Tests

Unit tests focus on a single function. These ensure that with a specific input, the function
in question produces expected output or side effect. Integration tests, on the other hand,
ensure that multiple functions work together correctly. Integration tests ensure your new
code composes with other existing pieces.

Take a look at [the following test](/pkg/configurator/client_test.go),
which tests the functionality of multiple functions together. In this particular example, the test:

- uses a mock Kubernetes client via `testclient.NewSimpleClientset()` from the `github.com/flomesh-io/fsm/pkg/gen/client/config/clientset/versioned/fake` library
- [creates a MeshConfig](/pkg/configurator/client_test.go#L50)
- [tests whether](/pkg/configurator/client_test.go#L63-L69) the underlying functions compose correctly by fetching the results of the top-level function `IsEgressEnabled()`

### End-to-End (e2e) Tests

End-to-end tests verify the behavior of the entire system. For FSM, e2e tests will install a control plane, install test workloads and SMI policies, and check that the workload is behaving as expected.

FSM's e2e tests are located in tests/e2e. The tests can be run using the `test-e2e` Makefile target. The Makefile target will also build the necessary container images and `fsm` CLI binary before running the tests. The tests are written using Ginkgo and Gomega so they may also be directly invoked using `go test`. Be sure to build the `fsm-controller` and `init` container images and `fsm` CLI before directly invoking the tests. With either option, it is suggested to explicitly set the container registry location and tag to ensure up-to-date images are used by setting the `CTR_REGISTRY` and `CTR_TAG` environment variables.

In addition to the flags provided by `go test` and Ginkgo, there are several custom command line flags that may be used for e2e tests to configure global parameters like container image locations and cleanup behavior. The full list of custom flags can be found in [tests/e2e/](/tests/e2e#flags).

For more information, please refer to [FSM's E2E Readme](/tests/e2e/README.md).

#### Simulation / Demo

When we want to ensure that the entire system works correctly over time and
transitions state as expected - we run
[the demo included in the docs](/demo/README.md).
This type of test is the slowest, but also most comprehensive. This test will ensure that your changes
work with a real Kubernetes cluster, with real SMI policy, and real functions - no mocked or fake Go objects.

#### Profiling

FSM control plane exposes an HTTP server able to serve a number of resources.

For mesh visibility and debugabbility, one can refer to the endpoints provided under [pkg/debugger](/pkg/debugger) which contains a number of endpoints able to inspect and list most of the common structures used by the control plane at runtime.

Additionally, the current implementation of the debugger imports and hooks [pprof endpoints](https://golang.org/pkg/net/http/pprof/).
Pprof is a golang package able to provide profiling information at runtime through HTTP protocol to a connecting client.

Debugging endpoints can be turned on or off through the runtime argument `enable-debug-server`, normally set on the deployment at install time through the CLI.

Example usage:

```
scripts/port-forward-fsm-debug.sh &
go tool pprof http://localhost:9091/debug/pprof/heap
```

From pprof tool, it is possible to extract a large variety of profiling information, from heap and cpu profiling, to goroutine blocking, mutex profiling or execution tracing. We suggest to refer to the [pprof documentation](https://golang.org/pkg/net/http/pprof/) for more information.

## Helm charts

The Flomesh Service Mesh control plane chart is located in the
[`charts/fsm`](/charts/fsm) folder.

The [`charts/fsm/values.yaml`](/charts/fsm/values.yaml) file defines the default value for properties
referenced by the different chart templates.

The [`charts/fsm/templates/`](/charts/fsm/templates) folder contains the chart templates
for the different Kubernetes resources that are deployed as a part of the FSM control plane installation.
The different chart templates are used as follows:

- `fsm-*.yaml` chart templates are directly consumed by the `fsm-controller` service.
- `mutatingwebhook.yaml` is used to deploy a `MutatingWebhookConfiguration` kubernetes resource that enables automatic sidecar injection
- `grafana-*.yaml` chart templates are used to deploy a Grafana instance when grafana installation is enabled
- `prometheus-*.yaml` chart templates are used to deploy a Prometheus instance when prometheus installation is enabled
- `fluentbit-configmap.yaml` is used to provide configurations for the fluent bit sidecar and its plugins when fluent bit is enabled
- `jaeger-*.yaml` chart templates are used to deploy a Jaeger instance when Jaeger deployment and tracing are enabled

### Custom Resource Definitions

The [`charts/fsm/crds/`](/charts/fsm/crds/) folder contains the charts corresponding to the SMI CRDs.
Experimental CRDs can be found under [`charts/fsm/crds/experimental/`](/charts/fsm/crds/experimental).

### Updating Dependencies

Dependencies for the FSM chart are listed in Chart.yaml. To update a dependency,
modify its version as needed in Chart.yaml, run `helm dependency update`, then
commit all changes to Chart.yaml, Chart.lock, and the charts/fsm/charts
directory which stores the source for the updated dependency chart.
