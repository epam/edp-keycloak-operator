<a name="unreleased"></a>
## [Unreleased]


<a name="v1.36.0"></a>
## v1.36.0 - 2026-11-01
### Features
- Add print columns for KeycloakRealm Resources ([#109](https://github.com/epam/edp-keycloak-operator/issues/109))
- Add DisplayName to KeycloakRealm/ClusterKeycloakRealm ([#94](https://github.com/epam/edp-keycloak-operator/issues/94))
- Allow multiple redirect URIs ([#12](https://github.com/epam/edp-keycloak-operator/issues/12))
- Enable secret support in KeycloakRealmIdentityProvider resource ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))
- Add annotation for preserving resources deletion ([#18](https://github.com/epam/edp-keycloak-operator/issues/18))
- Enable secret reference support in KeycloakClient resource ([#21](https://github.com/epam/edp-keycloak-operator/issues/21))
- Add missing fields to KeycloakClient ([#24](https://github.com/epam/edp-keycloak-operator/issues/24))
- Allow secret references in KeycloakRealmComponent ([#30](https://github.com/epam/edp-keycloak-operator/issues/30))
- Enable review for pull requests ([#32](https://github.com/epam/edp-keycloak-operator/issues/32))
- Allow creating Authorization Policies for a Client ([#28](https://github.com/epam/edp-keycloak-operator/issues/28))
- Allow creating Authorization Permissions for a Client ([#28](https://github.com/epam/edp-keycloak-operator/issues/28))
- Add custom certificate support ([#36](https://github.com/epam/edp-keycloak-operator/issues/36))
- Add ability to configure Realm token Settings ([#38](https://github.com/epam/edp-keycloak-operator/issues/38))
- Add Scopes to KeycloakClient Authorization spec ([#41](https://github.com/epam/edp-keycloak-operator/issues/41))
- Full reconciliation of KeycloakRealmUser  ([#45](https://github.com/epam/edp-keycloak-operator/issues/45))
- Remove SSORealm functionality from KeycloakRealm ([#47](https://github.com/epam/edp-keycloak-operator/issues/47))
- Add support for composite client role ([#44](https://github.com/epam/edp-keycloak-operator/issues/44))
- Add imagePullSecrets to enable private repository
- Add ClusterKeycloakRealm browserFlow setting ([#66](https://github.com/epam/edp-keycloak-operator/issues/66))
- Add displayHTMLName to realm resource ([#80](https://github.com/epam/edp-keycloak-operator/issues/80))
- Remove deprecated v1alpha1 versions from the operator ([#86](https://github.com/epam/edp-keycloak-operator/issues/86))
- Add childRequirement for KeycloakAuthFlow ([#82](https://github.com/epam/edp-keycloak-operator/issues/82))
- Add support for optional client scopes
- Add postBrokerLoginFlowAlias to KeycloakRealmIdentityProvider ([#231](https://github.com/epam/edp-keycloak-operator/issues/231))
- Add ability to set temporary user password ([#252](https://github.com/epam/edp-keycloak-operator/issues/252))
- Add hideOnLogin support for KeycloakRealmIdentityProvider
- Add managing Authorization Resources for a Client ([#75](https://github.com/epam/edp-keycloak-operator/issues/75))
- Add the ability to manage Realm Attributes ([#85](https://github.com/epam/edp-keycloak-operator/issues/85))
- Add setting adminUrl homeUrl for Client ([#106](https://github.com/epam/edp-keycloak-operator/issues/106))
- Add realm SMTP configuration ([#96](https://github.com/epam/edp-keycloak-operator/issues/96))
- Add realm SMTP configuration ([#96](https://github.com/epam/edp-keycloak-operator/issues/96))
- Add Browser and Direct Grant Flow fields to Keycloak Client
- Add Admin Fine Grained Permissions to Keycloak Client
- Add adminEventsExpiration to KeycloakRealm realmEventConfig ([#122](https://github.com/epam/edp-keycloak-operator/issues/122))
- Add login settings for KeycloakRealm and ClusterKeycloakRealm ([#166](https://github.com/epam/edp-keycloak-operator/issues/166))
- support hierarchical group paths in user groups
- Make ownerReference in Keycloak resources optional ([#71](https://github.com/epam/edp-keycloak-operator/issues/71))
- add description field to KeycloakRealmGroup ([#276](https://github.com/epam/edp-keycloak-operator/issues/276))
- Add the ability to add KeycloakClient service accounts to groups
- Introduce generated Keycloak client to replace gocloak
- Add SSO session settings support for KeycloakRealm ([#274](https://github.com/epam/edp-keycloak-operator/issues/274))
- add cluster domain for DNS name construction in kubernetes services and webhook certificates
- Add webhook defaulting for KeycloakClient resources ([#205](https://github.com/epam/edp-keycloak-operator/issues/205))
- Add Type field to KeycloakClientScope for default/optional/none scopes ([#220](https://github.com/epam/edp-keycloak-operator/issues/220))
- Add webhook validation for KeycloakRealm resources ([#259](https://github.com/epam/edp-keycloak-operator/issues/259))
- Add Conditions to KeycloakClient status ([#206](https://github.com/epam/edp-keycloak-operator/issues/206))
- Allow finalizer cleanup when realm is already deleted ([#173](https://github.com/epam/edp-keycloak-operator/issues/173))
- Add the ability to add client roles to KeycloakRealmUser ([#135](https://github.com/epam/edp-keycloak-operator/issues/135))
- Add support for Identity Providers in KeycloakRealmUser ([#148](https://github.com/epam/edp-keycloak-operator/issues/148))
- Enhance deletion logic to skip non-existent resources ([#238](https://github.com/epam/edp-keycloak-operator/issues/238))
- Add parentGroup to KeycloakRealmGroup for improved subgroup support ([#172](https://github.com/epam/edp-keycloak-operator/issues/172))
- Add Docker registry and podLabels to helm chart
- Add support for list of attributes with the same key
- Add ability to add Permissions to KeycloakRealmIdentityProvider
- Add KeycloakClient ClientRolesV2 field ([#152](https://github.com/epam/edp-keycloak-operator/issues/152))
- Introduce KeycloakOrganization feature ([#115](https://github.com/epam/edp-keycloak-operator/issues/115))
- Add validation for user identity provider ([#183](https://github.com/epam/edp-keycloak-operator/issues/183))
- **Keycloak:** Add spec.tlsServerName to verify the certificate by name
- **KeycloakAuthFlow:** migrate controller to keycloakv2 client
- **KeycloakAuthFlow:** Resolve secret references in authenticatorConfig.config
- **KeycloakClient:** migrate controller to keycloakv2 client
- **KeycloakClient:** add accessTokenLifespan to advancedSettings
- **KeycloakClientScope:** migrate controller to keycloakv2 client
- **KeycloakRealm:** Deprecate spec.users
- **KeycloakRealm:** Record realm internal ID in status
- **KeycloakRealm:** add localization support with customizable locales and message bundles
- **KeycloakRealmComponent:** migrate controller to keycloakv2 client
- **KeycloakRealmIdentityProvider:** migrate controller to keycloakv2 client
- **KeycloakRealmIdentityProvider:** change HideOnLogin to *bool
- **claude:** add the lane skill for isolated defect and feature work
- **helm:** make operator replica count configurable
- **helm:** add configurable serviceAccountName to operator chart
- **helm:** Expose SUCCESS_RECONCILE_TIMEOUT as successReconcileTimeout
- **keycloak:** add extensible auth spec to Keycloak and ClusterKeycloak
- **keycloak:** migrate Keycloak and ClusterKeycloak controllers to keycloakv2 client
- **keycloakapi:** Add client scope mapping methods
- **keycloakapi:** Update the generated client to Keycloak 26.8.1
- **keycloakapi:** Add organization identity provider link update
- **keycloakapi:** update the generated client to Keycloak 26.7.3
- **keycloakclient:** Add client-scope authorization policy
- **keycloakclient:** Manage client scope mappings
- **keycloakorganization:** migrate controller to keycloakv2 client
- **keycloakorganization:** Add domainsV2 with per-domain settings
- **keycloakorganization:** Manage identity provider link settings
- **keycloakorganization:** Keep Keycloak-side domain settings on update
- **keycloakrealm:** migrate realm controllers to keycloakv2 client
- **keycloakrealmrole:** migrate controller to keycloakv2 client
- **keycloakrealmuser:** migrate controller to keycloakv2 client
- **keycloakv2:** add groups, roles, and clients resource management
- **keycloakv2:** expand API coverage, add docs, and new resource domains
- **organization:** make alias field immutable
- **realm:** add brute force detection support
- **security:** add a destination allowlist for credential-bearing requests

### Bug Fixes
- Unable to set realm userProfileConfig ([#221](https://github.com/epam/edp-keycloak-operator/issues/221))
- keycloakrealmidentityprovider try to delete first in reconciliation loop_
- The default realm role is no longer works ([#22](https://github.com/epam/edp-keycloak-operator/issues/22))
- Remove from code coverage mock files ([#28](https://github.com/epam/edp-keycloak-operator/issues/28))
- KeycloakRealmRole CR duplicated status ([#68](https://github.com/epam/edp-keycloak-operator/issues/68))
- Error if KeycloakClient secret is deleted before it ([#62](https://github.com/epam/edp-keycloak-operator/issues/62))
- move imagePullSecrets to spec.template.spec ([#73](https://github.com/epam/edp-keycloak-operator/issues/73))
- Resolve subgroup creation and assignment issues ([#95](https://github.com/epam/edp-keycloak-operator/issues/95))
- Deletion resources related to subgroup ([#95](https://github.com/epam/edp-keycloak-operator/issues/95))
- Boolean parameters with default values are always 'true' ([#56](https://github.com/epam/edp-keycloak-operator/issues/56))
- Spelling mistake in keycloak client deletion
- Add Resty HTTP client to keycloak_go_client.Client
- Realm HTML Display Name not properly set
- KeycloakClient service account users groups aren't being populated correctly
- Do not force SMTP port 25 when port is omitted ([#422](https://github.com/epam/edp-keycloak-operator/issues/422))
- Don't delete Authorization settings on KeycloakClient when using addOnly strategy
- KeycloakRealmIdentityProvider config secret reference is replaced by the plain secret ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))
- Handle not found error when deleting KeycloakRealmUser ([#181](https://github.com/epam/edp-keycloak-operator/issues/181))
- KeycloakRealmIdentityProvider policies not being added to permissions
- Unable to update default browserFlow in master realm ([#143](https://github.com/epam/edp-keycloak-operator/issues/143))
- preserve spec zero values when updating finalizers
- Reconciliation KeycloakAuthFlow fails - authFlowConfig not found ([#234](https://github.com/epam/edp-keycloak-operator/issues/234))
- Proceed to next group in list if SubGroups is an empty slice
- Unable to get realm userProfileConfig ([#221](https://github.com/epam/edp-keycloak-operator/issues/221))
- Unable to create userProfileConfig without a group ([#165](https://github.com/epam/edp-keycloak-operator/issues/165))
- Client creation fails due to admin fine grained permissions ([#180](https://github.com/epam/edp-keycloak-operator/issues/180))
- Authentication flow priorities are not respected ([#235](https://github.com/epam/edp-keycloak-operator/issues/235))
- **ClusterKeycloakRealm:** report errors before the chain in status
- **ClusterKeycloakRealm:** Back off on chain failure instead of hot-looping
- **Keycloak:** accept a trailing slash in spec.url
- **KeycloakAuthFlow:** preserve execution priority on creation
- **KeycloakAuthFlow:** skip Keycloak writes when state already matches spec
- **KeycloakAuthFlow:** Skip masked keys in the authenticator config compare
- **KeycloakClient:** reject duplicate client scope names in a client-scope policy
- **KeycloakClient:** Restore realmRoles composite direction and deprecate the field
- **KeycloakClient:** report errors before the chain in status
- **KeycloakClient:** stop deleting authorization permissions as orphaned policies
- **KeycloakClient:** Back off on chain failure and persist failureCount
- **KeycloakClient:** Keep an explicit empty webOrigins and redirectUris
- **KeycloakClient:** reject duplicate authorization policy, permission and resource names
- **KeycloakClient:** reject duplicate entries in authorization string lists
- **KeycloakClient:** reject duplicate role and group names in authorization policies
- **KeycloakClientScope:** skip Keycloak writes when state already matches spec
- **KeycloakClientScope:** settle the attribute compare on blank values
- **KeycloakRealm:** skip status write after finalizer removal on deletion
- **KeycloakRealm:** enforce set semantics on locale and event list fields
- **KeycloakRealm:** skip Keycloak writes when state already matches spec
- **KeycloakRealm:** ignore duplicate entries when comparing locale and event lists
- **KeycloakRealmComponent:** Resolve component parentRef via the parent CR ([#423](https://github.com/epam/edp-keycloak-operator/issues/423))
- **KeycloakRealmComponent:** skip Keycloak writes when state already matches spec
- **KeycloakRealmGroup:** skip status write after finalizer removal on deletion
- **KeycloakRealmGroup:** prevent same-named groups from sharing one Keycloak group ID ([#362](https://github.com/epam/edp-keycloak-operator/issues/362))
- **KeycloakRealmGroup:** scope group ownership to the resolved realm instance
- **KeycloakRealmGroup:** resolve client roles' client through GetClientUUID
- **KeycloakRealmIdentityProvider:** require mapper name and identityProviderMapper
- **KeycloakRealmIdentityProvider:** skip Keycloak writes when state already matches spec
- **KeycloakRealmIdentityProvider:** reject a mapper identityProviderAlias other than spec.alias
- **KeycloakRealmRole:** skip status write after finalizer removal on deletion
- **KeycloakRealmUser:** default spec.enabled to true and keep an explicit false
- **KeycloakRealmUser:** leave spec.emailVerified unmanaged on update when omitted
- **KeycloakRealmUser:** keep brute-force failures when spec.enabled matches Keycloak
- **KeycloakRealmUser:** treat an unset role list as unmanaged, not empty
- **api:** reject clientRoles entries with an empty clientId
- **bundle:** publish to all current and future OpenShift versions
- **controllers:** pass realm group and role batch TryToDelete errors through
- **controllers:** report a failed realm finalizer add as such
- **controllers:** return an error instead of panicking on a realm GET without a JSON body
- **controllers:** match the clientId exactly in client lookups
- **controllers:** write status.value for errors before the chain
- **e2e:** Set the Keycloak hostname for the tlsServerName scenario
- **helper:** Compare canonical endpoints and effective TLS server names
- **helper:** skip the realm owner reference on a deleting resource
- **keycloak:** reject an id that Keycloak cannot address
- **keycloak:** reject a client role that Keycloak cannot address
- **keycloak:** treat unset group and realm role lists as unmanaged
- **keycloakapi:** Bound dials and TLS handshakes, close idle connections
- **keycloakapi:** route admin requests through resty.Request to enable retries
- **keycloakapi:** refuse HTTP redirects in the Keycloak client
- **keycloakapi:** Fail fast on TLS errors and an unparsable caCert
- **keycloakapi:** never resolve a group that has no id
- **keycloakclient:** skip the permissions update when the flag matches the spec
- **keycloakclient:** skip the client update when Keycloak matches the spec
- **keycloakclient:** name the Secret only in the generated secret Create error
- **keycloakclient:** Correct TimePolicy description and drop unused ScopePolicyData
- **keycloakorganization:** treat unset identityProviders as unmanaged
- **keycloakorganization:** reject Keycloak 26.8-only fields before any write on older Keycloak
- **keycloakorganization:** omit autoRedirect false from the domain write
- **keycloakorganization:** Name KeycloakRealmIdentityProvider when a linked identity provider is missing
- **keycloakorganization:** write status when the domains migration fails
- **keycloakrealm:** skip deleting the master realm
- **keycloakrealmcomponent:** Refuse to delete a component another CR manages
- **keycloakrealmcomponent:** Refuse to adopt a component another CR manages
- **keycloakrealmcomponent:** Match components by providerType
- **keycloakrealmgroup:** skip the group update when Keycloak matches the spec
- **keycloakrealmgroup:** treat unset clientRoles as unmanaged
- **keycloakrealmgroup:** omit an empty description from the group payload
- **keycloakrealmgroup:** use status ID to detect group renames
- **keycloakrealmrole:** sync isDefault membership of the realm default role
- **keycloakrealmuser:** leave spec.enabled unmanaged on update when omitted
- **keycloakrealmuser:** keep passwords synced before passwordSecretHash on upgrade
- **keycloakrealmuser:** write the password when the passwordSecret source changes
- **keycloakrealmuser:** Report unstored attributesV2
- **keycloakrealmuser:** Drop the empty passwordSecret stored before 1.31
- **keycloakv2:** redact credentials from debug log output
- **realm:** preserve externally-set display name and organizations flag on reconcile ([#376](https://github.com/epam/edp-keycloak-operator/issues/376))
- **realm:** use *bool and *int for RealmEventConfig nullable flags
- **realm:** preserve externally-set adminEventsExpiration when CR omits it
- **security:** correct the destination allowlist
- **security:** ship read-only helper roles for credential-forwarding CRs
- **security:** derive configSecretsHash from Secret versions, not secret values
- **webhook:** scope realm uniqueness to the resolved Keycloak instance ([#360](https://github.com/epam/edp-keycloak-operator/issues/360))
- **webhook:** scope realm uniqueness to KeycloakRef

### Code Refactoring
- update keycloak client package references from keycloakv2 to keycloakapi
- remove legacy gocloak adapter package
- add golangci-lint gofmt rewrite rule to enforce the use of `any`
- **controllers:** share the pre-chain reconcile flow of realm-scoped kinds
- **helper:** Share realm target resolution between controllers
- **keycloakapi:** move ServerVersionAtLeast out of testutils
- **keycloakapi:** Remove unused FindComponentByName

### Testing
- Add e2e for KeycloakRealmIdentityProvider using secret ([#20](https://github.com/epam/edp-keycloak-operator/issues/20))
- Create client without specifying client secret ([#21](https://github.com/epam/edp-keycloak-operator/issues/21))
- Add integration tests for KeycloakRealm ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Satisfy golangci-lint v2.14.0 in tests
- Add integration tests for KeycloakAuthFlow ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Add integration tests for KeycloakRealmUser ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Add integration tests for KeycloakClientScope ([#31](https://github.com/epam/edp-keycloak-operator/issues/31))
- Use Keycloak version 26.3 for tests ([#192](https://github.com/epam/edp-keycloak-operator/issues/192))
- **KeycloakOrganization:** match the stable part of the duplicate-domain error
- **KeycloakRealmUser:** wait for each failed login in the brute-force spec
- **envtest:** Stop the control plane in DeferCleanup
- **idempotency:** add regression suites for unnecessary Keycloak writes
- **integration:** give every integration suite a unique Keycloak realm name
- **keycloakapi:** retry GetRealms until Keycloak returns a complete body
- **keycloakapi:** Name test objects with testutils.UniqueName
- **keycloakorganization:** Allow repeat migration write
- **keycloakrealmidentityprovider:** Add idempotency and admin role mapper suites
- **security:** cover the destination allowlist against a running Keycloak

### Routine
- Publish update on OperatorHub ([#76](https://github.com/epam/edp-keycloak-operator/issues/76))
- Update current development version ([#76](https://github.com/epam/edp-keycloak-operator/issues/76))
- Update current development version ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))
- Publish v1.17.0 on OperatorHub ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))
- Publish v1.17.1 on OperatorHub ([#10](https://github.com/epam/edp-keycloak-operator/issues/10))
- Update current development version ([#13](https://github.com/epam/edp-keycloak-operator/issues/13))
- Upgrade Go to 1.20 ([#14](https://github.com/epam/edp-keycloak-operator/issues/14))
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#16](https://github.com/epam/edp-keycloak-operator/issues/16))
- Require Kubernetes 1.32 and OpenShift 4.19
- Update current development version ([#185](https://github.com/epam/edp-keycloak-operator/issues/185))
- Bump to Go 1.26.9
- Upgrade pull request template ([#17](https://github.com/epam/edp-keycloak-operator/issues/17))
- Update GH actions and release pipeline ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Generate bundle for OperatorHub v1.19.0 ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Update current development version ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Add printcolumn status for all custom resources ([#23](https://github.com/epam/edp-keycloak-operator/issues/23))
- Update current development version ([#27](https://github.com/epam/edp-keycloak-operator/issues/27))
- Generate OperatorHub bundle for the version 1.20.0 ([#27](https://github.com/epam/edp-keycloak-operator/issues/27))
- Implement cache in github workflow ([#34](https://github.com/epam/edp-keycloak-operator/issues/34))
- Remove explicit caching in workflows ([#34](https://github.com/epam/edp-keycloak-operator/issues/34))
- Add ClusterRoleBinding for operatorHub([#37](https://github.com/epam/edp-keycloak-operator/issues/37))
- Update operator bundle ([#37](https://github.com/epam/edp-keycloak-operator/issues/37))
- Bump google.golang.org/protobuf from 1.28.1 to 1.33.0 ([#39](https://github.com/epam/edp-keycloak-operator/issues/39))
- Migrate from gerrit to github pipelines ([#49](https://github.com/epam/edp-keycloak-operator/issues/49))
- Publish 1.28.0 on the OperatorHub ([#185](https://github.com/epam/edp-keycloak-operator/issues/185))
- Add CODEOWNERS ([#49](https://github.com/epam/edp-keycloak-operator/issues/49))
- Bump to Go 1.22 ([#57](https://github.com/epam/edp-keycloak-operator/issues/57))
- Update current development version ([#59](https://github.com/epam/edp-keycloak-operator/issues/59))
- Generate OperatorHub bundle for v1.21.0 ([#59](https://github.com/epam/edp-keycloak-operator/issues/59))
- Update current development version ([#285](https://github.com/epam/edp-keycloak-operator/issues/285))
- Add additional examples of Keycloak AuthFlow resource ([#79](https://github.com/epam/edp-keycloak-operator/issues/79))
- Update KubeRocketCI names and documentation links ([#91](https://github.com/epam/edp-keycloak-operator/issues/91))
- Update Pull Request Template ([#17](https://github.com/epam/edp-keycloak-operator/issues/17))
- Update current development version ([#102](https://github.com/epam/edp-keycloak-operator/issues/102))
- Update current development version ([#269](https://github.com/epam/edp-keycloak-operator/issues/269))
- Publish Operator to stable and alpha channels ([#263](https://github.com/epam/edp-keycloak-operator/issues/263))
- Update current development version ([#263](https://github.com/epam/edp-keycloak-operator/issues/263))
- Update current development version ([#253](https://github.com/epam/edp-keycloak-operator/issues/253))
- Update current development version ([#102](https://github.com/epam/edp-keycloak-operator/issues/102))
- Update krci-ai framework to the latest version ([#198](https://github.com/epam/edp-keycloak-operator/issues/198))
- Generate containerImage annotation for ClusterServiceVersion ([#225](https://github.com/epam/edp-keycloak-operator/issues/225))
- Update bundle manifests v1.29.0 ([#225](https://github.com/epam/edp-keycloak-operator/issues/225))
- Generate bundle manifests v1.29.0 ([#225](https://github.com/epam/edp-keycloak-operator/issues/225))
- Add development documentation ([#204](https://github.com/epam/edp-keycloak-operator/issues/204))
- Update KubeRocketAI ([#198](https://github.com/epam/edp-keycloak-operator/issues/198))
- Setup KubeRocketAI ([#198](https://github.com/epam/edp-keycloak-operator/issues/198))
- Refactor dependencies to use standard library alternatives ([#203](https://github.com/epam/edp-keycloak-operator/issues/203))
- Updated mockery from v2.53.2 to v3.5.4 ([#202](https://github.com/epam/edp-keycloak-operator/issues/202))
- Upgrade golangci-lint to v2 ([#199](https://github.com/epam/edp-keycloak-operator/issues/199))
- Update Operator SDK from v1.39.2 to v1.41.1 ([#199](https://github.com/epam/edp-keycloak-operator/issues/199))
- Update current development version ([#253](https://github.com/epam/edp-keycloak-operator/issues/253))
- Update current development version ([#123](https://github.com/epam/edp-keycloak-operator/issues/123))
- Bump golangci-lint and code generation tools
- Update codeql and codecov scan gh actions ([#178](https://github.com/epam/edp-keycloak-operator/issues/178))
- Publish on the OperatorHub ([#170](https://github.com/epam/edp-keycloak-operator/issues/170))
- Add multi-architecture build support ([#168](https://github.com/epam/edp-keycloak-operator/issues/168))
- Update current development version ([#160](https://github.com/epam/edp-keycloak-operator/issues/160))
- Update current development version ([#160](https://github.com/epam/edp-keycloak-operator/issues/160))
- Remove deprecated properties from CRs ([#154](https://github.com/epam/edp-keycloak-operator/issues/154))
- Publish 1.24.0 on the OperatorHub ([#123](https://github.com/epam/edp-keycloak-operator/issues/123))
- Bump GitHub Actions runner image to 22.04([#150](https://github.com/epam/edp-keycloak-operator/issues/150))
- Update current development version ([#146](https://github.com/epam/edp-keycloak-operator/issues/146))
- Make securityContext configurable via values.yaml ([#141](https://github.com/epam/edp-keycloak-operator/issues/141))
- Update current development version ([#132](https://github.com/epam/edp-keycloak-operator/issues/132))
- Publish on OperatorHub ([#132](https://github.com/epam/edp-keycloak-operator/issues/132))
- **KeycloakOrganization:** deprecate status.error and SetError
- **KeycloakRealmGroup:** migrate controller to keycloakv2 client with chain pattern
- **KeycloakRealmIdentityProvider:** deprecate mappers[].identityProviderAlias
- **api:** remove unused SSORealmMapper type
- **bundle:** regenerate OLM bundle for 1.36.0
- **bundle:** regenerate the OLM bundle for 1.36.0
- **bundle:** regenerate OLM bundle for 1.36.0
- **ci:** Publish 1.34 on operatorhub
- **ci:** update codecov configuration ([#269](https://github.com/epam/edp-keycloak-operator/issues/269))
- **ci:** update GitHub Actions workflows and testing infrastructure ([#280](https://github.com/epam/edp-keycloak-operator/issues/280))
- **ci:** align codecov configuration and enhance README badges ([#269](https://github.com/epam/edp-keycloak-operator/issues/269))
- **ci:** fix release action to make sure changelog available ([#280](https://github.com/epam/edp-keycloak-operator/issues/280))
- **ci:** Test against Keycloak 26.8
- **ci:** bump actions and Keycloak versions
- **ci:** Update development version
- **ci:** Update development version
- **deps:** Bump google.golang.org/grpc from 1.79.3 to 1.82.1
- **deps:** Bump google.golang.org/grpc from 1.83.1 to 1.83.2
- **deps:** Bump golang.org/x/net from 0.17.0 to 0.23.0
- **deps:** Bump go.opentelemetry.io/otel/sdk from 1.40.0 to 1.43.0
- **deps:** Bump golang.org/x/net from 0.48.0 to 0.55.0
- **deps:** Bump google.golang.org/grpc from 1.75.1 to 1.79.3
- **deps:** Bump go.opentelemetry.io/otel/sdk from 1.37.0 to 1.40.0
- **deps:** Bump github.com/google/cel-go from 0.23.2 to 0.29.0
- **deps:** Bump golang.org/x/oauth2 from 0.21.0 to 0.27.0
- **deps:** Bump github.com/golang-jwt/jwt/v4 from 4.5.1 to 4.5.2
- **deps:** Bump google.golang.org/grpc from 1.82.1 to 1.83.1
- **deps:** Upgrade operator-sdk to v1.39.2 ([#139](https://github.com/epam/edp-keycloak-operator/issues/139))
- **deps:** Bump go.opentelemetry.io/otel/sdk from 1.44.0 to 1.45.0
- **deps:** Bump go.opentelemetry.io/otel/exporters/otlp/otlptrace
- **deps:** Bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
- **deps:** Bump golang.org/x/net from 0.37.0 to 0.38.0
- **deps:** Bump github.com/golang-jwt/jwt/v4 from 4.4.3 to 4.5.1
- **deps:** Bump golang.org/x/net to v.0.33.0 ([#120](https://github.com/epam/edp-keycloak-operator/issues/120))
- **keycloak:** surface the connection result in Keycloak and ClusterKeycloak status
- **lint:** Forbid direct envtest Start and Stop in suites
- **release:** Update current development version

### Documentation
- add shared AI assistant context and reusable dev workflow skills
- restructure upgrade notes ([#476](https://github.com/epam/edp-keycloak-operator/issues/476))
- trim 1.36.0 upgrade notes and add known limits
- Update README md file ([#132](https://github.com/epam/edp-keycloak-operator/issues/132))
- add kubectl wait example for Keycloak status check ([#243](https://github.com/epam/edp-keycloak-operator/issues/243))
- Document Keycloak 26.8 support and admin role mappers
- Add security considerations for Secret references
- correct stale facts and de-prescribe agent instructions
- fix the auth flow query and add the helm upgrade command ([#476](https://github.com/epam/edp-keycloak-operator/issues/476))
- **KeycloakRealmIdentityProvider:** upgrade note and security note for the mapper alias
- **README:** add note about keycloakapi as standalone library
- **upgrade-notes:** add 1.36.0 upgrade notes
- **upgrade-notes:** KeycloakClient duplicate authorization entries

### Reverts
- [EPMDEDP-4226] Correctly update KeycloakClient CR to get correct .status.value after reconciliation

### BREAKING CHANGE:

omitting clientRoles no longer removes client role
mappings; write [] to remove all. KeycloakRealmGroupSpec.ClientRoles is
*[]UserClientRole.

omitting identityProviders no longer unlinks identity
providers. Write [] to unlink all. Helm drops []: pass --set-json, and do
not guard the value with {{ if }}.

Kubernetes 1.32 or later, or OpenShift 4.19 or later,
is required.

webOrigins: [] and redirectUris: [] now clear the list in
Keycloak. Remove the key to keep the webUrl fallback. CRs stored with [] are
cleared on the first reconcile after upgrade. Confidential clients the old
operator rewrote hold webOrigins: [webUrl] and no redirectUris in etcd;
re-apply [] to clear them.

parentRef.name for kind KeycloakRealmComponent is the

omitting either field no longer clears it. Write [] instead.
Helm drops []: pass --set-json, and do not guard the value with {{ if }}.

Webhook functionality now requires cert-manager
to be installed in the cluster by default.
Users can disable webhooks by setting \`enableWebhooks: false\` in Helm values.


[Unreleased]: https://github.com/epam/edp-keycloak-operator/compare/v1.36.0...HEAD
