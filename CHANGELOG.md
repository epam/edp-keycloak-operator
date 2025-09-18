<a name="unreleased"></a>
## [Unreleased]

### Features

- implement KeycloakRealmIdentityProvider CR [EPMDEDP-7911](https://jiraeu.epam.com/browse/EPMDEDP-7911)
- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- implement synchronization on access token cache [EPMDEDP-7818](https://jiraeu.epam.com/browse/EPMDEDP-7818)
- implement KeycloakRealmComponent CR [EPMDEDP-7666](https://jiraeu.epam.com/browse/EPMDEDP-7666)
- add reconciliation strategy to realm user [EPMDEDP-7694](https://jiraeu.epam.com/browse/EPMDEDP-7694)
- implement reconciliation strategy for client [EPMDEDP-7653](https://jiraeu.epam.com/browse/EPMDEDP-7653)
- invalidate keycloak client token after creation of realm [EPMDEDP-7655](https://jiraeu.epam.com/browse/EPMDEDP-7655)
- Add reconciliation phase after successful one [EPMDEDP-7358](https://jiraeu.epam.com/browse/EPMDEDP-7358)
- add ability to login to master realm with service account [EPMDEDP-7445](https://jiraeu.epam.com/browse/EPMDEDP-7445)
- add frontChannelLogout param to kc client CR [EPMDEDP-7526](https://jiraeu.epam.com/browse/EPMDEDP-7526)
- add ability to create kc default client scopes [EPMDEDP-7531](https://jiraeu.epam.com/browse/EPMDEDP-7531)

### Bug Fixes

- Expand edp-keycloak-operator role [EPMDEDP-7736](https://jiraeu.epam.com/browse/EPMDEDP-7736)

### Code Refactoring

- Address golangci-lint issues [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)
- fix client scope errors [EPMDEDP-7734](https://jiraeu.epam.com/browse/EPMDEDP-7734)
- Expand keycloak-operator role [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Add namespace field in roleRef in OKD RB, align CRB name [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Replace cluster-wide role/rolebinding to namespaced [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)

### Formatting

- Remove unnecessary space [EPMDEDP-7943](https://jiraeu.epam.com/browse/EPMDEDP-7943)

### Testing

- tests for controller helper and adapter [EPMDEDP-7818](https://jiraeu.epam.com/browse/EPMDEDP-7818)

### Routine

- update Go version at codecov.yaml [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update gocloak to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Add codecov report [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)

### Documentation

- Update the links on GitHub [EPMDEDP-7781](https://jiraeu.epam.com/browse/EPMDEDP-7781)


<a name="v1.9.0"></a>
## [v1.9.0] - 2021-12-03

<a name="v1.8.0"></a>
## [v1.8.0] - 2021-12-03

<a name="v1.7.3"></a>
## [v1.7.3] - 2021-12-03

<a name="v1.7.2"></a>
## [v1.7.2] - 2021-12-03

<a name="v1.7.1"></a>
## [v1.7.1] - 2021-12-03

<a name="v1.7.0"></a>
## [v1.7.0] - 2021-12-03

[Unreleased]: https://github.com/epam/edp-keycloak-operator/compare/v1.9.0...HEAD
[v1.9.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.8.0...v1.9.0
[v1.8.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.3...v1.8.0
[v1.7.3]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.2...v1.7.3
[v1.7.2]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.1...v1.7.2
[v1.7.1]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.0...v1.7.1
[v1.7.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.3.0-alpha-81...v1.7.0
