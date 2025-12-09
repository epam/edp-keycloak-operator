PACKAGE=github.com/epam/edp-common/pkg/config
CURRENT_DIR=$(shell pwd)
DIST_DIR=${CURRENT_DIR}/dist
BIN_NAME=manager

HOST_OS?=$(shell go env GOOS)
HOST_ARCH?=$(shell go env GOARCH)

VERSION?=$(shell git describe --tags)
BUILD_DATE=$(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_TAG=$(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
KUBECTL_VERSION=$(shell go list -m all | grep k8s.io/client-go| cut -d' ' -f2)
## Location to install dependencies to
LOCALBIN ?= ${CURRENT_DIR}/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Use kind cluster for testing
CONTAINER_REGISTRY_URL?="repo"
CONTAINER_REGISTRY_SPACE?="edp"
START_KIND_CLUSTER?=true
KIND_CLUSTER_NAME?="keycloak-operator"
KUBE_VERSION?=1.33
KIND_CONFIG?=./hack/kind-$(KUBE_VERSION).yaml

E2E_IMAGE_REPOSITORY?="keycloak-image"
E2E_IMAGE_TAG?="latest"

TEST_KEYCLOAK_URL?=""

override LDFLAGS += \
  -X ${PACKAGE}.version=${VERSION} \
  -X ${PACKAGE}.buildDate=${BUILD_DATE} \
  -X ${PACKAGE}.gitCommit=${GIT_COMMIT} \
  -X ${PACKAGE}.kubectlVersion=${KUBECTL_VERSION}

ifneq (${GIT_TAG},)
LDFLAGS += -X ${PACKAGE}.gitTag=${GIT_TAG}
endif

override GCFLAGS +=all=-trimpath=${CURRENT_DIR}

# Image URL to use all building/pushing image targets
IMG ?= docker.io/epamedp/keycloak-operator:$(VERSION)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# epamedp/keycloak-operator-bundle:$VERSION and epamedp/keycloak-operator-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= epamedp/keycloak-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

.DEFAULT_GOAL:=help
# set default shell
SHELL=/bin/bash -o pipefail -o errexit
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=deploy-templates/crds
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(MAKE) api-docs

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: validate-docs
validate-docs: api-docs helm-docs  ## Validate helm and api docs
	@git diff -s --exit-code deploy-templates/README.md || (echo "Run 'make helm-docs' to address the issue." && git diff && exit 1)
	@git diff -s --exit-code docs/api.md || (echo " Run 'make api-docs' to address the issue." && git diff && exit 1)

# Run tests
test: setup-envtest
	@if [ -z "$(TEST_KEYCLOAK_URL)" ]; then \
		echo ""; \
		echo "WARNING: TEST_KEYCLOAK_URL is not specified, integration tests will be skipped."; \
	fi
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	TEST_KEYCLOAK_URL=${TEST_KEYCLOAK_URL} \
	go test ./... -coverprofile=coverage.out `go list ./...`

## Run e2e tests. Requires kind with running cluster and kuttl tool.
e2e: build-linux-amd64
	docker build --no-cache --platform linux/amd64 -t ${CONTAINER_REGISTRY_URL}/${CONTAINER_REGISTRY_SPACE}/${E2E_IMAGE_REPOSITORY}:${E2E_IMAGE_TAG} .
	kind load --name $(KIND_CLUSTER_NAME) docker-image ${CONTAINER_REGISTRY_URL}/${CONTAINER_REGISTRY_SPACE}/${E2E_IMAGE_REPOSITORY}:${E2E_IMAGE_TAG}
	E2E_IMAGE_REPOSITORY=${E2E_IMAGE_REPOSITORY} CONTAINER_REGISTRY_URL=${CONTAINER_REGISTRY_URL} CONTAINER_REGISTRY_SPACE=${CONTAINER_REGISTRY_SPACE} E2E_IMAGE_TAG=${E2E_IMAGE_TAG} kubectl-kuttl test

.PHONY: fmt
fmt:  ## Run go fmt
	go fmt ./...

.PHONY: vet
vet:  ## Run go vet
	go vet ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

.PHONY: build
build:  ## build operator's binary
	CGO_ENABLED=0 GOOS=${HOST_OS} GOARCH=${HOST_ARCH} go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${BIN_NAME}-${HOST_ARCH} -gcflags '${GCFLAGS}' ./cmd

.PHONY: build-linux-amd64
build-linux-amd64:  ## build operator's binary for Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${BIN_NAME}-amd64 -gcflags '${GCFLAGS}' ./cmd

.PHONY: clean
clean:  ## clean up
	-rm -rf ${DIST_DIR}

# use https://github.com/git-chglog/git-chglog/
.PHONY: changelog
changelog: git-chglog	## generate changelog
ifneq (${NEXT_RELEASE_TAG},)
	$(GITCHGLOG) --next-tag v${NEXT_RELEASE_TAG} -o CHANGELOG.md v1.17.0..
else
	$(GITCHGLOG) -o CHANGELOG.md v1.17.0..
endif

.PHONY: api-docs
api-docs: crdoc	## generate CRD docs
	$(CRDOC) --resources deploy-templates/crds --output docs/api.md

.PHONY: helm-docs
helm-docs: helmdocs	## generate helm docs
	$(HELMDOCS)

GOLANGCI_LINT = ${CURRENT_DIR}/bin/golangci-lint
.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

##@ Build Dependencies

## Tool Versions
KUSTOMIZE_VERSION ?= v5.6.0
CONTROLLER_TOOLS_VERSION ?= v0.18.0
ENVTEST_VERSION := $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
ENVTEST_K8S_VERSION := $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
GOLANGCI_LINT_VERSION ?= v2.1.6
MOCKERY_VERSION ?= v3.5.4
HELMDOCS_VERSION ?= v1.14.2
GITCHGLOG_VERSION ?= v0.15.4
CRDOC_VERSION ?= v0.6.4
OPERATOR_SDK_VERSION ?= v1.41.1

KUSTOMIZE ?= $(LOCALBIN)/kustomize
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

HELMDOCS = $(LOCALBIN)/helm-docs
.PHONY: helmdocs
helmdocs: ## Download helm-docs locally if necessary.
	$(call go-install-tool,$(HELMDOCS),github.com/norwoodj/helm-docs/cmd/helm-docs,$(HELMDOCS_VERSION))

GITCHGLOG = $(LOCALBIN)/git-chglog
.PHONY: git-chglog
git-chglog: ## Download git-chglog locally if necessary.
	$(call go-install-tool,$(GITCHGLOG),github.com/git-chglog/git-chglog/cmd/git-chglog,$(GITCHGLOG_VERSION))

CRDOC = $(LOCALBIN)/crdoc
.PHONY: crdoc
crdoc: ## Download crdoc locally if necessary.
	$(call go-install-tool,$(CRDOC),fybrik.io/crdoc,$(CRDOC_VERSION))

CONTROLLER_GEN = $(LOCALBIN)/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

.PHONY: bundle
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	# Add OpenShift version to bundle.Dockerfile and bundle/metadata/annotations.yaml manually because operator-sdk cleans any additional labels.
	echo "" >> bundle.Dockerfile
	echo "LABEL com.redhat.openshift.versions=v4.7-v4.19" >> bundle.Dockerfile
	echo "" >> bundle/metadata/annotations.yaml
	echo "  com.redhat.openshift.versions: v4.7-v4.19" >> bundle/metadata/annotations.yaml
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

ENVTEST ?= $(LOCALBIN)/setup-envtest
.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: start-kind
start-kind:	## Start kind cluster
ifeq (true,$(START_KIND_CLUSTER))
	kind create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG)
endif

.PHONY: delete-kind
delete-kind:	## Delete kind cluster
	kind delete cluster --name $(KIND_CLUSTER_NAME) || true

mocks: mockery
	$(MOCKERY)

MOCKERY = $(LOCALBIN)/mockery
.PHONY: mockery
mockery: ## Download mockery locally if necessary.
	$(call go-install-tool,$(MOCKERY),github.com/vektra/mockery/v3,$(MOCKERY_VERSION))

.PHONY: operator-sdk
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
operator-sdk: ## Download operator-sdk locally if necessary.
ifeq (,$(wildcard $(OPERATOR_SDK)))
ifeq (, $(shell which operator-sdk 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPERATOR_SDK)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)/operator-sdk_$${OS}_$${ARCH} ;\
	chmod +x $(OPERATOR_SDK) ;\
	}
else
OPERATOR_SDK = $(shell which operator-sdk)
endif
endif

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

.PHONY: start-keycloak
start-keycloak: ## Start Keycloak instance for testing
	docker run -d -p 8086:8080 -e KEYCLOAK_ADMIN=admin -e KEYCLOAK_ADMIN_PASSWORD=admin -e KC_FEATURES=admin-fine-grained-authz:v1 --name keycloak-test quay.io/keycloak/keycloak:latest start-dev

.PHONY: delete-keycloak
delete-keycloak: ## Stop Keycloak test instance
	docker stop keycloak-test || true
	docker rm keycloak-test || true
