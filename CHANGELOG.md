<a name="unreleased"></a>
## [Unreleased]


<a name="v1.20.0"></a>
## v1.20.0 - 2023-12-26
### Features
- Add missing fields to KeycloakClient ([#24](https://github.com/epam/edp-keycloak-operator/issues/24))
- Enable secret reference support in KeycloakClient resource ([#21](https://github.com/epam/edp-keycloak-operator/issues/21))
- Add annotation for preserving resources deletion ([#18](https://github.com/epam/edp-keycloak-operator/issues/18))
- Enable secret support in KeycloakRealmIdentityProvider resource ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))
- Allow multiple redirect URIs ([#12](https://github.com/epam/edp-keycloak-operator/issues/12))

### Bug Fixes
- The default realm role is no longer works ([#22](https://github.com/epam/edp-keycloak-operator/issues/22))
- KeycloakRealmIdentityProvider config secret reference is replaced by the plain secret ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))

### Testing
- Create client without specifying client secret ([#21](https://github.com/epam/edp-keycloak-operator/issues/21))
- Add e2e for KeycloakRealmIdentityProvider using secret ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))

### Routine
- Add printcolumn status for all custom resources ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Update current development version ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Generate bundle for OperatorHub v1.19.0 ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Update GH actions and release pipeline ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Upgrade pull request template ([#17](https://github.com/epam/edp-keycloak-operator/issues/17))
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#16](https://github.com/epam/edp-keycloak-operator/issues/16))
- Upgrade Go to 1.20 ([#14](https://github.com/epam/edp-keycloak-operator/issues/14))
- Update current development version ([#13](https://github.com/epam/edp-keycloak-operator/issues/13))
- Publish v1.17.1 on OperatorHub ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))
- Publish v1.17.0 on OperatorHub ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))
- Update current development version ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))

### Reverts
- [EPMDEDP-4226] Correctly update KeycloakClient CR to get correct .status.value after reconciliation


[Unreleased]: https://github.com/epam/edp-keycloak-operator/compare/v1.20.0...HEAD
