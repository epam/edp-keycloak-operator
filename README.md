# How to Install Operator

EDP installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Installation of operators is platform-independent, that is why there is a unified instruction for deploying._


### Prerequisites
1. Machine with [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) installed with an authorized access to the cluster;
2. Admin space is deployed by following the repository instruction: [edp-install](https://github.com/epmd-edp/edp-install#admin-space).

### Installation
* Go to the [releases](https://github.com/epmd-edp/keycloak-operator/releases) page of this repository, choose a version, download an archive and unzip it;

_**NOTE:** It is highly recommended to use the latest released version._

* Go to the unzipped directory and apply all files with the Custom Resource Definitions resource:

`for file in $(ls crds/*_crd.yaml); do kubectl apply -f $file; done`

* Deploy operator:

`kubectl patch -n <edp_cicd_project> -f deploy/operator.yaml --local=true --patch='{"spec":{"template":{"spec":{"containers":[{"image":"epamedp/keycloak-operator:<operator_version>", "name":"keycloak-operator", "env": [{"name":"WATCH_NAMESPACE", "value":"<watch_namespace>"}, {"name":"PLATFORM_TYPE","value":"<platform>"}]}]}}}}' -o yaml | kubectl -n <edp_cicd_project> apply -f -`

- _<operator_version> - a selected release version;_

- _<edp_cicd_project> - a namespace or a project name (in case of OpenSift) where the Admin Space is deployed and that is created by following the [edp-install](https://github.com/epmd-edp/edp-install#install-edp) instructions;_

- _<platform_type> - a platform type that can be "kubernetes" or "openshift"_.

* Check the <edp_cicd_project> namespace that should contain Deployment with your operator in a running status.