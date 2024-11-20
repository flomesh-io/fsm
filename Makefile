#!make

TARGETS      := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
BINNAME      ?= fsm
DIST_DIRS    := find * -type d -exec
CTR_REGISTRY ?= flomesh
CTR_TAG      ?= latest
VERIFY_TAGS  ?= false
DISTROLESS_TAG ?= nonroot

GOPATH = $(shell go env GOPATH)
GOBIN  = $(GOPATH)/bin
GOX    = go run github.com/mitchellh/gox
SHA256 = sha256sum
ifeq ($(shell uname),Darwin)
	SHA256 = shasum -a 256
endif

BUILD_DIR = bin

VERSION ?= dev
BUILD_DATE ?= $(shell date +%Y-%m-%d-%H:%M-%Z)
GIT_SHA=$$(git rev-parse HEAD)
BUILD_DATE_VAR := github.com/flomesh-io/fsm/pkg/version.BuildDate
BUILD_VERSION_VAR := github.com/flomesh-io/fsm/pkg/version.Version
BUILD_GITCOMMIT_VAR := github.com/flomesh-io/fsm/pkg/version.GitCommit
DOCKER_GO_VERSION = 1.23
DOCKER_BUILDX_PLATFORM ?= linux/amd64
# Value for the --output flag on docker buildx build.
# https://docs.docker.com/engine/reference/commandline/buildx_build/#output
DOCKER_BUILDX_OUTPUT ?= type=registry

LDFLAGS ?= "-X $(BUILD_DATE_VAR)=$(BUILD_DATE) -X $(BUILD_VERSION_VAR)=$(VERSION) -X $(BUILD_GITCOMMIT_VAR)=$(GIT_SHA) -s -w"

# These two values are combined and passed to go test
E2E_FLAGS ?= -installType=KindCluster
E2E_FLAGS_DEFAULT := -test.v -ginkgo.v -ginkgo.progress -ctrRegistry $(CTR_REGISTRY) -fsmImageTag $(CTR_TAG)

# Installed Go version
# This is the version of Go going to be used to compile this project.
# It will be compared with the minimum requirements for FSM.
GO_VERSION_MAJOR = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f1)
GO_VERSION_MINOR = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f2)
GO_VERSION_PATCH = $(shell go version | cut -c 14- | cut -d' ' -f1 | cut -d'.' -f3)
ifeq ($(GO_VERSION_PATCH),)
GO_VERSION_PATCH := 0
endif

export CHART_COMPONENTS_DIR = charts/fsm/components
export SCRIPTS_TAR = $(CHART_COMPONENTS_DIR)/scripts.tar.gz

check-env:
ifndef CTR_REGISTRY
	$(error CTR_REGISTRY environment variable is not defined; see the .env.example file for more information; then source .env)
endif
ifndef CTR_TAG
	$(error CTR_TAG environment variable is not defined; see the .env.example file for more information; then source .env)
endif

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd:allowDangerousTypes=true paths="./pkg/apis/..." output:crd:artifacts:config=cmd/fsm-bootstrap/crds

.PHONY: labels
labels: kustomize ## Attach required labels to gateway-api resources
	$(KUSTOMIZE) build cmd/fsm-bootstrap/raw/ -o cmd/fsm-bootstrap/crds/

.PHONY: build
build: charts-tgz manifests go-fmt go-vet ## Build commands with release args, the result will be optimized.
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -v -o $(BUILD_DIR) -ldflags ${LDFLAGS} ./cmd/{fsm-bootstrap,fsm-connector,fsm-controller,fsm-gateway,fsm-healthcheck,fsm-ingress,fsm-injector,fsm-xnetmgmt,fsm-preinstall}

.PHONY: build-fsm
build-fsm: helm-update-dep cmd/cli/chart.tgz
	CGO_ENABLED=0 go build -v -o ./bin/fsm -ldflags ${LDFLAGS} ./cmd/cli

cmd/cli/chart.tgz: scripts/generate_chart/generate_chart.go $(shell find charts/fsm)
	go run $< --chart-name=fsm > $@

pkg/controllers/namespacedingress/v1alpha1/chart.tgz: scripts/generate_chart/generate_chart.go $(shell find charts/namespaced-ingress)
	go run $< --chart-name=namespaced-ingress > $@

pkg/controllers/gateway/v1/chart.tgz: scripts/generate_chart/generate_chart.go $(shell find charts/gateway)
	go run $< --chart-name=gateway > $@

pkg/controllers/connector/v1alpha1/chart.tgz: scripts/generate_chart/generate_chart.go $(shell find charts/connector)
	go run $< --chart-name=connector > $@

helm-update-dep: helm
	$(HELM) dependency update charts/fsm/
	$(HELM) dependency update charts/gateway/
	$(HELM) dependency update charts/namespaced-ingress/
	$(HELM) dependency update charts/connector/

