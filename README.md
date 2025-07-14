[![codecov](https://codecov.io/gh/epam/edp-keycloak-operator/branch/master/graph/badge.svg?token=WJ7YFRPUX2)](https://codecov.io/gh/epam/edp-keycloak-operator)

# Keycloak Operator

| :heavy_exclamation_mark: Please refer to [KubeRocketCI documentation](https://docs.kuberocketci.io/) to get the main concepts and guidelines. |
| --- |

Get acquainted with the Keycloak Operator, the installation process, the quick start, and the local development guidelines.

## Overview

Keycloak Operator is a KubeRocketCI operator responsible for configuring existing Keycloak instances. The operator runs both on OpenShift and Kubernetes.

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
     epamedp/keycloak-operator      1.28.0          1.28.0          A Helm chart for KRCI Keycloak Operator
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

## Local Development

To develop the operator, first set up a local environment, and refer to the [Local Development](https://docs.kuberocketci.io/docs/developer-guide/local-development) page.

Development versions are also available from the [snapshot Helm Chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

### Related Articles

* [Install KubeRocketCI](https://docs.kuberocketci.io/docs/operator-guide/install-kuberocketci)
