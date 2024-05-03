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
ENVTEST ?= $(LOCALBIN)/setup-envtest
ENVTEST_K8S_VERSION = 1.23.5

# Use kind cluster for testing
CONTAINER_REGISTRY_URL?="repo"
CONTAINER_REGISTRY_SPACE?="edp"
START_KIND_CLUSTER?=true
KIND_CLUSTER_NAME?="keycloak-operator"
KUBE_VERSION?=1.26
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

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: validate-docs
validate-docs: api-docs helm-docs  ## Validate helm and api docs
	@git diff -s --exit-code deploy-templates/README.md || (echo "Run 'make helm-docs' to address the issue." && git diff && exit 1)
	@git diff -s --exit-code docs/api.md || (echo " Run 'make api-docs' to address the issue." && git diff && exit 1)

# Run tests
test: fmt vet envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) --arch=amd64 use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	TEST_KEYCLOAK_URL=${TEST_KEYCLOAK_URL} \
	go test ./... -coverprofile=coverage.out `go list ./...`

## Run e2e tests. Requires kind with running cluster and kuttl tool.
e2e: build
	docker build --no-cache -t ${CONTAINER_REGISTRY_URL}/${CONTAINER_REGISTRY_SPACE}/${E2E_IMAGE_REPOSITORY}:${E2E_IMAGE_TAG} .
	kind load --name $(KIND_CLUSTER_NAME) docker-image ${CONTAINER_REGISTRY_URL}/${CONTAINER_REGISTRY_SPACE}/${E2E_IMAGE_REPOSITORY}:${E2E_IMAGE_TAG}
	E2E_IMAGE_REPOSITORY=${E2E_IMAGE_REPOSITORY} CONTAINER_REGISTRY_URL=${CONTAINER_REGISTRY_URL} CONTAINER_REGISTRY_SPACE=${CONTAINER_REGISTRY_SPACE} E2E_IMAGE_TAG=${E2E_IMAGE_TAG} kubectl-kuttl test

.PHONY: fmt
fmt:  ## Run go fmt
	go fmt ./...

.PHONY: vet
vet:  ## Run go vet
	go vet ./...

lint: golangci-lint ## Run go lint
	${GOLANGCILINT} run

.PHONY: build
build: clean ## build operator's binary
	CGO_ENABLED=0 GOOS=${HOST_OS} GOARCH=${HOST_ARCH} go build -v -ldflags '${LDFLAGS}' -o ${DIST_DIR}/${BIN_NAME} -gcflags '${GCFLAGS}' .

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

GOLANGCILINT = ${CURRENT_DIR}/bin/golangci-lint
.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary.
	$(call go-get-tool,$(GOLANGCILINT),github.com/golangci/golangci-lint/cmd/golangci-lint,v1.55.2)

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

##@ Build Dependencies

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.5

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

HELMDOCS = $(LOCALBIN)/helm-docs
.PHONY: helmdocs
helmdocs: ## Download helm-docs locally if necessary.
	$(call go-get-tool,$(HELMDOCS),github.com/norwoodj/helm-docs/cmd/helm-docs,v1.11.0)

GITCHGLOG = $(LOCALBIN)/git-chglog
.PHONY: git-chglog
git-chglog: ## Download git-chglog locally if necessary.
	$(call go-get-tool,$(GITCHGLOG),github.com/git-chglog/git-chglog/cmd/git-chglog,v0.15.4)

CRDOC = $(LOCALBIN)/crdoc
.PHONY: crdoc
crdoc: ## Download crdoc locally if necessary.
	$(call go-get-tool,$(CRDOC),fybrik.io/crdoc,v0.6.1)

CONTROLLER_GEN = $(LOCALBIN)/controller-gen
.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,v0.11.3)
# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
go get -d $(2)@$(3) ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle $(BUNDLE_GEN_FLAGS)
	operator-sdk bundle validate ./bundle

ENVTEST=$(LOCALBIN)/setup-envtest
.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-get-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,release-0.16)

.PHONY: start-kind
start-kind:	## Start kind cluster
ifeq (true,$(START_KIND_CLUSTER))
	kind create cluster --name $(KIND_CLUSTER_NAME) --config $(KIND_CONFIG)
endif

mocks: mockery
	$(MOCKERY)

MOCKERY = $(LOCALBIN)/mockery
.PHONY: mockery
mockery: ## Download mockery locally if necessary.
	$(call go-get-tool,$(MOCKERY),github.com/vektra/mockery/v2,v2.38.0)
