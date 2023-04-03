[![codecov](https://codecov.io/gh/epam/edp-keycloak-operator/branch/master/graph/badge.svg?token=WJ7YFRPUX2)](https://codecov.io/gh/epam/edp-keycloak-operator)

# Keycloak Operator

| :heavy_exclamation_mark: Please refer to [EDP documentation](https://epam.github.io/edp-install/) to get the main concepts and guidelines. |
| --- |

Get acquainted with the Keycloak Operator, the installation process, the local development, and the architecture scheme.

## Overview

Keycloak Operator is an EDP operator responsible for configuring existing Keycloak instances. The operator runs both on OpenShift and Kubernetes.

_**NOTE:** Operator is platform-independent, which is why there is a unified instruction for deployment._

## Prerequisites

1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;

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
     epamedp/keycloak-operator      1.15.0          1.15.0          A Helm chart for EDP Keycloak Operator
     epamedp/keycloak-operator      1.14.0          1.14.0          A Helm chart for EDP Keycloak Operator
     ```

    _**NOTE:** It is highly recommended to use the latest stable version._

3. Full chart parameters available in [deploy-templates/README.md](deploy-templates/README.md).

4. Install the operator in the <edp-project> namespace with the helm command; find below the installation command example:

    ```bash
    helm install keycloak-operator epamedp/keycloak-operator --version <chart_version> --namespace <edp-project> --set name=keycloak-operator
    ```

5. Check the <edp-project> namespace containing Deployment with your operator in running status.

## Quick Start

1. Create a User in the Keycloak `Master` realm, and assign a `create-realm` role, check [official documentation](https://github.com/keycloak/keycloak-documentation/blob/main/server_admin/topics/admin-console-permissions/master-realm.adoc#global-roles)

2. Insert newly created user credentials into Kubernetes secret:

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
      name:  keycloak-access
      namespace: default
    type: Opaque
    data:
      username: "user"
      password: "pass"
    ```

3. Create Custom Resource `kind: Keycloak` with Keycloak instance URL and secret created on the previous step:

    ```yaml
    apiVersion: v1.edp.epam.com/v1
    kind: Keycloak
    metadata:
      name: main
      namespace: default
    spec:
      secret: keycloak-access             # Secret name
      url: https://keycloak.example.com   # Keycloak URL
    ```

    Wait for the `.status` field with  `status.connected: true`

4. Create Keycloak realm and group using Custom Resources:

    ```yaml
    apiVersion: v1.edp.epam.com/v1
    kind: KeycloakRealm
    metadata:
      name: demo
      namespace: default
    spec:
      keycloakOwner: main         # the name of `kind: Keycloak`
      realmName: product-dev      # realm name in keycloak instance
      ssoRealmEnabled: false
    ```

    ```yaml
    apiVersion: v1.edp.epam.com/v1
    kind: KeycloakRealmGroup
    metadata:
      name: argocd-admins
      namespace: default
    spec:
      name: ArgoCDAdmins
      realm: demo              # the name of `kind: KeycloakRealm`
    ```

    Inspect [available custom resource](./docs/arch.md) and [CR templates folder](./deploy-templates/_crd_examples/) for more examples

## Local Development

To develop the operator, first set up a local environment, and refer to the [Local Development](https://epam.github.io/edp-install/developer-guide/local-development/) page.

Development versions are also available from the [snapshot helm chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

### Related Articles

* [Install EDP](https://epam.github.io/edp-install/operator-guide/install-edp/)
