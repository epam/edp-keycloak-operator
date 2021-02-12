# Keycloak Operator

Get acquainted with the Keycloak Operator and the installation process as well as the local development, 
and architecture scheme.

## Overview

Keycloak Operator is an EDP operator that is responsible for configuring existing Keycloak for integration with EDP. 
Operator installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites
1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following one of the instructions: [edp-install-openshift](https://github.com/epam/edp-install/blob/master/documentation/openshift_install_edp.md#edp-project) or [edp-install-kubernetes](https://github.com/epam/edp-install/blob/master/documentation/kubernetes_install_edp.md#edp-namespace).

## Installation
In order to install the Keycloak Operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://chartmuseum.demo.edp-epam.com/
     ```
2. Choose available Helm chart version:
     ```bash
     helm search repo epamedp/keycloak-operator
     NAME                           CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/keycloak-operator      v2.4.0                          Helm chart for Golang application/service deplo...
     ```

    _**NOTE:** It is highly recommended to use the latest released version._

3. For correct work, Keycloak Operator should have administrative access to Keycloak as it uses secret with credentials for this purpose. 
It is necessary to create such secret manually or from the existing secret using these commands as examples:  

    3.1 OpenShift:
    ```bash
    oc -n <edp_main_keycloak_project> get secret <edp_main_keycloak_secret> --export -o yaml | oc -n <edp_cicd_project> apply -f -
    ```

    3.2 Kubernetes: 
    ```bash
    kubectl -n <edp_main_keycloak_project> get secret <edp_main_keycloak_secret> --export -o yaml | kubectl -n <edp_cicd_project> apply -f -
    ```
    >_INFO: The `<edp_main_keycloak_project>` parameter is the namespace with the deployed Keycloak; the `<edp_main_keycloak_secret>` parameter is 
the name of a Keycloak secret._

   Full available chart parameters list:
   ```
     - chart_version                                 # a version of Keycloak operator Helm chart;
     - global.edpName                                # a namespace or a project name (in case of OpenShift);
     - global.platform                               # openshift or kubernetes;
     - global.admins                                 # Administrators of your tenant separated by comma (,) (eg --set 'global.admins={test@example.com}');
     - global.developers                             # Developers of your tenant separated by comma (,) (eg --set 'global.developers={test@example.com}');
     - image.name                                    # EDP image. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/keycloak-operator);
     - image.version                                 # EDP tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/keycloak-operator/tags);
     - keycloak.url                                  # URL to Keycloak;
   ```
4. Install operator in the <edp_cicd_project> namespace with the helm command; find below the installation command example:
    ```bash
    helm install keycloak-operator epamedp/keycloak-operator --version <chart_version> --namespace <edp_cicd_project> --set name=keycloak-operator --set global.edpName=<edp_cicd_project> --set global.platform=<platform_type> 
    ```
5. Check the <edp_cicd_project> namespace that should contain Deployment with your operator in a running status.

## Local Development
In order to develop the operator, first set up a local environment. For details, please refer to the [Local Development](documentation/local-development.md) page.

### Related Articles
* [Architecture Scheme of Keycloak Operator](documentation/arch.md)