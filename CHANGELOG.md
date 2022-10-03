<a name="unreleased"></a>
## [Unreleased]

### Routine

- Update current development version [EPMDEDP-10639](https://jiraeu.epam.com/browse/EPMDEDP-10639)


<a name="v1.13.0"></a>
## [v1.13.0] - 2022-10-03
### Features

- Upgrade operator-sdk [EPMDEDP-10417](https://jiraeu.epam.com/browse/EPMDEDP-10417)
- Align deploy-templates to new operator-sdk config [EPMDEDP-10540](https://jiraeu.epam.com/browse/EPMDEDP-10540)
- Generate OperatorHub bundle [EPMDEDP-10617](https://jiraeu.epam.com/browse/EPMDEDP-10617)

### Bug Fixes

- Add secret permissions, operator image path [EPMDEDP-10540](https://jiraeu.epam.com/browse/EPMDEDP-10540)
- Revert removed service account for operator [EPMDEDP-10540](https://jiraeu.epam.com/browse/EPMDEDP-10540)
- Align realm name to the existing approach [EPMDEDP-10648](https://jiraeu.epam.com/browse/EPMDEDP-10648)
- Metric ports default value [EPMDEDP-10648](https://jiraeu.epam.com/browse/EPMDEDP-10648)

### Code Refactoring

- Apply wrapcheck lint [EPMDEDP-10449](https://jiraeu.epam.com/browse/EPMDEDP-10449)
- Apply wsl lint [EPMDEDP-10449](https://jiraeu.epam.com/browse/EPMDEDP-10449)
- Apply new lint config [EPMDEDP-10449](https://jiraeu.epam.com/browse/EPMDEDP-10449)
- Remove edp dependencies from controllers [EPMDEDP-10648](https://jiraeu.epam.com/browse/EPMDEDP-10648)

### Routine

- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update bundle content [EPMDEDP-10617](https://jiraeu.epam.com/browse/EPMDEDP-10617)

### BREAKING CHANGE:


KeycloakRealm with name `main` is now managed by helm chart and NOT by keycloak cotroller


<a name="v1.12.0"></a>
## [v1.12.0] - 2022-08-26
### Features

- Switch to use V1 apis of EDP components [EPMDEDP-10080](https://jiraeu.epam.com/browse/EPMDEDP-10080)
- Download helm-docs locally if required by make [EPMDEDP-10105](https://jiraeu.epam.com/browse/EPMDEDP-10105)
- Download required tools for Makefile targets [EPMDEDP-10105](https://jiraeu.epam.com/browse/EPMDEDP-10105)
- Pre-create edp clientscope as a part of kecloakclientscope CR [EPMDEDP-8323](https://jiraeu.epam.com/browse/EPMDEDP-8323)
- Default scopes can be assigned for keycloakclient CR [EPMDEDP-8323](https://jiraeu.epam.com/browse/EPMDEDP-8323)
- Switch CRDs to v1 version [EPMDEDP-9219](https://jiraeu.epam.com/browse/EPMDEDP-9219)

### Bug Fixes

- Re-reconcile Keycloak client if the client scope is not found [EPMDEDP-10098](https://jiraeu.epam.com/browse/EPMDEDP-10098)
- Use installed helm-docs instead of global one [EPMDEDP-10105](https://jiraeu.epam.com/browse/EPMDEDP-10105)
- Realm password policy [EPMDEDP-9223](https://jiraeu.epam.com/browse/EPMDEDP-9223)
- Fix artifacthub.io CRD examples [EPMDEDP-9515](https://jiraeu.epam.com/browse/EPMDEDP-9515)
- Removed duplicate CRD example from Cart.yaml [EPMDEDP-9515](https://jiraeu.epam.com/browse/EPMDEDP-9515)

### Code Refactoring

- Use repository and tag for image reference in chart [EPMDEDP-10389](https://jiraeu.epam.com/browse/EPMDEDP-10389)

### Routine

- Upgrade go version to 1.18 [EPMDEDP-10110](https://jiraeu.epam.com/browse/EPMDEDP-10110)
- Fix Jira Ticket pattern for changelog generator [EPMDEDP-10159](https://jiraeu.epam.com/browse/EPMDEDP-10159)
- Update alpine base image to 3.16.2 version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update alpine base image version [EPMDEDP-10280](https://jiraeu.epam.com/browse/EPMDEDP-10280)
- Change 'go get' to 'go install' for git-chglog [EPMDEDP-10337](https://jiraeu.epam.com/browse/EPMDEDP-10337)
- Remove VERSION file [EPMDEDP-10387](https://jiraeu.epam.com/browse/EPMDEDP-10387)
- Add gcflags for go build artifact [EPMDEDP-10411](https://jiraeu.epam.com/browse/EPMDEDP-10411)
- Update current development version [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update chart annotation [EPMDEDP-9515](https://jiraeu.epam.com/browse/EPMDEDP-9515)

### Documentation

- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)


<a name="v1.11.0"></a>
## [v1.11.0] - 2022-05-25
### Features

- Update Makefile changelog target [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- implement hierarchical auth flow [EPMDEDP-8326](https://jiraeu.epam.com/browse/EPMDEDP-8326)
- add priority and requirement params to child auth flow [EPMDEDP-8326](https://jiraeu.epam.com/browse/EPMDEDP-8326)
- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)
- Password policy for realm [EPMDEDP-8395](https://jiraeu.epam.com/browse/EPMDEDP-8395)
- Add ability to disable central idp mappers creation [EPMDEDP-8397](https://jiraeu.epam.com/browse/EPMDEDP-8397)
- Full reconciliation for keycloak realm user [EPMDEDP-8786](https://jiraeu.epam.com/browse/EPMDEDP-8786)

### Bug Fixes

- Fix changelog generation in GH Release Action [EPMDEDP-8468](https://jiraeu.epam.com/browse/EPMDEDP-8468)
- User group sync [EPMDEDP-8786](https://jiraeu.epam.com/browse/EPMDEDP-8786)
- Keycloak auth flow deletion [EPMDEDP-8903](https://jiraeu.epam.com/browse/EPMDEDP-8903)
- User roles sync [EPMDEDP-9006](https://jiraeu.epam.com/browse/EPMDEDP-9006)
- Realm password policy [EPMDEDP-9223](https://jiraeu.epam.com/browse/EPMDEDP-9223)

### Routine

- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Update changelog [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Add artifacthub tags [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update keycloak URL link [EPMDEDP-8204](https://jiraeu.epam.com/browse/EPMDEDP-8204)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Add examples for ArgoCD config in Keycloak [EPMDEDP-8312](https://jiraeu.epam.com/browse/EPMDEDP-8312)
- Update base docker image to alpine 3.15.4 [EPMDEDP-8853](https://jiraeu.epam.com/browse/EPMDEDP-8853)
- Update changelog [EPMDEDP-9185](https://jiraeu.epam.com/browse/EPMDEDP-9185)


<a name="v1.10.1"></a>
## [v1.10.1] - 2022-02-09
### Features

- Update Makefile changelog target [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)

### Routine

- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Add artifacthub tags [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)


<a name="v1.10.0"></a>
## [v1.10.0] - 2021-12-06
### Features

- Add reconciliation phase after successful one [EPMDEDP-7358](https://jiraeu.epam.com/browse/EPMDEDP-7358)
- add ability to login to master realm with service account [EPMDEDP-7445](https://jiraeu.epam.com/browse/EPMDEDP-7445)
- add frontChannelLogout param to kc client CR [EPMDEDP-7526](https://jiraeu.epam.com/browse/EPMDEDP-7526)
- add ability to create kc default client scopes [EPMDEDP-7531](https://jiraeu.epam.com/browse/EPMDEDP-7531)
- implement reconciliation strategy for client [EPMDEDP-7653](https://jiraeu.epam.com/browse/EPMDEDP-7653)
- invalidate keycloak client token after creation of realm [EPMDEDP-7655](https://jiraeu.epam.com/browse/EPMDEDP-7655)
- implement KeycloakRealmComponent CR [EPMDEDP-7666](https://jiraeu.epam.com/browse/EPMDEDP-7666)
- add reconciliation strategy to realm user [EPMDEDP-7694](https://jiraeu.epam.com/browse/EPMDEDP-7694)
- implement synchronization on access token cache [EPMDEDP-7818](https://jiraeu.epam.com/browse/EPMDEDP-7818)
- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- implement KeycloakRealmIdentityProvider CR [EPMDEDP-7911](https://jiraeu.epam.com/browse/EPMDEDP-7911)

### Bug Fixes

- Expand edp-keycloak-operator role [EPMDEDP-7736](https://jiraeu.epam.com/browse/EPMDEDP-7736)
- Changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Code Refactoring

- Expand keycloak-operator role [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Add namespace field in roleRef in OKD RB, align CRB name [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Replace cluster-wide role/rolebinding to namespaced [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- fix client scope errors [EPMDEDP-7734](https://jiraeu.epam.com/browse/EPMDEDP-7734)
- Address golangci-lint issues [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)

### Formatting

- Remove unnecessary space [EPMDEDP-7943](https://jiraeu.epam.com/browse/EPMDEDP-7943)

### Testing

- tests for controller helper and adapter [EPMDEDP-7818](https://jiraeu.epam.com/browse/EPMDEDP-7818)

### Routine

- Add changelog generator [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- update Go version at codecov.yaml [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Add codecov report [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)
- Update gocloak to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)

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

[Unreleased]: https://github.com/epam/edp-keycloak-operator/compare/v1.13.0...HEAD
[v1.13.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.12.0...v1.13.0
[v1.12.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.11.0...v1.12.0
[v1.11.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.10.1...v1.11.0
[v1.10.1]: https://github.com/epam/edp-keycloak-operator/compare/v1.10.0...v1.10.1
[v1.10.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.9.0...v1.10.0
[v1.9.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.8.0...v1.9.0
[v1.8.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.3...v1.8.0
[v1.7.3]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.2...v1.7.3
[v1.7.2]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.1...v1.7.2
[v1.7.1]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.0...v1.7.1
[v1.7.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.3.0-alpha-81...v1.7.0