.PHONY: package-scripts
package-scripts:
	find $(CHART_COMPONENTS_DIR) -type f -name '._*' -exec echo "deleting: {}" \; -delete
	find $(CHART_COMPONENTS_DIR) -type f -name '.DS_Store' -exec echo "deleting: {}" \; -delete
	## Tar all repo initializing scripts
	tar --no-xattrs -C $(CHART_COMPONENTS_DIR)/ --exclude='.DS_Store' --exclude='._*' -zcvf $(SCRIPTS_TAR) scripts/

.PHONY: charts-tgz
charts-tgz: pkg/controllers/namespacedingress/v1alpha1/chart.tgz pkg/controllers/gateway/v1/chart.tgz pkg/controllers/connector/v1alpha1/chart.tgz

.PHONY: clean-fsm
clean-fsm:
	@rm -rf bin/fsm

.PHONY: codegen
codegen:
	./codegen/gen-crd-client.sh

.PHONY: chart-readme
chart-readme:
	go run github.com/norwoodj/helm-docs/cmd/helm-docs -c charts -t charts/fsm/README.md.gotmpl

.PHONY: chart-check-readme
chart-check-readme: chart-readme
	@git diff --exit-code charts/fsm/README.md || { echo "----- Please commit the changes made by 'make chart-readme' -----"; exit 1; }
	@git diff --exit-code charts/gateway/README.md || { echo "----- Please commit the changes made by 'make chart-readme' -----"; exit 1; }
	@git diff --exit-code charts/namespaced-ingress/README.md || { echo "----- Please commit the changes made by 'make chart-readme' -----"; exit 1; }
	@git diff --exit-code charts/connector/README.md || { echo "----- Please commit the changes made by 'make chart-readme' -----"; exit 1; }

.PHONY: helm-lint
helm-lint:
	@helm lint charts/fsm/ || { echo "----- Schema validation failed for FSM chart values -----"; exit 1; }

.PHONY: chart-checks
chart-checks: chart-check-readme helm-lint

.PHONY: check-mocks
check-mocks:
	@go run ./mockspec/generate.go
	@git diff --exit-code || { echo "----- Please commit the changes made by 'go run ./mockspec/generate.go' -----"; exit 1; }

.PHONY: check-codegen
check-codegen:
	@./codegen/gen-crd-client.sh
	@git diff --exit-code -- ':!go.mod' ':!go.sum' || { echo "----- Please commit the changes made by './codegen/gen-crd-client.sh' -----"; exit 1; }

.PHONY: check-scripts
check-scripts:
	./scripts/check-scripts.sh

.PHONY: check-manifests
check-manifests:
	@git diff --exit-code cmd/fsm-bootstrap/crds/ || { echo "----- Please commit the changes made by 'make manifests' -----"; exit 1; }

.PHONY: go-checks
go-checks: go-lint go-fmt go-mod-tidy check-mocks check-codegen

.PHONY: go-vet
go-vet:
	go vet ./...

.PHONY: go-lint
go-lint: embed-files-test
	docker run --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.55 golangci-lint run --config .golangci.yml

.PHONY: go-fmt
go-fmt:
	go fmt ./...

.PHONY: go-mod-tidy
go-mod-tidy:
	./scripts/go-mod-tidy.sh

.PHONY: go-test
go-test: helm-update-dep cmd/cli/chart.tgz
	./scripts/go-test.sh

.PHONY: go-test-coverage
go-test-coverage: embed-files
	./scripts/test-w-coverage.sh

.PHONY: go-benchmark
go-benchmark: embed-files
	./scripts/go-benchmark.sh

