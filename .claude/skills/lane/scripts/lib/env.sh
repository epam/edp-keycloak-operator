# Repo constants for the lane skill. The hook .claude/hooks/lane-allow.py parses this file:
# keep every value as `NAME=value` or `NAME=(a b c)` on one line.

CONTEXT=kind-krci
KIND_CLUSTER=krci
LABEL=lane/item
IMAGE_REPO=repo/edp/keycloak-image
RELEASE_PREFIX=keycloak-operator-
DEPLOY_NAME=keycloak-operator
CHART=deploy-templates
CRD_DIR=deploy-templates/crds
CRD_GROUP=v1.edp.epam.com
# A change under one of these paths makes the item `scope: crd`.
CRD_PATHS=(api/ deploy-templates/crds/ config/crd/ internal/webhook/)
INSTALL_FLAGS=(--skip-crds --set clusterReconciliationEnabled=false)
READY_LOG='Starting workers'
TOOLS=(golangci-lint setup-envtest controller-gen mockery kustomize crdoc helmdocs)
TEST_URL_VAR=TEST_KEYCLOAK_URL

# Shared Keycloak on the testbed.
KC_NS=keycloak
KC_SVC=keycloak
KC_PORT=8081
KC_SECRET=keycloak-secret
KC_DEPLOY=keycloak
KC_CONTAINER=keycloak
KC_MANIFEST=tests/e2e/helm-success-path/01-install-keycloak-server.yaml

# Namespaces an item never deletes in. Deployments that must be ready before intake.
PROTECTED_NS=(keycloak cert-manager krci argocd)
TESTBED_DEPLOYS=(keycloak/keycloak cert-manager/cert-manager)

# Make targets that touch the cluster, a registry or the network: always a permission prompt.
PROMPT_TARGETS=(start-kind delete-kind e2e bundle bundle-build bundle-push docker-push install generate-keycloak-go-client start-keycloak delete-keycloak)

# Namespaced CR kinds, deleted in this order at teardown (children before realms, realms before Keycloak).
CR_DELETE_ORDER=(keycloakrealmusers keycloakrealmgroups keycloakrealmroles keycloakrealmrolebatches keycloakclientscopes keycloakclients keycloakrealmidentityproviders keycloakauthflows keycloakrealmcomponents keycloakorganizations keycloakrealms keycloaks)

WORK_DIR=.work
LOCK_FILE=.work/.crd-lock

# Required plugins (installed_plugins.json keys), repo skills and commands.
DEPS_PLUGINS=(krci-general krci-godev krci-triage mattpocock-skills)
DEPS_SKILLS=(cr-field)
DEPS_CMDS=(git gh kubectl helm docker kind go make jq python3 curl)
