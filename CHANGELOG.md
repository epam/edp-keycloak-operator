<a name="unreleased"></a>
## [Unreleased]

### Routine
- Publish Operator to stable and alpha channels ([#263](https://github.com/epam/edp-keycloak-operator/issues/263))
- Update current development version ([#263](https://github.com/epam/edp-keycloak-operator/issues/263))


<a name="v1.31.1"></a>
## [v1.31.1] - 2026-01-13
### Routine
- Publish Operator to stable and alpha channels ([#263](https://github.com/epam/edp-keycloak-operator/issues/263))


<a name="v1.31.0"></a>
## [v1.31.0] - 2025-12-22
### Features
- Add webhook defaulting for KeycloakClient resources ([#205](https://github.com/epam/edp-keycloak-operator/issues/205))
- Add Type field to KeycloakClientScope for default/optional/none scopes ([#220](https://github.com/epam/edp-keycloak-operator/issues/220))
- Add webhook validation for KeycloakRealm resources ([#259](https://github.com/epam/edp-keycloak-operator/issues/259))
- Add Conditions to KeycloakClient status ([#206](https://github.com/epam/edp-keycloak-operator/issues/206))
- Add ability to set temporary user password ([#252](https://github.com/epam/edp-keycloak-operator/issues/252))

### Routine
- Update current development version ([#253](https://github.com/epam/edp-keycloak-operator/issues/253))

### BREAKING CHANGE:

Webhook functionality now requires cert-manager
to be installed in the cluster by default.
Users can disable webhooks by setting \`enableWebhooks: false\` in Helm values.


<a name="v1.30.0"></a>
## [v1.30.0] - 2025-11-21
### Features
- Add postBrokerLoginFlowAlias to KeycloakRealmIdentityProvider ([#231](https://github.com/epam/edp-keycloak-operator/issues/231))
- Add login settings for KeycloakRealm and ClusterKeycloakRealm ([#166](https://github.com/epam/edp-keycloak-operator/issues/166))
- Enhance deletion logic to skip non-existent resources ([#238](https://github.com/epam/edp-keycloak-operator/issues/238))
- Add parentGroup to KeycloakRealmGroup for improved subgroup support ([#172](https://github.com/epam/edp-keycloak-operator/issues/172))
- Add Docker registry and podLabels to helm chart

### Bug Fixes
- Unable to get realm userProfileConfig ([#221](https://github.com/epam/edp-keycloak-operator/issues/221))
- Unable to set realm userProfileConfig ([#221](https://github.com/epam/edp-keycloak-operator/issues/221))
- Authentication flow priorities are not respected ([#235](https://github.com/epam/edp-keycloak-operator/issues/235))
- Reconciliation KeycloakAuthFlow fails - authFlowConfig not found ([#234](https://github.com/epam/edp-keycloak-operator/issues/234))
- Unable to create userProfileConfig without a group ([#165](https://github.com/epam/edp-keycloak-operator/issues/165))

### Routine
- Update current development version ([#253](https://github.com/epam/edp-keycloak-operator/issues/253))
- Update krci-ai framework to the latest version ([#198](https://github.com/epam/edp-keycloak-operator/issues/198))
- Generate containerImage annotation for ClusterServiceVersion ([#225](https://github.com/epam/edp-keycloak-operator/issues/225))
- Update bundle manifests v1.29.0 ([#225](https://github.com/epam/edp-keycloak-operator/issues/225))
- Generate bundle manifests v1.29.0 ([#225](https://github.com/epam/edp-keycloak-operator/issues/225))

### Documentation
- add kubectl wait example for Keycloak status check ([#243](https://github.com/epam/edp-keycloak-operator/issues/243))


<a name="v1.29.0"></a>
## [v1.29.0] - 2025-09-22
### Features
- Add support for list of attributes with the same key
- Add ability to add Permissions to KeycloakRealmIdentityProvider
- Add KeycloakClient ClientRolesV2 field ([#152](https://github.com/epam/edp-keycloak-operator/issues/152))
- Introduce KeycloakOrganization feature ([#115](https://github.com/epam/edp-keycloak-operator/issues/115))

### Bug Fixes
- Don't delete Authorization settings on KeycloakClient when using addOnly strategy
- KeycloakRealmIdentityProvider policies not being added to permissions
- Unable to update default browserFlow in master realm ([#143](https://github.com/epam/edp-keycloak-operator/issues/143))
- Client creation fails due to admin fine grained permissions ([#180](https://github.com/epam/edp-keycloak-operator/issues/180))

### Testing
- Use Keycloak version 26.3 for tests ([#192](https://github.com/epam/edp-keycloak-operator/issues/192))

### Routine
- Add development documentation ([#204](https://github.com/epam/edp-keycloak-operator/issues/204))
- Update KubeRocketAI ([#198](https://github.com/epam/edp-keycloak-operator/issues/198))
- Setup KubeRocketAI ([#198](https://github.com/epam/edp-keycloak-operator/issues/198))
- Refactor dependencies to use standard library alternatives ([#203](https://github.com/epam/edp-keycloak-operator/issues/203))
- Updated mockery from v2.53.2 to v3.5.4 ([#202](https://github.com/epam/edp-keycloak-operator/issues/202))
- Upgrade golangci-lint to v2 ([#199](https://github.com/epam/edp-keycloak-operator/issues/199))
- Update Operator SDK from v1.39.2 to v1.41.1 ([#199](https://github.com/epam/edp-keycloak-operator/issues/199))
- Publish 1.28.0 on the OperatorHub ([#185](https://github.com/epam/edp-keycloak-operator/issues/185))
- Update current development version ([#185](https://github.com/epam/edp-keycloak-operator/issues/185))


<a name="v1.28.0"></a>
## [v1.28.0] - 2025-07-14
### Features
- Add validation for user identity provider ([#183](https://github.com/epam/edp-keycloak-operator/issues/183))
- Add the ability to add client roles to KeycloakRealmUser ([#135](https://github.com/epam/edp-keycloak-operator/issues/135))
- Allow finalizer cleanup when realm is already deleted ([#173](https://github.com/epam/edp-keycloak-operator/issues/173))

### Bug Fixes
- Handle not found error when deleting KeycloakRealmUser ([#181](https://github.com/epam/edp-keycloak-operator/issues/181))
- keycloakrealmidentityprovider try to delete first in reconciliation loop_
- KeycloakClient service account users groups aren't being populated correctly

### Routine
- Update codeql and codecov scan gh actions ([#178](https://github.com/epam/edp-keycloak-operator/issues/178))
- Publish on the OperatorHub ([#170](https://github.com/epam/edp-keycloak-operator/issues/170))
- Add multi-architecture build support ([#168](https://github.com/epam/edp-keycloak-operator/issues/168))
- Update current development version ([#160](https://github.com/epam/edp-keycloak-operator/issues/160))
- Update current development version ([#160](https://github.com/epam/edp-keycloak-operator/issues/160))


<a name="v1.27.1"></a>
## [v1.27.1] - 2025-05-08
### Bug Fixes
- KeycloakClient service account users groups aren't being populated correctly


<a name="v1.27.0"></a>
## [v1.27.0] - 2025-05-07
### Features
- Add the ability to add KeycloakClient service accounts to groups
- Make ownerReference in Keycloak resources optional ([#71](https://github.com/epam/edp-keycloak-operator/issues/71))
- Add support for Identity Providers in KeycloakRealmUser ([#148](https://github.com/epam/edp-keycloak-operator/issues/148))

### Routine
- Remove deprecated properties from CRs ([#154](https://github.com/epam/edp-keycloak-operator/issues/154))
- Bump GitHub Actions runner image to 22.04([#150](https://github.com/epam/edp-keycloak-operator/issues/150))
- Update current development version ([#146](https://github.com/epam/edp-keycloak-operator/issues/146))


<a name="v1.26.0"></a>
## [v1.26.0] - 2025-04-11
### Routine
- Make securityContext configurable via values.yaml ([#141](https://github.com/epam/edp-keycloak-operator/issues/141))
- Publish on OperatorHub ([#132](https://github.com/epam/edp-keycloak-operator/issues/132))
- Update current development version ([#132](https://github.com/epam/edp-keycloak-operator/issues/132))


<a name="v1.25.0"></a>
## [v1.25.0] - 2025-02-13
### Features
- Add adminEventsExpiration to KeycloakRealm realmEventConfig ([#122](https://github.com/epam/edp-keycloak-operator/issues/122))

### Bug Fixes
- Realm HTML Display Name not properly set
- Add Resty HTTP client to keycloak_go_client.Client
- Spelling mistake in keycloak client deletion

### Routine
- Publish 1.24.0 on the OperatorHub ([#123](https://github.com/epam/edp-keycloak-operator/issues/123))
- Update current development version ([#123](https://github.com/epam/edp-keycloak-operator/issues/123))


<a name="v1.24.0"></a>
## [v1.24.0] - 2025-02-05
### Features
- Add Admin Fine Grained Permissions to Keycloak Client
- Add Browser and Direct Grant Flow fields to Keycloak Client
- Add realm SMTP configuration ([#96](https://github.com/epam/edp-keycloak-operator/issues/96))
- Add realm SMTP configuration ([#96](https://github.com/epam/edp-keycloak-operator/issues/96))
- Add setting adminUrl homeUrl for Client ([#106](https://github.com/epam/edp-keycloak-operator/issues/106))
- Add the ability to manage Realm Attributes ([#85](https://github.com/epam/edp-keycloak-operator/issues/85))
- Add print columns for KeycloakRealm Resources ([#109](https://github.com/epam/edp-keycloak-operator/issues/109))
- Add managing Authorization Resources for a Client ([#75](https://github.com/epam/edp-keycloak-operator/issues/75))

### Bug Fixes
- Boolean parameters with default values are always 'true' ([#56](https://github.com/epam/edp-keycloak-operator/issues/56))

### Routine
- Update current development version ([#102](https://github.com/epam/edp-keycloak-operator/issues/102))
- Update current development version ([#102](https://github.com/epam/edp-keycloak-operator/issues/102))


<a name="v1.23.0"></a>
## [v1.23.0] - 2024-10-29
### Features
- Add DisplayName to KeycloakRealm/ClusterKeycloakRealm ([#94](https://github.com/epam/edp-keycloak-operator/issues/94))
- Add support for optional client scopes
- Add childRequirement for KeycloakAuthFlow ([#82](https://github.com/epam/edp-keycloak-operator/issues/82))
- Remove deprecated v1alpha1 versions from the operator ([#86](https://github.com/epam/edp-keycloak-operator/issues/86))
- Add displayHTMLName to realm resource ([#80](https://github.com/epam/edp-keycloak-operator/issues/80))

### Bug Fixes
- Deletion resources related to subgroup ([#95](https://github.com/epam/edp-keycloak-operator/issues/95))
- Resolve subgroup creation and assignment issues ([#95](https://github.com/epam/edp-keycloak-operator/issues/95))

### Routine
- Update Pull Request Template ([#17](https://github.com/epam/edp-keycloak-operator/issues/17))
- Update KubeRocketCI names and documentation links ([#91](https://github.com/epam/edp-keycloak-operator/issues/91))
- Publish update on OperatorHub ([#76](https://github.com/epam/edp-keycloak-operator/issues/76))
- Add additional examples of Keycloak AuthFlow resource ([#79](https://github.com/epam/edp-keycloak-operator/issues/79))
- Update current development version ([#76](https://github.com/epam/edp-keycloak-operator/issues/76))


<a name="v1.22.0"></a>
## [v1.22.0] - 2024-07-23
### Features
- Add ClusterKeycloakRealm browserFlow setting ([#66](https://github.com/epam/edp-keycloak-operator/issues/66))

### Bug Fixes
- move imagePullSecrets to spec.template.spec ([#73](https://github.com/epam/edp-keycloak-operator/issues/73))
- Error if KeycloakClient secret is deleted before it ([#62](https://github.com/epam/edp-keycloak-operator/issues/62))
- KeycloakRealmRole CR duplicated status ([#68](https://github.com/epam/edp-keycloak-operator/issues/68))

### Routine
- Generate OperatorHub bundle for v1.21.0 ([#59](https://github.com/epam/edp-keycloak-operator/issues/59))
- Update current development version ([#59](https://github.com/epam/edp-keycloak-operator/issues/59))


<a name="v1.21.0"></a>
## [v1.21.0] - 2024-05-16
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

### Bug Fixes
- Remove from code coverage mock files ([#28](https://github.com/epam/edp-keycloak-operator/issues/28))

### Testing
- Add integration tests for KeycloakClientScope ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Add integration tests for KeycloakRealmUser ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Add integration tests for KeycloakAuthFlow ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Add integration tests for KeycloakRealm ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))

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

### Documentation
- Update README md file ([#132](https://github.com/epam/edp-keycloak-operator/issues/132))


<a name="v1.20.0"></a>
## [v1.20.0] - 2024-01-11
### Features
- Add missing fields to KeycloakClient ([#24](https://github.com/epam/edp-keycloak-operator/issues/24))

### Bug Fixes
- The default realm role is no longer works ([#22](https://github.com/epam/edp-keycloak-operator/issues/22))

### Routine
- Add printcolumn status for all custom resources ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Update current development version ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Generate bundle for OperatorHub v1.19.0 ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))


<a name="v1.19.0"></a>
## [v1.19.0] - 2023-11-15
### Features
- Enable secret reference support in KeycloakClient resource ([#21](https://github.com/epam/edp-keycloak-operator/issues/21))
- Add annotation for preserving resources deletion ([#18](https://github.com/epam/edp-keycloak-operator/issues/18))
- Enable secret support in KeycloakRealmIdentityProvider resource ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))

### Bug Fixes
- KeycloakRealmIdentityProvider config secret reference is replaced by the plain secret ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))

### Testing
- Create client without specifying client secret ([#21](https://github.com/epam/edp-keycloak-operator/issues/21))
- Add e2e for KeycloakRealmIdentityProvider using secret ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))

### Routine
- Generate bundle for OperatorHub v1.19.0 ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Update GH actions and release pipeline ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Upgrade pull request template ([#17](https://github.com/epam/edp-keycloak-operator/issues/17))
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#16](https://github.com/epam/edp-keycloak-operator/issues/16))
- Upgrade Go to 1.20 ([#14](https://github.com/epam/edp-keycloak-operator/issues/14))
- Update current development version ([#13](https://github.com/epam/edp-keycloak-operator/issues/13))


<a name="v1.18.2"></a>
## [v1.18.2] - 2023-10-31
### Routine
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#16](https://github.com/epam/edp-keycloak-operator/issues/16))


<a name="v1.18.1"></a>
## [v1.18.1] - 2023-09-25
### Routine
- Upgrade Go to 1.20 ([#14](https://github.com/epam/edp-keycloak-operator/issues/14))
- Update CHANGELOG.md ([#85](https://github.com/epam/edp-keycloak-operator/issues/85))


<a name="v1.18.0"></a>
## [v1.18.0] - 2023-09-20
### Features
- Allow multiple redirect URIs ([#12](https://github.com/epam/edp-keycloak-operator/issues/12))

### Routine
- Publish v1.17.1 on OperatorHub ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))
- Publish v1.17.0 on OperatorHub ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))
- Update current development version ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))


<a name="v1.17.1"></a>
## [v1.17.1] - 2023-09-04
### Routine
- Publish v1.17.0 on OperatorHub ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))


<a name="v1.17.0"></a>
## [v1.17.0] - 2023-08-17

[Unreleased]: https://github.com/epam/edp-keycloak-operator/compare/v1.31.1...HEAD
[v1.31.1]: https://github.com/epam/edp-keycloak-operator/compare/v1.31.0...v1.31.1
[v1.31.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.30.0...v1.31.0
[v1.30.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.29.0...v1.30.0
[v1.29.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.28.0...v1.29.0
[v1.28.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.27.1...v1.28.0
[v1.27.1]: https://github.com/epam/edp-keycloak-operator/compare/v1.27.0...v1.27.1
[v1.27.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.26.0...v1.27.0
[v1.26.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.25.0...v1.26.0
[v1.25.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.24.0...v1.25.0
[v1.24.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.23.0...v1.24.0
[v1.23.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.22.0...v1.23.0
[v1.22.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.21.0...v1.22.0
[v1.21.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.20.0...v1.21.0
[v1.20.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.19.0...v1.20.0
[v1.19.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.18.2...v1.19.0
[v1.18.2]: https://github.com/epam/edp-keycloak-operator/compare/v1.18.1...v1.18.2
[v1.18.1]: https://github.com/epam/edp-keycloak-operator/compare/v1.18.0...v1.18.1
[v1.18.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.17.1...v1.18.0
[v1.17.1]: https://github.com/epam/edp-keycloak-operator/compare/v1.17.0...v1.17.1
[v1.17.0]: https://github.com/epam/edp-keycloak-operator/compare/v1.16.0...v1.17.0
