<a name="unreleased"></a>
## [Unreleased]


<a name="v1.21.0"></a>
## v1.21.0 - 2024-05-15
### Features
- Add imagePullSecrets to enable private repository
- Add support for composite client role ([#44](https://github.com/epam/edp-keycloak-operator/issues/44))
- Remove SSORealm functionality from KeycloakRealm ([#47](https://github.com/epam/edp-keycloak-operator/issues/47))
- Full reconciliation of KeycloakRealmUser  ([#45](https://github.com/epam/edp-keycloak-operator/issues/45))
- Add Scopes to KeycloakClient Authorization spec ([#41](https://github.com/epam/edp-keycloak-operator/issues/41))
- Add ability to configure Realm token Settings ([#38](https://github.com/epam/edp-keycloak-operator/issues/38))
- Add custom certificate support ([#36](https://github.com/epam/edp-keycloak-operator/issues/36))
- Allow creating Authorization Permissions for a Client ([#28](https://github.com/epam/edp-keycloak-operator/issues/28))
- Allow creating Authorization Policies for a Client ([#28](https://github.com/epam/edp-keycloak-operator/issues/28))
- Enable review for pull requests ([#32](https://github.com/epam/edp-keycloak-operator/issues/32))
- Allow secret references in KeycloakRealmComponent ([#30](https://github.com/epam/edp-keycloak-operator/issues/30))
- Add missing fields to KeycloakClient ([#24](https://github.com/epam/edp-keycloak-operator/issues/24))
- Enable secret reference support in KeycloakClient resource ([#21](https://github.com/epam/edp-keycloak-operator/issues/21))
- Add annotation for preserving resources deletion ([#18](https://github.com/epam/edp-keycloak-operator/issues/18))
- Enable secret support in KeycloakRealmIdentityProvider resource ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))
- Allow multiple redirect URIs ([#12](https://github.com/epam/edp-keycloak-operator/issues/12))

### Bug Fixes
- Remove from code coverage mock files ([#28](https://github.com/epam/edp-keycloak-operator/issues/28))
- The default realm role is no longer works ([#22](https://github.com/epam/edp-keycloak-operator/issues/22))
- KeycloakRealmIdentityProvider config secret reference is replaced by the plain secret ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))

### Testing
- Add integration tests for KeycloakClientScope ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Add integration tests for KeycloakRealmUser ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Add integration tests for KeycloakAuthFlow ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Add integration tests for KeycloakRealm ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Create client without specifying client secret ([#21](https://github.com/epam/edp-keycloak-operator/issues/21))
- Add e2e for KeycloakRealmIdentityProvider using secret ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))

### Routine
- Bump to Go 1.22 ([#57](https://github.com/epam/edp-keycloak-operator/issues/57))
- Add CODEOWNERS ([#49](https://github.com/epam/edp-keycloak-operator/issues/49))
- Migrate from gerrit to github pipelines ([#49](https://github.com/epam/edp-keycloak-operator/issues/49))
- Bump google.golang.org/protobuf from 1.28.1 to 1.33.0 ([#39](https://github.com/epam/edp-keycloak-operator/issues/39))
- Update operator bundle ([#37](https://github.com/epam/edp-keycloak-operator/issues/37))
- Add ClusterRoleBinding for operatorHub([#37](https://github.com/epam/edp-keycloak-operator/issues/37))
- Remove explicit caching in workflows ([#34](https://github.com/epam/edp-keycloak-operator/issues/34))
- Implement cache in github workflow ([#34](https://github.com/epam/edp-keycloak-operator/issues/34))
- Generate OperatorHub bundle for the version 1.20.0 ([#27](https://github.com/epam/edp-keycloak-operator/issues/27))
- Update current development version ([#27](https://github.com/epam/edp-keycloak-operator/issues/27))
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

### Documentation
- Update README md file ([#132](https://github.com/epam/edp-keycloak-operator/issues/132))

### Reverts
- [EPMDEDP-4226] Correctly update KeycloakClient CR to get correct .status.value after reconciliation


[Unreleased]: https://github.com/epam/edp-keycloak-operator/compare/v1.21.0...HEAD
