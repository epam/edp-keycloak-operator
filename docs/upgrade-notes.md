# Upgrade notes

This page lists the changes that need attention when you upgrade the operator or Keycloak.

## How to use this page

1. Find your current operator version and Keycloak version.
2. Check the [compatibility](#compatibility) table for the target versions.
3. Read each [operator release](#operator-releases) section after your current version, oldest first.
4. Do the checks in "Before you upgrade" of each section.
5. Upgrade the operator with the [operator upgrade procedure](#operator-upgrade-procedure).
6. Do the checks in "After you upgrade" of each section.
7. Upgrade Keycloak after the operator, if necessary. Read [Keycloak upgrades](#keycloak-upgrades) first.

Do not upgrade the operator and Keycloak at the same time. The operator must support the Keycloak version
before and after each step.

The queries on this page need `kubectl` and `jq`.

## Compatibility

| Operator | Keycloak |
|---|---|
| 1.36.0 | 26.4 to 26.8 |
| 1.35.0 and earlier | 26.4 to 26.7 (not tested on 26.8) |

The "Keycloak" column in each release section shows the Keycloak versions that a change applies to.

## Operator upgrade procedure

Helm does not upgrade CRDs. Stop the operator before you replace the CRDs, so that the old operator does
not reconcile resources under the new schema.

1. Do the checks in "Before you upgrade" of each release section.
2. Stop the operator: `kubectl scale deployment <name> -n <namespace> --replicas=0`. The default
   `<name>` is `keycloak-operator`. With GitOps, stop the automatic sync of the operator first.
3. Replace the CRDs with those of the target chart version: `kubectl replace -f deploy-templates/crds/`.
   Create a CRD that is not on the cluster yet: `kubectl create -f <file>`.
4. Upgrade the chart: `helm upgrade`. Helm sets the replica count from the chart values again.
5. Make sure that the operator pod runs.
6. Do the checks in "After you upgrade" of each release section.

## Operator releases

### 1.35.0 to 1.36.0

Supported Keycloak versions: 26.4 to 26.8.

Required: Kubernetes 1.32 or later, or OpenShift 4.19 or later. `helm install` and `helm upgrade` fail on an
older cluster.

#### Before you upgrade

**1. `KeycloakClient` with `redirectUris: []` or `webOrigins: []`** ([#413](https://github.com/epam/edp-keycloak-operator/issues/413))

1.36.0 sends an empty list to Keycloak, and Keycloak clears the list. Login (`redirectUris`) and CORS
(cross-origin requests, `webOrigins`) then fail for the client. 1.35.0 read `[]` as a missing field.

| Field in the resource | 1.35.0 sent | 1.36.0 sends |
|---|---|---|
| `[]` | The value from `webUrl`. Nothing if `webUrl` is empty. | `[]`. Keycloak clears the list. |
| Omitted | The value from `webUrl`. Nothing if `webUrl` is empty. | The same. |
| A list with items | The list. | The same. |

`redirectUris` derives `<webUrl>/*` and `webOrigins` derives `<webUrl>`. Only the two lists change. Other
client settings stay as they are.

Typical resources with `[]`: a manifest in Git, or a Helm chart whose values default to an empty list and
render `redirectUris: []`.

```yaml
kind: KeycloakClient
spec:
  clientId: portal
  webUrl: https://portal.example.org
  webOrigins: []   # 1.35.0 sent ["https://portal.example.org"]. 1.36.0 clears the list.
```

1.35.0 can remove `[]` from the stored resource on its first reconcile. Then the list is not cleared. A
client applied again (GitOps, `kubectl apply`) has `[]` in the cluster and is cleared on the first
reconcile after the upgrade.

```sh
kubectl get keycloakclients -A -o json | jq -r '.items[] | select(.spec.redirectUris == [] or .spec.webOrigins == []) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Also find `redirectUris: []` and `webOrigins: []` in the manifests in Git. For Helm charts, check the
output of `helm template`. The next apply stores `[]` again. To keep the `webUrl` value, remove the key.
To keep a fixed list, write the list. To clear the list on purpose, keep `[]`.

**2. `KeycloakRealmComponent` with `parentRef`** ([#423](https://github.com/epam/edp-keycloak-operator/issues/423))

- Kind `KeycloakRealmComponent`: `parentRef.name` is the `metadata.name` of the parent CR in the same
  namespace. 1.33.0 to 1.35.0 used it as the Keycloak name of the parent component.
- Kind `KeycloakRealm`: `parentRef.name` must be the same as `spec.realmRef.name`.

A child that does not resolve shows the error in `status.value` and is retried every 30 seconds.
The operator does not change the child in Keycloak.

```sh
kubectl get keycloakrealmcomponents -A -o json | jq -r '(.items | map("\(.metadata.namespace)/\(.metadata.name)")) as $crs | .items[] | select(.spec.parentRef != null) | select(if (.spec.parentRef.kind // "KeycloakRealm") == "KeycloakRealm" then .spec.parentRef.name != .spec.realmRef.name else ("\(.metadata.namespace)/\(.spec.parentRef.name)" | IN($crs[]) | not) end) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Set `parentRef.name` of each listed component to the parent CR name, or to `spec.realmRef.name`.

**3. `KeycloakAuthFlow` config values that start with `$`** ([#407](https://github.com/epam/edp-keycloak-operator/issues/407))

`authenticatorConfig.config` now accepts Secret references (`$secretName:secretKey`). A value that starts
with `$` is always read as a reference. Values that start with `${` are not changed. There is no escape
syntax. A value such as `$abc` fails the reconcile, and Keycloak keeps the old value. 1.35.0 sent the
value to Keycloak as text.

```sh
kubectl get keycloakauthflows -A -o json | jq -r '.items[] | select([.spec.authenticationExecutions[]?.authenticatorConfig.config // {} | .[] | select(startswith("$") and (startswith("${") | not))] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"'
```

For each listed flow, change the value to a Secret reference.

If you set `allowedDestinationHosts`, the operator now also checks the destination values in
`authenticatorConfig.config`. Add the hosts of your auth flows to the list before you upgrade. See
[security.md](security.md).

**4. `KeycloakRealm` and `ClusterKeycloakRealm` with duplicate list entries**

`localization.supportedLocales`, `realmEventConfig.enabledEventTypes` and
`realmEventConfig.eventsListeners` are now sets. The API server rejects a duplicate entry.

```sh
for k in keycloakrealms clusterkeycloakrealms; do kubectl get $k -A -o json | jq -r '.items[] | select([.spec.localization.supportedLocales, .spec.realmEventConfig.enabledEventTypes, .spec.realmEventConfig.eventsListeners | select(. != null) | length != (unique | length)] | any) | "\(.kind) \(.metadata.namespace // "-")/\(.metadata.name)"'; done
```

Duplicates that are already stored stay accepted, and the query finds them. Remove the duplicate entries
from the listed resources and from their manifests in Git.

**5. `KeycloakOrganization` with `spec.domains`**

The operator copies `domains` into `domainsV2` one time and then reads only `domainsV2`. Later edits
of `domains` are ignored. Move the list from `domains` to `domainsV2` in the manifests in Git at upgrade
time.

**6. CRDs**

1.36.0 needs the new CRDs. With the old `KeycloakOrganization` CRD, the API server drops `domainsV2`,
and the reconcile of each `KeycloakOrganization` with `spec.domains` fails with an error that asks for
the CRD upgrade. The old CRDs also drop `Keycloak` `spec.tlsServerName` and `status.value`. Use the [operator upgrade procedure](#operator-upgrade-procedure).

**7. Empty or blank `clientId` in `clientRoles` and empty or blank `compositesClientRoles` keys**

The API server now rejects an empty `clientId` in `KeycloakRealmUser` and `KeycloakRealmGroup` `clientRoles` and
in `KeycloakClient` `serviceAccount.clientRoles` and `scopeMappings.clientRoles`. It does not reject a blank
`clientId` (spaces only) or an empty or blank `compositesClientRoles` key. For a user or service-account entry
with `roles`, operator versions 1.33.0 to 1.35.0 assigned the roles of the first client of the realm, and
`roles: []` removed all roles of that client. For a composite `KeycloakRealmRole` with an empty or blank
`compositesClientRoles` key, they added composites from that client. The first client is the one with the
lowest `clientId` in sort order, because Keycloak orders the client list by `clientId`. In a default realm it
is `account`. 1.36.0 fails the reconcile of such an entry or key.

```sh
for k in keycloakrealmusers keycloakrealmgroups; do kubectl get $k -A -o json | jq -r '.items[] | select([.spec.clientRoles[]? | select((.clientId // "") | test("^\\s*$"))] | length > 0) | "\(.kind) \(.metadata.namespace)/\(.metadata.name)"'; done
kubectl get keycloakclients -A -o json | jq -r '.items[] | select([.spec.serviceAccount.clientRoles[]?, .spec.scopeMappings.clientRoles[]? | select((.clientId // "") | test("^\\s*$"))] | length > 0) | "\(.kind) \(.metadata.namespace)/\(.metadata.name)"'
kubectl get keycloakrealmroles -A -o json | jq -r '.items[] | select(.spec.compositesClientRoles // {} | any(keys[]; test("^\\s*$"))) | "\(.kind) \(.metadata.namespace)/\(.metadata.name)"'
```

Fix the listed resources before you upgrade. An entry with an empty `clientId` that is already stored stays
stored, but any later change to the list, including the operator's own update, is rejected while the entry is
empty. Set `clientId` in the listed resources, and set or remove the blank key in the listed roles, in the
cluster and in the manifests in Git. For each listed user and service account, find the first client with
`GET /admin/realms/<realm>/clients?first=0&max=1`. Check the client roles of that client in Keycloak, not only
of `account`, remove the roles that you did not assign, and assign again the roles that a `roles: []` entry
removed. The operator does not do either. For a listed role, the operator removes the composites from that
client after you fix the key. If you set `composite: false` instead, remove those composites by hand.

**8. `KeycloakRealmIdentityProvider` with `mappers[].identityProviderAlias`**

1.36.0 fails the reconcile before any Keycloak write when a mapper sets `identityProviderAlias` to a value other
than `spec.alias`. The field is deprecated. The operator always uses `spec.alias`. Until the field is fixed, the
operator does not create or update the identity provider, so other spec changes and rotated Secret values in
`spec.config` are not sent.

With Keycloak 26.4 to 26.7, 1.35.0 and earlier created such a mapper under the other alias, also when no identity provider has
that alias. The resource cannot see that mapper, and each later reconcile failed with "Failed to add mapper". The
mapper stays in Keycloak after the upgrade, and it applies to logins through the identity provider of that alias.

Find the affected resources:

```sh
kubectl get keycloakrealmidentityproviders -A -o json | jq -r '.items[] | select(.spec.alias as $a | [.spec.mappers[]? | select((.identityProviderAlias // "") != "" and .identityProviderAlias != $a)] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Find every resource that sets the field:

```sh
kubectl get keycloakrealmidentityproviders -A -o json | jq -r '.items[] | select([.spec.mappers[]? | select((.identityProviderAlias // "") != "")] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"'
```

For each resource from the first list, do steps 1 and 2. For each resource from the second list, do step 1.

1. Remove `identityProviderAlias` from each mapper in the resource and in its manifests in Git. Do this first: with
   1.35.0 or earlier and the old spec, the next reconcile creates a deleted mapper again.
2. With Keycloak 26.4 to 26.7, delete the mapper that 1.35.0 or earlier created under the other alias `<X>`:
   - If an identity provider with alias `<X>` exists, list its mappers and delete the one with the name from the
     resource:
     `GET /admin/realms/<realm>/identity-provider/instances/<X>/mappers`, then
     `DELETE /admin/realms/<realm>/identity-provider/instances/<X>/mappers/<id>` (returns 204).
   - If no identity provider has alias `<X>`, the list returns 404. Create a temporary, disabled identity provider
     with alias `<X>`, list its mappers, delete the mapper, then delete the temporary identity provider. Create it
     with `POST /admin/realms/<realm>/identity-provider/instances` and this body (returns 201 on Keycloak 26.7.3):
     `{"alias":"<X>","providerId":"keycloak-oidc","enabled":false,"config":{"clientId":"x","clientSecret":"y","authorizationUrl":"https://example.invalid/auth","tokenUrl":"https://example.invalid/token"}}`.
     Or, if you
     know the mapper ID, delete it through the path of any existing identity provider:
     `DELETE /admin/realms/<realm>/identity-provider/instances/<any-existing-alias>/mappers/<id>`.
     Keycloak looks up the mapper by ID only.

After the upgrade to 1.36.0, a resource with alias `<X>` that lists its mappers (`mappers` set, also `[]`) removes
the orphan mapper from `<X>` on its next reconcile, because the name is not in that list. 1.35.0 skips the removal
when `mappers` is `[]`.

#### After you upgrade

**1. Realm roles that 1.35.0 removed** ([#371](https://github.com/epam/edp-keycloak-operator/issues/371), [#374](https://github.com/epam/edp-keycloak-operator/issues/374))

1.35.0 removed all realm roles from each `KeycloakRealmUser` with no `spec.roles`, unless
`reconciliationStrategy` was `addOnly`. This includes `default-roles-<realm>`, and with it
`offline_access` and `uma_authorization`. 1.36.0 does not manage an omitted list, so the roles stay
removed.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.spec.roles == null and .spec.reconciliationStrategy != "addOnly") | "\(.metadata.namespace)/\(.metadata.name)"'
```

The query also lists users that 1.35.0 did not reconcile. Check the realm roles of each listed user in
Keycloak. Give back the missing roles in Keycloak, or add them to `spec.roles`. The same applies to
group memberships (`spec.groups`) and to the realm roles of `KeycloakRealmGroup` (`spec.realmRoles`).

**2. `KeycloakRealmUser` with `attributesV2`** ([#418](https://github.com/epam/edp-keycloak-operator/issues/418))

Keycloak drops attributes that the realm user profile does not allow, and returns success. 1.36.0 reads
the user back. If an attribute is not stored, `status.value` shows "keycloak did not store user
attributes". 1.35.0 showed `OK`. Roles, groups and passwords are still applied.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.status.value // "" | contains("did not store user attributes")) | "\(.metadata.namespace)/\(.metadata.name)"'
```

For each listed user, set `spec.userProfileConfig.unmanagedAttributePolicy` of the realm to
`ADMIN_EDIT` or `ENABLED`, or declare the attributes in `spec.userProfileConfig.attributes`. Keycloak
26.4 stores the attributes also with `ADMIN_VIEW`. Keycloak 26.5 and later do not.

**3. `KeycloakRealmUser` with an empty `passwordSecret`** ([#424](https://github.com/epam/edp-keycloak-operator/issues/424))

Before 1.31, the operator stored `passwordSecret: {name: "", key: ""}` in each user. 1.36.0 removes
this entry one time. If the entry is in Git, remove it there. Otherwise GitOps adds it again on each
sync, and the operator removes it again.

**4. `KeycloakRealmUser` without `spec.enabled`** ([#375](https://github.com/epam/edp-keycloak-operator/issues/375))

1.35.0 created and kept a `KeycloakRealmUser` without `spec.enabled` disabled. 1.36.0 does not change
`enabled` of an existing user, so the user stays disabled.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.spec.enabled == null) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Check each listed user in Keycloak. Where the user must be enabled, set `enabled: true` in the spec.

#### Behavior changes

| Resource | Keycloak | Change |
|---|---|---|
| `KeycloakRealmUser` | All | `spec.enabled` has no default. `true` and `false` are written on create and when they differ from Keycloak, and `false` stays in the stored resource after operator writes. Without the field, the operator creates the user enabled and does not change `enabled` of an existing user. A user without the field that 1.35.0 created or kept disabled stays disabled until you set `enabled: true`. 1.35.0 wrote `false` for a missing field and removed a stored `false`. Removing `enabled` from the spec stops management, and Keycloak keeps the last value. The operator does not send `enabled` when it equals the Keycloak value, so brute-force login failures and a temporary lockout stay. A user disabled in Keycloak, also by permanent lockout, is enabled again when `spec.enabled` is `true`, and Keycloak clears its failures. |
| `KeycloakRealmUser` | All | `spec.emailVerified` is not managed when you omit it. Without the field, the operator creates the user unverified and does not change `emailVerified` of an existing user. 1.35.0 wrote `false` for a missing field on every update and removed a stored `false`. A user that 1.35.0 unverified stays unverified until the user verifies the email again or you set `emailVerified: true`. `true` and `false` are written on create and when they differ from Keycloak, and `false` stays in the stored resource after operator writes. Removing `emailVerified` from the spec stops management, and Keycloak keeps the last value. A change of `email` keeps the flag when `emailVerified` is omitted. `emailVerified: false` makes the user unverified, because Keycloak does not reset the flag on an admin update. |
| `KeycloakRealmUser`, `KeycloakRealmGroup`, `KeycloakClient` | All | These lists are not managed when you omit them: user `roles` and `groups`, group `realmRoles`, the `roles` of each `clientRoles` entry, and client `serviceAccount.realmRoles`. `[]` removes all entries. |
| Helm charts that create the resources above | All | Helm drops an empty list. To send `[]`, use `--set-json`, and do not put the value in an `{{ if }}` block. |
| `KeycloakRealmUser` | All | A `passwordSecret` with an empty `name` or `key` fails with a Secret error. 1.35.0 used `spec.password`. |
| `KeycloakClient` | All | `redirectUris` and `webOrigins`: no key sends the value from `webUrl`. If `webUrl` is empty, Keycloak keeps its list. `[]` clears the list. A list with items is sent as it is. |
| `KeycloakClient` | All | `spec.realmRoles` is deprecated. Use `KeycloakRealmRole`. A new admission webhook shows a warning for the field. The webhook does not block requests. |
| `KeycloakClient` | All | Authorization permissions are not deleted and created again on each reconcile. Their IDs do not change. An aggregate policy that names a permission fails with "policy <name> does not exist". Keycloak cannot aggregate a permission. |
| `KeycloakAuthFlow` | All | A rotated Secret is sent on the next reconcile. |
| `KeycloakRealm`, `ClusterKeycloakRealm` | All | `smtpServer.port` has no default. The operator sends the port only when you set it. Keycloak then uses 465 with SSL and 25 without. A resource that the old CRD defaulted to 25 keeps the stored 25. |
| `KeycloakRealm`, `ClusterKeycloakRealm` | All | `displayName`, `displayNameHtml`, `organizationsEnabled` and `realmEventConfig.adminEventsExpiration` are not managed when you omit them. The operator keeps the Keycloak value. An explicit empty string, `false` or `0` is still written. A resource created under the old CRD has `organizationsEnabled: false` stored. It keeps managing the flag until you remove the field from the spec. |
| Go packages `api/...` of module `github.com/epam/edp-keycloak-operator` | All | Types are in `api/v1` unless noted. These fields change from `[]string` to `*[]string`: `KeycloakClientSpec.RedirectUris`, `WebOrigins`, `KeycloakRealmUserSpec.Roles`, `Groups`, `KeycloakRealmGroupSpec.RealmRoles`, `UserClientRole.Roles`, `ServiceAccount.RealmRoles`. These fields change from `bool` to `*bool`: `KeycloakRealmUserSpec.Enabled` and `EmailVerified`, `OrganizationsEnabled` of `KeycloakRealmSpec` and of `ClusterKeycloakRealmSpec` in `api/v1alpha1`. These fields change from `string` to `*string`: `DisplayName` and `DisplayHTMLName` of both realm specs. These fields change from `int` to `*int`: `EmailConnection.Port` and `RealmEventConfig.AdminEventsExpiration` in `api/common`. |
| `KeycloakOrganization` | All | `domains` is deprecated. Use `domainsV2`. The operator copies `domains` into `domainsV2` one time. |
| `KeycloakOrganization` | All | The migration update drops an explicit `identityProviders: []` and `attributes: {}`, with the same meaning. It also drops empty strings from the stored spec (`description: ""`, `redirectUrl: ""`). A `domains` entry with an empty name fails the migration, and the operator log shows the error. Remove the empty entry. |
| `KeycloakOrganization` | All | `spec.attributes` replace the Keycloak attributes on each spec change. Attributes added in Keycloak stay until the next spec change. 1.35.0 replaced them on each reconcile. |
| `KeycloakOrganization` | All | An update keeps the Keycloak-side settings of the domains that the spec still lists. A spec edit does not reset them. |
| `KeycloakOrganization` | All | `domainsV2[].verified` is written after the identity provider links and repaired on drift. If you do not set it, the Keycloak value stays. |
| `KeycloakOrganization` | 26.8 | `domainsV2[].identityProviderAlias` and `autoRedirect` are written after the identity provider links and repaired on drift. If you do not set them, the Keycloak values stay. An empty `identityProviderAlias` clears the routing. |
| `KeycloakOrganization` | 26.4 to 26.7 | Keycloak rejects `domainsV2[].identityProviderAlias` and `autoRedirect` with HTTP 400 (unknown field). The status shows the Keycloak message. Do not set these fields. |
| `KeycloakOrganization` | 26.8 | `identityProviders[].autoMembership` and `membershipType` are applied to the organization link and repaired on drift. If you do not set them, the Keycloak values stay. A change sends both fields. |
| `KeycloakOrganization` | 26.4 to 26.7 | `identityProviders[].autoMembership` and `membershipType` fail the reconcile with an error that names the 26.8 requirement. The operator links the identity provider without these settings. Do not set these fields. |
| `KeycloakOrganization` | 26.8 | An identity provider can belong to more than one organization. |
| `KeycloakOrganization` | All | A `domainsV2[].identityProviderAlias` that is not in `identityProviders`, or the same domain two times with different settings, fails the reconcile before any Keycloak write. |
| `KeycloakOrganization` | All | An update sends the `enabled` value that Keycloak returned. An organization disabled in Keycloak stays disabled after an operator write. 1.35.0 enabled it again. |
| `KeycloakOrganization` | All | A link to an identity provider that is not in the realm fails before the link request. The error names `KeycloakRealmIdentityProvider`. |
| `KeycloakRealmIdentityProvider` | All | `mappers[].identityProviderAlias` is deprecated. A value other than `spec.alias` fails the reconcile before any Keycloak write. Until you remove the value, the identity provider is not created or updated, and a rotated Secret in `spec.config` is not sent. |

## Keycloak upgrades

### To Keycloak 26.8

Upgrade the operator to 1.36.0 first. 1.36.0 is the first operator version that supports Keycloak 26.8.

Keycloak 26.8 rejects some requests that 26.7 and earlier accepted. Resources that work now can fail
after the Keycloak upgrade. The status of the resource shows the Keycloak message.

#### Before you upgrade

| Resource | Keycloak 26.8 rule | Action |
|---|---|---|
| `KeycloakRealmIdentityProvider` with a mapper that grants an admin role or an admin group | Returns 403 unless `spec.config.allowAdminRoleMapping` is `"true"`. | Set the key. The operator account needs `manage-realm` on the realm. Keep the key in the spec while the mapper exists. See [security.md](security.md). |
| `KeycloakRealmIdentityProvider` with changed `providerId` | Rejected on an existing identity provider. | Create a new identity provider. |
| `Keycloak`, `ClusterKeycloak` with `spec.tlsServerName` | Returns 421 when all of these are true: the TLS server name is not the `Host` of the request, the server has no proxy headers, and `Host` is not the Keycloak `hostname` or `hostname-admin`. | Set the Keycloak `hostname` option to the host of `spec.url`, or configure proxy headers on the server. |
| `KeycloakRealmIdentityProvider` with `mappers[].identityProviderAlias` other than `spec.alias` | Returns 400. | Remove the field. 1.36.0 rejects the value before the request. See 1.35.0 to 1.36.0, item 8. |
