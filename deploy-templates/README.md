# keycloak-operator

![Version: 1.12.0-SNAPSHOT](https://img.shields.io/badge/Version-1.12.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.12.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-1.12.0--SNAPSHOT-informational?style=flat-square)

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
| affinity | object | `{}` |  |
| annotations | object | `{}` |  |
| global.admins[0] | string | `"stub_user_one@example.com"` |  |
| global.developers[0] | string | `"stub_user_one@example.com"` |  |
| global.edpName | string | `""` |  |
| global.platform | string | `"openshift"` |  |
| image.repository | string | `"epamedp/keycloak-operator"` |  |
| image.tag | string | `nil` |  |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| keycloak.url | string | `"https://keycloak.example.com"` |  |
| name | string | `"keycloak-operator"` |  |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

