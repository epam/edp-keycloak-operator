<a name="unreleased"></a>
## [Unreleased]


<a name="v1.10.0"></a>
## [v1.10.0] - 2021-12-06
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

- Changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
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

- Add changelog generator [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
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

<a name="v1.3.0-alpha-81"></a>
## [v1.3.0-alpha-81] - 2020-01-20

<a name="v1.3.0-alpha-79"></a>
## [v1.3.0-alpha-79] - 2019-12-05

<a name="v1.2.1-80"></a>
## [v1.2.1-80] - 2020-01-20

<a name="v1.2.0-78"></a>
## [v1.2.0-78] - 2019-12-05

<a name="v1.1.5-alpha-77"></a>
## [v1.1.5-alpha-77] - 2019-12-05

<a name="v1.1.5-alpha-76"></a>
## [v1.1.5-alpha-76] - 2019-12-04

<a name="v1.1.4-alpha-75"></a>
## [v1.1.4-alpha-75] - 2019-12-04

<a name="v1.1.4-alpha-74"></a>
## [v1.1.4-alpha-74] - 2019-12-03

<a name="v1.1.3-alpha-73"></a>
## [v1.1.3-alpha-73] - 2019-12-03

<a name="v1.1.3-alpha-72"></a>
## [v1.1.3-alpha-72] - 2019-12-03
### Reverts

- [EPMDEDP-4226] Correctly update KeycloakClient CR to get correct .status.value after reconciliation


<a name="v1.1.3-alpha-71"></a>
## [v1.1.3-alpha-71] - 2019-12-03

<a name="v1.1.3-alpha-70"></a>
## [v1.1.3-alpha-70] - 2019-11-27

<a name="v1.1.3-alpha-69"></a>
## [v1.1.3-alpha-69] - 2019-11-25

<a name="v1.1.3-alpha-68"></a>
## [v1.1.3-alpha-68] - 2019-11-25

<a name="v1.1.3-alpha-67"></a>
## [v1.1.3-alpha-67] - 2019-11-25

<a name="v1.1.2-alpha-66"></a>
## [v1.1.2-alpha-66] - 2019-11-25

<a name="v1.1.2-alpha-65"></a>
## [v1.1.2-alpha-65] - 2019-11-11

<a name="v1.1.2-alpha-64"></a>
## [v1.1.2-alpha-64] - 2019-11-06

<a name="v1.1.2-alpha-63"></a>
## [v1.1.2-alpha-63] - 2019-11-06

<a name="v1.1.2-alpha-61"></a>
## [v1.1.2-alpha-61] - 2019-10-30

<a name="v1.1.1-alpha-60"></a>
## [v1.1.1-alpha-60] - 2019-10-17

<a name="v1.1.0-alpha-59"></a>
## [v1.1.0-alpha-59] - 2019-09-30

<a name="v1.0.32-alpha-58"></a>
## [v1.0.32-alpha-58] - 2019-09-30

<a name="v1.0.32-alpha-57"></a>
## [v1.0.32-alpha-57] - 2019-09-30

<a name="v1.0.31-alpha-56"></a>
## [v1.0.31-alpha-56] - 2019-09-25

<a name="v1.0.30-alpha-55"></a>
## [v1.0.30-alpha-55] - 2019-09-20

<a name="v1.0.29-alpha-54"></a>
## [v1.0.29-alpha-54] - 2019-09-20

<a name="v1.0.28-alpha-53"></a>
## [v1.0.28-alpha-53] - 2019-09-19

<a name="v1.0.27-alpha-52"></a>
## [v1.0.27-alpha-52] - 2019-09-19

<a name="v1.0.26-alpha-51"></a>
## [v1.0.26-alpha-51] - 2019-09-19

<a name="v1.0.26-alpha-50"></a>
## [v1.0.26-alpha-50] - 2019-09-18

<a name="v1.0.25-alpha-49"></a>
## [v1.0.25-alpha-49] - 2019-09-18

<a name="v1.0.24-alpha-48"></a>
## [v1.0.24-alpha-48] - 2019-09-17

<a name="v1.0.24-alpha-47"></a>
## [v1.0.24-alpha-47] - 2019-09-16

<a name="v1.0.23-alpha-46"></a>
## [v1.0.23-alpha-46] - 2019-09-16

<a name="v1.0.22-alpha-45"></a>
## [v1.0.22-alpha-45] - 2019-09-16

<a name="v1.0.21-alpha-44"></a>
## [v1.0.21-alpha-44] - 2019-09-16

<a name="v1.0.20-alpha-43"></a>
## [v1.0.20-alpha-43] - 2019-09-16

<a name="v1.0.19-alpha-40"></a>
## [v1.0.19-alpha-40] - 2019-09-13

<a name="v1.0.18-alpha-39"></a>
## [v1.0.18-alpha-39] - 2019-09-13

<a name="v1.0.17-alpha-38"></a>
## [v1.0.17-alpha-38] - 2019-09-12

<a name="v1.0.16-alpha-37"></a>
## [v1.0.16-alpha-37] - 2019-09-12

<a name="v1.0.16-alpha-36"></a>
## [v1.0.16-alpha-36] - 2019-09-11

<a name="v1.0.15-alpha-34"></a>
## [v1.0.15-alpha-34] - 2019-09-11

<a name="v1.0.14-alpha-33"></a>
## [v1.0.14-alpha-33] - 2019-09-10

<a name="v1.0.14-alpha-32"></a>
## v1.0.14-alpha-32 - 2019-09-10

[Unreleased]: https://github.com/epam/edp-keycloak-operator/compare/v1.10.0...HEAD
[v1.10.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.9.0...v1.10.0
[v1.9.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.8.0...v1.9.0
[v1.8.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.3...v1.8.0
[v1.7.3]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.2...v1.7.3
[v1.7.2]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.1...v1.7.2
[v1.7.1]: https://github.com/epam/edp-keycloak-operator/compare/v1.7.0...v1.7.1
[v1.7.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.3.0-alpha-81...v1.7.0
[v1.3.0-alpha-81]: https://github.com/epam/edp-keycloak-operator/compare/v1.3.0-alpha-79...v1.3.0-alpha-81
[v1.3.0-alpha-79]: https://github.com/epam/edp-keycloak-operator/compare/v1.2.1-80...v1.3.0-alpha-79
[v1.2.1-80]: https://github.com/epam/edp-keycloak-operator/compare/v1.2.0-78...v1.2.1-80
[v1.2.0-78]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.5-alpha-77...v1.2.0-78
[v1.1.5-alpha-77]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.5-alpha-76...v1.1.5-alpha-77
[v1.1.5-alpha-76]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.4-alpha-75...v1.1.5-alpha-76
[v1.1.4-alpha-75]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.4-alpha-74...v1.1.4-alpha-75
[v1.1.4-alpha-74]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.3-alpha-73...v1.1.4-alpha-74
[v1.1.3-alpha-73]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.3-alpha-72...v1.1.3-alpha-73
[v1.1.3-alpha-72]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.3-alpha-71...v1.1.3-alpha-72
[v1.1.3-alpha-71]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.3-alpha-70...v1.1.3-alpha-71
[v1.1.3-alpha-70]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.3-alpha-69...v1.1.3-alpha-70
[v1.1.3-alpha-69]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.3-alpha-68...v1.1.3-alpha-69
[v1.1.3-alpha-68]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.3-alpha-67...v1.1.3-alpha-68
[v1.1.3-alpha-67]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.2-alpha-66...v1.1.3-alpha-67
[v1.1.2-alpha-66]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.2-alpha-65...v1.1.2-alpha-66
[v1.1.2-alpha-65]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.2-alpha-64...v1.1.2-alpha-65
[v1.1.2-alpha-64]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.2-alpha-63...v1.1.2-alpha-64
[v1.1.2-alpha-63]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.2-alpha-61...v1.1.2-alpha-63
[v1.1.2-alpha-61]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.1-alpha-60...v1.1.2-alpha-61
[v1.1.1-alpha-60]: https://github.com/epam/edp-keycloak-operator/compare/v1.1.0-alpha-59...v1.1.1-alpha-60
[v1.1.0-alpha-59]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.32-alpha-58...v1.1.0-alpha-59
[v1.0.32-alpha-58]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.32-alpha-57...v1.0.32-alpha-58
[v1.0.32-alpha-57]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.31-alpha-56...v1.0.32-alpha-57
[v1.0.31-alpha-56]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.30-alpha-55...v1.0.31-alpha-56
[v1.0.30-alpha-55]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.29-alpha-54...v1.0.30-alpha-55
[v1.0.29-alpha-54]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.28-alpha-53...v1.0.29-alpha-54
[v1.0.28-alpha-53]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.27-alpha-52...v1.0.28-alpha-53
[v1.0.27-alpha-52]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.26-alpha-51...v1.0.27-alpha-52
[v1.0.26-alpha-51]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.26-alpha-50...v1.0.26-alpha-51
[v1.0.26-alpha-50]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.25-alpha-49...v1.0.26-alpha-50
[v1.0.25-alpha-49]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.24-alpha-48...v1.0.25-alpha-49
[v1.0.24-alpha-48]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.24-alpha-47...v1.0.24-alpha-48
[v1.0.24-alpha-47]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.23-alpha-46...v1.0.24-alpha-47
[v1.0.23-alpha-46]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.22-alpha-45...v1.0.23-alpha-46
[v1.0.22-alpha-45]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.21-alpha-44...v1.0.22-alpha-45
[v1.0.21-alpha-44]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.20-alpha-43...v1.0.21-alpha-44
[v1.0.20-alpha-43]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.19-alpha-40...v1.0.20-alpha-43
[v1.0.19-alpha-40]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.18-alpha-39...v1.0.19-alpha-40
[v1.0.18-alpha-39]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.17-alpha-38...v1.0.18-alpha-39
[v1.0.17-alpha-38]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.16-alpha-37...v1.0.17-alpha-38
[v1.0.16-alpha-37]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.16-alpha-36...v1.0.16-alpha-37
[v1.0.16-alpha-36]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.15-alpha-34...v1.0.16-alpha-36
[v1.0.15-alpha-34]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.14-alpha-33...v1.0.15-alpha-34
[v1.0.14-alpha-33]: https://github.com/epam/edp-keycloak-operator/compare/v1.0.14-alpha-32...v1.0.14-alpha-33
