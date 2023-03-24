# keycloak-operator

![Version: 1.15.0](https://img.shields.io/badge/Version-1.15.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.15.0](https://img.shields.io/badge/AppVersion-1.15.0-informational?style=flat-square)

A Helm chart for EDP Keycloak Operator

**Homepage:** <https://epam.github.io/edp-install/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| epmd-edp | <SupportEPMD-EDP@epam.com> | <https://solutionshub.epam.com/solution/epam-delivery-platform> |
| sergk |  | <https://github.com/SergK> |

## Source Code

* <https://github.com/epam/edp-keycloak-operator>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity for pod assignment |
| annotations | object | `{}` | Annotations to be added to the Deployment |
| extraVolumeMounts | list | `[]` | Additional volumeMounts to be added to the container |
| extraVolumes | list | `[]` | Additional volumes to be added to the pod |
| image.repository | string | `"epamedp/keycloak-operator"` | EDP keycloak-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/keycloak-operator) |
| image.tag | string | `nil` | EDP keycloak-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/keycloak-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` | If defined, a imagePullPolicy applied to the deployment |
| name | string | `"keycloak-operator"` | Application name string |
| nodeSelector | object | `{}` | Node labels for pod assignment |
| resources | object | `{"limits":{"memory":"192Mi"},"requests":{"cpu":"50m","memory":"64Mi"}}` | Resource limits and requests for the pod |
| tolerations | list | `[]` | Node tolerations for server scheduling to nodes with taints |

