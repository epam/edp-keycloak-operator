<a name="unreleased"></a>
## [Unreleased]


<a name="v1.17.0"></a>
## [v1.17.0] - 2023-08-17
### Features

- Add cluster wide CR ClusterKeycloak [EPMDEDP-121186](https://jiraeu.epam.com/browse/EPMDEDP-121186)
- Add additional printer columns for CR Keycloak and Realm [EPMDEDP-12184](https://jiraeu.epam.com/browse/EPMDEDP-12184)
- Allow configuring a realm/keycloak in a different namespace [EPMDEDP-12186](https://jiraeu.epam.com/browse/EPMDEDP-12186)
- Add KeycloakRealmComponent parentRef property [EPMDEDP-12233](https://jiraeu.epam.com/browse/EPMDEDP-12233)
- Add KeycloakClient attributes default value [EPMDEDP-12334](https://jiraeu.epam.com/browse/EPMDEDP-12334)

### Bug Fixes

- Add kind cluster name to load image command [EPMDEDP-11400](https://jiraeu.epam.com/browse/EPMDEDP-11400)
- Auth flow executions order adjustment [EPMDEDP-1204](https://jiraeu.epam.com/browse/EPMDEDP-1204)
- Fix linting [EPMDEDP-121186](https://jiraeu.epam.com/browse/EPMDEDP-121186)
- Fix cluster resource reconcilation flag [EPMDEDP-12186](https://jiraeu.epam.com/browse/EPMDEDP-12186)
- Allow non-interactive login with set password for KeycloakRealmUser [EPMDEDP-12204](https://jiraeu.epam.com/browse/EPMDEDP-12204)
- Update legacy keycloak response check [EPMDEDP-12293](https://jiraeu.epam.com/browse/EPMDEDP-12293)
- KeycloakRealm SSO use TargetRealm instead of RealmRef [EPMDEDP-12396](https://jiraeu.epam.com/browse/EPMDEDP-12396)
- Use targetRealm for KeycloakClient for backward compatibility [EPMDEDP-12396](https://jiraeu.epam.com/browse/EPMDEDP-12396)

### Testing

- Add baseline tests for clusterkeycloak controller [EPMDEDP-121186](https://jiraeu.epam.com/browse/EPMDEDP-121186)
- Add GoCloakAdapter GetRealm test [EPMDEDP-12233](https://jiraeu.epam.com/browse/EPMDEDP-12233)

### Routine

- Fix CI for GH Actions [EPMDEDP-11400](https://jiraeu.epam.com/browse/EPMDEDP-11400)
- Refactor CI for GitHub actions [EPMDEDP-11400](https://jiraeu.epam.com/browse/EPMDEDP-11400)
- Publish 1.16.0 version on OperatorHub [EPMDEDP-12148](https://jiraeu.epam.com/browse/EPMDEDP-12148)
- Update current development version [EPMDEDP-12148](https://jiraeu.epam.com/browse/EPMDEDP-12148)
- Update bundle version and helm description [EPMDEDP-12148](https://jiraeu.epam.com/browse/EPMDEDP-12148)
- Update cluster wide object naming approach [EPMDEDP-12186](https://jiraeu.epam.com/browse/EPMDEDP-12186)
- Add GH Action Codecov support [EPMDEDP-12186](https://jiraeu.epam.com/browse/EPMDEDP-12186)
- Add /bundle and /hack to sonar exclusions [EPMDEDP-12334](https://jiraeu.epam.com/browse/EPMDEDP-12334)

### BREAKING CHANGE:


Need to update all CRDs to add realmRef, keycloakRef properties, and new ClusterKeycloakRealm resource.


<a name="v1.16.0"></a>
## [v1.16.0] - 2023-06-15
### Features

- Add frontend url property for realm [EPMDEDP-11747](https://jiraeu.epam.com/browse/EPMDEDP-11747)
- Allow define KeycloakRealmUser password in Kubernetes secret [EPMDEDP-12148](https://jiraeu.epam.com/browse/EPMDEDP-12148)

### Routine

- Update current development version [EPMDEDP-11472](https://jiraeu.epam.com/browse/EPMDEDP-11472)
- Publish 1.15.0 version on OperatorHub [EPMDEDP-11825](https://jiraeu.epam.com/browse/EPMDEDP-11825)
- Update current development version [EPMDEDP-11826](https://jiraeu.epam.com/browse/EPMDEDP-11826)

### Documentation

- Add a description to the Custom Resources fields [EPMDEDP-11551](https://jiraeu.epam.com/browse/EPMDEDP-11551)


<a name="v1.15.0"></a>
## [v1.15.0] - 2023-03-24
### Features

- Added support for both legacy and modern Gocloak clients [EPMDEDP-11396](https://jiraeu.epam.com/browse/EPMDEDP-11396)
- Integration/e2e tests for operator [EPMDEDP-11398](https://jiraeu.epam.com/browse/EPMDEDP-11398)
- Add the ability to use additional volumes in helm chart [EPMDEDP-11529](https://jiraeu.epam.com/browse/EPMDEDP-11529)

### Bug Fixes

- Set proper Kubernetes version for envtest [EPMDEDP-11398](https://jiraeu.epam.com/browse/EPMDEDP-11398)
- KeycloakAuthFlow reconciliation creates new auth configs every time [EPMDEDP-11550](https://jiraeu.epam.com/browse/EPMDEDP-11550)
- Remove parallel map access in GetClientscopesByNames test [EPMDEDP-11757](https://jiraeu.epam.com/browse/EPMDEDP-11757)

### Code Refactoring

- Remove global section [EPMDEDP-11369](https://jiraeu.epam.com/browse/EPMDEDP-11369)
- Remove EDP resources out of keycloak chart [EPMDEDP-11369](https://jiraeu.epam.com/browse/EPMDEDP-11369)
- Remove EDP dependencies from chart installation [EPMDEDP-11369](https://jiraeu.epam.com/browse/EPMDEDP-11369)
- Add constant for keycloak client secret field [EPMDEDP-11656](https://jiraeu.epam.com/browse/EPMDEDP-11656)

### Routine

- Update current development version [EPMDEDP-10610](https://jiraeu.epam.com/browse/EPMDEDP-10610)
- Update version on OperatorHub [EPMDEDP-10944](https://jiraeu.epam.com/browse/EPMDEDP-10944)
- Updated dependencies [EPMDEDP-11206](https://jiraeu.epam.com/browse/EPMDEDP-11206)
- Add community cooperation templates [EPMDEDP-11401](https://jiraeu.epam.com/browse/EPMDEDP-11401)
- Update e2e tests [EPMDEDP-11483](https://jiraeu.epam.com/browse/EPMDEDP-11483)
- Add getting a Keycloak URL for tests [EPMDEDP-11483](https://jiraeu.epam.com/browse/EPMDEDP-11483)
- Update git-chglog for keycloak-operator [EPMDEDP-11518](https://jiraeu.epam.com/browse/EPMDEDP-11518)
- Bump golang.org/x/net from 0.5.0 to 0.8.0 [EPMDEDP-11578](https://jiraeu.epam.com/browse/EPMDEDP-11578)

### Documentation

- Update chart and application version in Readme file [EPMDEDP-11221](https://jiraeu.epam.com/browse/EPMDEDP-11221)


<a name="v1.14.0"></a>
## [v1.14.0] - 2022-12-05
### Features

- Keycloak client updating on CR changes [EPMDEDP-10930](https://jiraeu.epam.com/browse/EPMDEDP-10930)

### Bug Fixes

- Conversion of keycloak adapter client structure to gocloak lib structure [EPMDEDP-10930](https://jiraeu.epam.com/browse/EPMDEDP-10930)

### Routine

- Update metadata information [EPMDEDP-10639](https://jiraeu.epam.com/browse/EPMDEDP-10639)
- Update current development version [EPMDEDP-10639](https://jiraeu.epam.com/browse/EPMDEDP-10639)
- Add OpenShift specific annotation to bundle [EPMDEDP-10730](https://jiraeu.epam.com/browse/EPMDEDP-10730)
- Update installModes for operator [EPMDEDP-10730](https://jiraeu.epam.com/browse/EPMDEDP-10730)


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

[Unreleased]: https://github.com/epam/edp-keycloak-operator/compare/v1.17.0...HEAD
[v1.17.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.16.0...v1.17.0
[v1.16.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.15.0...v1.16.0
[v1.15.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.14.0...v1.15.0
[v1.14.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.13.0...v1.14.0
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