lint-c:
	clang-format --Werror -n bpf/*.c bpf/headers/*.h

format-c:
	find . -regex '.*\.\(c\|h\)' -exec clang-format -style=file -i {} \;

.PHONY: kind-up
kind-up:
	./scripts/kind-with-registry.sh

.PHONY: kind-reset
kind-reset:
	kind delete cluster --name fsm

.PHONY: test-e2e
test-e2e: DOCKER_BUILDX_OUTPUT=type=docker
test-e2e: docker-build-fsm build-fsm docker-build-tcp-echo-server
	E2E_FLAGS="--timeout=0" go test ./tests/e2e $(E2E_FLAGS_DEFAULT) $(E2E_FLAGS)

.env:
	cp .env.example .env

.PHONY: kind-demo
kind-demo: export CTR_REGISTRY=localhost:5000
kind-demo: .env kind-up clean-fsm
	./demo/run-fsm-demo.sh

.PHONE: build-bookwatcher
build-bookwatcher:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./demo/bin/bookwatcher/bookwatcher ./demo/cmd/bookwatcher

DEMO_TARGETS = bookbuyer bookthief bookstore bookwarehouse tcp-echo-server tcp-client
# docker-build-bookbuyer, etc
DOCKER_DEMO_TARGETS = $(addprefix docker-build-, $(DEMO_TARGETS))
.PHONY: $(DOCKER_DEMO_TARGETS)
$(DOCKER_DEMO_TARGETS): NAME=$(@:docker-build-%=%)
$(DOCKER_DEMO_TARGETS):
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-demo-$(NAME):$(CTR_TAG) -f dockerfiles/Dockerfile.demo --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg BINARY=$(NAME) .

.PHONY: docker-build-demo
docker-build-demo: $(DOCKER_DEMO_TARGETS)

.PHONY: docker-build-fsm-curl
docker-build-fsm-curl:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-curl:$(CTR_TAG) - < dockerfiles/Dockerfile.fsm-curl

.PHONY: docker-build-fsm-sidecar-init
docker-build-fsm-sidecar-init:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-sidecar-init:$(CTR_TAG) - < dockerfiles/Dockerfile.fsm-sidecar-init

.PHONY: docker-build-fsm-controller
docker-build-fsm-controller:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-controller:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-controller --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-fsm-injector
docker-build-fsm-injector:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-injector:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-injector --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-fsm-crds
docker-build-fsm-crds:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-crds:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-crds .

.PHONY: docker-build-fsm-bootstrap
docker-build-fsm-bootstrap:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-bootstrap:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-bootstrap --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-fsm-preinstall
docker-build-fsm-preinstall:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-preinstall:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-preinstall --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-fsm-healthcheck
docker-build-fsm-healthcheck:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-healthcheck:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-healthcheck --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-fsm-xnetmgmt
docker-build-fsm-xnetmgmt:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-xnetmgmt:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-xnetmgmt --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-fsm-connector
docker-build-fsm-connector:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-connector:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-connector --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) .

.PHONY: docker-build-fsm-ingress
docker-build-fsm-ingress:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-ingress:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-ingress --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) --build-arg DISTROLESS_TAG=$(DISTROLESS_TAG) .

.PHONY: docker-build-fsm-gateway
docker-build-fsm-gateway:
	docker buildx build --builder fsm --platform=$(DOCKER_BUILDX_PLATFORM) -o $(DOCKER_BUILDX_OUTPUT) -t $(CTR_REGISTRY)/fsm-gateway:$(CTR_TAG) -f dockerfiles/Dockerfile.fsm-gateway --build-arg GO_VERSION=$(DOCKER_GO_VERSION) --build-arg LDFLAGS=$(LDFLAGS) --build-arg DISTROLESS_TAG=$(DISTROLESS_TAG) .

FSM_TARGETS = fsm-curl fsm-sidecar-init fsm-controller fsm-injector fsm-crds fsm-bootstrap fsm-preinstall fsm-healthcheck fsm-connector fsm-xnetmgmt fsm-ingress fsm-gateway

DOCKER_FSM_TARGETS = $(addprefix docker-build-, $(FSM_TARGETS))

.PHONY: docker-build-fsm
docker-build-fsm: charts-tgz $(DOCKER_FSM_TARGETS)

.PHONY: buildx-context
buildx-context:
	@if ! docker buildx ls | grep -q "^fsm"; then docker buildx create --name fsm --driver-opt network=host; fi

check-image-exists-%: NAME=$(@:check-image-exists-%=%)
check-image-exists-%:
	@if [ "$(VERIFY_TAGS)" = "true" ]; then scripts/image-exists.sh $(CTR_REGISTRY)/$(NAME):$(CTR_TAG); fi

$(foreach target,$(FSM_TARGETS) $(DEMO_TARGETS),$(eval docker-build-$(target): check-image-exists-$(target) buildx-context))

docker-digest-%: NAME=$(@:docker-digest-%=%)
docker-digest-%:
	@docker buildx imagetools inspect $(CTR_REGISTRY)/$(NAME):$(CTR_TAG) --raw | $(SHA256) | awk '{print "$(NAME): sha256:"$$1}'

.PHONY: docker-digests-fsm
docker-digests-fsm: $(addprefix docker-digest-, $(FSM_TARGETS))

.PHONY: docker-build
docker-build: docker-build-fsm docker-build-demo

.PHONY: docker-build-cross-fsm docker-build-cross-demo docker-build-cross
docker-build-cross-fsm: DOCKER_BUILDX_PLATFORM=linux/amd64,linux/arm64
docker-build-cross-fsm: docker-build-fsm
docker-build-cross-demo: DOCKER_BUILDX_PLATFORM=linux/amd64,linux/arm64
docker-build-cross-demo: docker-build-demo
docker-build-cross: docker-build-cross-fsm docker-build-cross-demo

.PHONY: embed-files
embed-files: helm-update-dep cmd/cli/chart.tgz charts-tgz

.PHONY: embed-files-test
embed-files-test:
	./scripts/generate-dummy-embed.sh

.PHONY: build-ci
build-ci: embed-files
	go build -v ./...

.PHONY: trivy-ci-setup
trivy-ci-setup:
	wget https://github.com/aquasecurity/trivy/releases/download/v0.57.0/trivy_0.57.0_Linux-64bit.tar.gz
	tar zxvf trivy_0.57.0_Linux-64bit.tar.gz
	echo $$(pwd) >> $(GITHUB_PATH)

# Show all vulnerabilities in logs
trivy-scan-verbose-%: NAME=$(@:trivy-scan-verbose-%=%)
trivy-scan-verbose-%:
	trivy image --scanners vuln,secret \
	  --pkg-types os \
	  --db-repository aquasec/trivy-db:2 \
	  "$(CTR_REGISTRY)/$(NAME):$(CTR_TAG)"

# Exit if vulnerability exists
trivy-scan-fail-%: NAME=$(@:trivy-scan-fail-%=%)
trivy-scan-fail-%:
	trivy image --exit-code 1 \
	  --ignore-unfixed \
	  --severity MEDIUM,HIGH,CRITICAL \
	  --dependency-tree \
	  --scanners vuln,secret \
	  --pkg-types os \
	  --db-repository aquasec/trivy-db:2 \
	  "$(CTR_REGISTRY)/$(NAME):$(CTR_TAG)"

.PHONY: trivy-scan-images trivy-scan-images-fail trivy-scan-images-verbose
trivy-scan-images-verbose: $(addprefix trivy-scan-verbose-, $(FSM_TARGETS))
trivy-scan-images-fail: $(addprefix trivy-scan-fail-, $(FSM_TARGETS))
trivy-scan-images: trivy-scan-images-verbose trivy-scan-images-fail

.PHONY: shellcheck
shellcheck:
	shellcheck -x $(shell find . -name '*.sh')

.PHONY: install-git-pre-push-hook
install-git-pre-push-hook:
	./scripts/install-git-pre-push-hook.sh

# -------------------------------------------
#  release targets below
# -------------------------------------------
##@ Release Targets

.PHONY: generate-cli-chart
generate-cli-chart: helm-update-dep cmd/cli/chart.tgz

.PHONY: build-cross
build-cross: helm-update-dep cmd/cli/chart.tgz
	GO111MODULE=on CGO_ENABLED=0 $(GOX) -ldflags $(LDFLAGS) -parallel=5 -output="_dist/{{.OS}}-{{.Arch}}/$(BINNAME)" -osarch='$(TARGETS)' ./cmd/cli

.PHONY: dist
dist:
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf fsm-${VERSION}-{}.tar.gz {} \; && \
		$(DIST_DIRS) zip -r fsm-${VERSION}-{}.zip {} \; && \
		$(SHA256) fsm-* > sha256sums.txt \
	)

.PHONY: release-artifacts
release-artifacts: build-cross dist

.PHONY: release
VERSION_REGEXP := ^v[0-9]+\.[0-9]+\.[0-9]+(\-(alpha|beta|rc)\.[0-9]+)?$
release: ## Create a release tag, push to git repository and trigger the release workflow.
ifeq (,$(RELEASE_VERSION))
	$(error "RELEASE_VERSION must be set to tag HEAD")
endif
ifeq (,$(shell [[ "$(RELEASE_VERSION)" =~ $(VERSION_REGEXP) ]] && echo 1))
	$(error "Version $(RELEASE_VERSION) must match regexp $(VERSION_REGEXP)")
endif
	git tag --sign --message "fsm $(RELEASE_VERSION)" $(RELEASE_VERSION)
	git verify-tag --verbose $(RELEASE_VERSION)
	git push origin --tags

# -------------------------------------------
#  Build Dependencies below
# -------------------------------------------
##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
HELM ?= $(LOCALBIN)/helm
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.4.3
HELM_VERSION ?= v3.16.2
CONTROLLER_TOOLS_VERSION ?= v0.16.5

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	[ -f $(KUSTOMIZE) ] || curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

HELM_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3"
.PHONY: helm
helm: $(HELM) ## Download kustomize locally if necessary.
$(HELM): $(LOCALBIN)
	[ -f $(HELM) ] || curl -s $(HELM_INSTALL_SCRIPT) | HELM_INSTALL_DIR=$(LOCALBIN) bash -s -- --version $(HELM_VERSION) --no-sudo

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	[ -f $(CONTROLLER_GEN) ] || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	[ -f $(ENVTEST) ] || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

