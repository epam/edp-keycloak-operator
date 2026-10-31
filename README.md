[![Build Status](https://github.com/epam/edp-keycloak-operator/actions/workflows/codecov.yaml/badge.svg)](https://github.com/epam/edp-keycloak-operator/actions/workflows/codecov.yaml)
[![codecov](https://codecov.io/gh/epam/edp-keycloak-operator/branch/master/graph/badge.svg?token=WJ7YFRPUX2)](https://codecov.io/gh/epam/edp-keycloak-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/epam/edp-keycloak-operator)](https://goreportcard.com/report/github.com/epam/edp-keycloak-operator)
[![Go Reference](https://pkg.go.dev/badge/github.com/epam/edp-keycloak-operator.svg)](https://pkg.go.dev/github.com/epam/edp-keycloak-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# Keycloak Operator

| Please refer to [KubeRocketCI documentation](https://docs.kuberocketci.io/) to get the main concepts and guidelines. |
| --- |

Get acquainted with the Keycloak Operator, the installation process, the quick start, and the local development guidelines.

## Overview

Keycloak Operator is a KubeRocketCI operator responsible for configuring existing Keycloak instances. The operator runs both on OpenShift and Kubernetes.

_**NOTE:** Operator is platform-independent, which is why there is a unified instruction for deployment._

## Prerequisites

1. Kubernetes 1.32 or later, or OpenShift 4.19 or later;
2. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
3. Cluster admin access to the cluster;
4. [cert-manager](https://cert-manager.io/docs/installation/) installed in the cluster (required for webhook functionality, can be disabled via `enableWebhooks: false`);

## Installation Using Helm Chart

To install the Keycloak Operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for a local client, run "helm repo add":

     ```bash
     helm repo add epamedp https://epam.github.io/edp-helm-charts/stable
     ```

2. Choose the available Helm chart version:

     ```bash
     helm search repo epamedp/keycloak-operator -l
     NAME                           CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/keycloak-operator      1.35.0          1.35.0          A Helm chart for KRCI Keycloak Operator
     ```

    _**NOTE:** It is highly recommended to use the latest stable version._

3. Full chart parameters available in [deploy-templates/README.md](deploy-templates/README.md).

4. Install the operator in the <edp-project> namespace with the helm command; find below the installation command example:

    ```bash
    helm install keycloak-operator epamedp/keycloak-operator --version <chart_version> --namespace <edp-project> --set name=keycloak-operator
    ```

5. Check the <edp-project> namespace containing Deployment with your operator in running status.

## Quick Start

1. Create a User in the Keycloak `Master` realm, and assign a `create-realm` role.

    Supported Keycloak versions: 26.4 to 26.8. The user holds the realm-management roles on realms it created only. On other realms it needs `manage-clients`, `manage-identity-providers` and, for `allowAdminRoleMapping`, `manage-realm`.

2. Insert newly created user credentials into Kubernetes secret:

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name:  keycloak-access
    type: Opaque
    data:
      username: dXNlcg==   # base64-encoded value of "user"
      password: cGFzcw==   # base64-encoded value of "pass"
    ```

3. Create Custom Resource `kind: Keycloak` with Keycloak instance URL and secret created on the previous step:

    ```yaml
    apiVersion: v1.edp.epam.com/v1
    kind: Keycloak
    metadata:
      name: keycloak-sample
    spec:
      secret: keycloak-access             # Secret name
      url: https://keycloak.example.com   # Keycloak URL
    ```

    Wait for the `.status` field with  `status.connected: true`

    ```bash
    # Check the current status
    kubectl get keycloak keycloak-sample -n  -o jsonpath='{.status.connected}'

    # Or wait automatically until connected
    kubectl wait --for=jsonpath='{.status.connected}'=true keycloak/keycloak-sample -n  --timeout=300s
    ```

4. Create Keycloak realm and group using Custom Resources:

   ```yaml
   apiVersion: v1.edp.epam.com/v1
   kind: KeycloakRealm
   metadata:
    name: keycloakrealm-sample
   spec:
    realmName: realm-sample
    keycloakRef:
      name: keycloak-sample
      kind: Keycloak
    ```

    ```yaml
    apiVersion: v1.edp.epam.com/v1
    kind: KeycloakRealmGroup
    metadata:
      name: argocd-admins
    spec:
      name: ArgoCDAdmins
      realmRef:
        name: keycloakrealm-sample
        kind: KeycloakRealm
    ```

    Inspect [available custom resource](./docs/arch.md) and [CR templates folder](./deploy-templates/_crd_examples/) for more examples.

#### Preventing the operator from deleting resources

To prevent the operator from deleting resources from Keycloak, add the `edp.epam.com/preserve-resources-on-deletion: "true"` annotation to the resource.

   ```yaml
   apiVersion: v1.edp.epam.com/v1
   kind: KeycloakRealm
   metadata:
    name: keycloakrealm-sample
    annotations:
      edp.epam.com/preserve-resources-on-deletion: "true"
   spec:
    realmName: realm-sample
    keycloakRef:
       name: keycloak-sample
       kind: Keycloak
   ```

#### Resources deletion

To avoid resources getting stuck during deletion, it is important to delete them in the correct order:

1. **First**, remove realm resources `KeycloakClient`, `KeycloakRealmUser`, etc.
2. **Then**, remove `KeycloakRealm`/`ClusterKeycloakRealm`.
3. **Finally**, remove `Keycloak`/`ClusterKeycloak`.

#### Known limits

- **`Keycloak`**: the operator does not watch the credential Secret. A connected `Keycloak` is checked again only on a change of the resource, an operator restart or the cache resync (about 10 hours). `status.connected` can stay `true` after a Secret change. To check at once, change an annotation of the resource. `ClusterKeycloak` is checked every 30 minutes.
- **`KeycloakClient`**: `spec.enabled` is always managed. Omitted means `true`. A client that you disable in Keycloak is enabled again on the next reconcile. To keep the client disabled, set `enabled: false`. With an empty `successReconcileTimeout` (chart default), the next reconcile comes after a change of the resource, an operator restart or the cache resync.
- **`KeycloakClient`, `KeycloakRealmIdentityProvider`**: the resource is the source of truth for the boolean flags that the operator sends. An omitted flag, such as `public`, `bearerOnly` or `trustEmail`, is written as `false`. `standardFlowEnabled` and `fullScopeAllowed` of `KeycloakClient` default to `true`. Adopting an existing client or identity provider overwrites its flags. A flag changed in the Keycloak UI is overwritten on the next reconcile. Copy the Keycloak values into the resource before adoption and after a change in the Keycloak UI.
- **`KeycloakClient`**: when webhooks are enabled (chart default), the defaulting webhook stores `webOrigins: [<webUrl>]` in a resource with `webUrl` and without `webOrigins`. A later change of `webUrl` does not change the stored `webOrigins`. After you change `webUrl`, set `webOrigins` to the new value.
- **`KeycloakClient`**: realm roles that the deprecated `spec.realmRoles` created stay in Keycloak after you delete the resource. To manage such a role, create a `KeycloakRealmRole` with the same name.
- **`KeycloakOrganization`** (Keycloak 26.8): only one organization can hold `membershipType: MANAGED` for an identity provider. In a second organization with `MANAGED`, the link stays `UNMANAGED`. Its reconcile fails before the domain settings. The status shows the Keycloak message.
- **`KeycloakOrganization`** (Keycloak older than 26.8): a non-empty `domainsV2[].identityProviderAlias`, `domainsV2[].autoRedirect: true`, `identityProviders[].autoMembership` and `identityProviders[].membershipType` fail the reconcile before any write. The status names the field and the server version. If Keycloak does not report a version, or reports one the operator cannot parse, the check is skipped. A failed `/admin/serverinfo` read fails the reconcile for such a resource on any Keycloak version.
- **`KeycloakRealmComponent`**: removing a key from `spec.config` does not remove it in Keycloak. For some providers, such as `rsa-generated`, set the key to `[]`, wait for the reconcile, then remove it from the spec.
- **`KeycloakRealmComponent`**: some providers, such as `rsa-generated`, drop `spec.config` keys that they do not declare. The operator then updates such a component on each reconcile, every 10 minutes. Remove such keys from the spec.
- **`KeycloakRealmUser`**: a user disabled in Keycloak, also by permanent lockout, is enabled again when `spec.enabled` is `true`. Keycloak then clears its login failures.
- **Metrics (Helm chart)**: the operator serves `/metrics` on port 8443 with authentication and authorization. The endpoint uses a self-signed certificate. The chart does not grant the operator `tokenreviews` and `subjectaccessreviews`, and creates no metrics Service. A scrape fails. To scrape, bind the rules of [config/rbac/metrics_auth_role.yaml](config/rbac/metrics_auth_role.yaml) to the operator service account, and bind [config/rbac/metrics_reader_role.yaml](config/rbac/metrics_reader_role.yaml) to the scraper.

## Security

Secret references, who may author which resources, and the destination allowlist: [docs/security.md](docs/security.md).

Changes that need action when you upgrade Keycloak or the operator: [docs/upgrade-notes.md](docs/upgrade-notes.md).

## Local Development

For comprehensive local development setup, testing, debugging, and common development tasks, refer to the [Development Guide](docs/development.md).

Development versions are also available from the [snapshot Helm Chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

## Using the Keycloak Client as a Library

The [`pkg/client/keycloakapi`](pkg/client/keycloakapi/) package is a standalone Go client for the Keycloak Admin REST API (Keycloak 25+, including Red Hat build of Keycloak) and can be imported independently of the operator. See the [package documentation on pkg.go.dev](https://pkg.go.dev/github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi) for usage, supported authentication options, and the available sub-clients.

### Related Articles

* [Install KubeRocketCI](https://docs.kuberocketci.io/docs/operator-guide/install-kuberocketci)
