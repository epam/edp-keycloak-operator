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

1.36.0 sends `[]` to Keycloak, and Keycloak clears the list. Login (`redirectUris`) and CORS (cross-origin
requests, `webOrigins`) then fail for the client. 1.35.0 read `[]` as a missing field.

1.35.0 can remove `[]` from the stored resource on its first reconcile. A client applied again (GitOps,
`kubectl apply`) has `[]` in the cluster and is cleared on the first reconcile after the upgrade.

```sh
kubectl get keycloakclients -A -o json | jq -r '.items[] | select(.spec.redirectUris == [] or .spec.webOrigins == []) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Also find `redirectUris: []` and `webOrigins: []` in the manifests in Git and in the output of `helm template`.
To derive the list from `webUrl`, remove the key. To keep a fixed list, write the list. To clear the list, keep `[]`.

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

`authenticatorConfig.config` accepts Secret references (`$secretName:secretKey`). A value that starts with `$`
is always read as a reference. Values that start with `${` are not changed. There is no escape syntax. A value
such as `$abc` fails the reconcile, and Keycloak keeps the old value. 1.35.0 sent the value as text.

```sh
kubectl get keycloakauthflows -A -o json | jq -r '.items[] | select([.spec.authenticationExecutions[]?.authenticatorConfig.config // {} | .[] | select(startswith("$") and (startswith("${") | not))] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"'
```

For each listed flow, change the value to a Secret reference.

With `allowedDestinationHosts` set, the operator also checks the destination values in
`authenticatorConfig.config`. Add the hosts of your auth flows to the list before you upgrade. See
[security.md](security.md).

**4. `KeycloakRealm` and `ClusterKeycloakRealm` with duplicate list entries**

`localization.supportedLocales`, `realmEventConfig.enabledEventTypes` and `realmEventConfig.eventsListeners`
are sets. The API server rejects a duplicate entry. Stored duplicates stay accepted.

```sh
for k in keycloakrealms clusterkeycloakrealms; do kubectl get $k -A -o json | jq -r '.items[] | select([.spec.localization.supportedLocales, .spec.realmEventConfig.enabledEventTypes, .spec.realmEventConfig.eventsListeners | select(. != null) | length != (unique | length)] | any) | "\(.kind) \(.metadata.namespace // "-")/\(.metadata.name)"'; done
```

Remove the duplicate entries from the listed resources and from their manifests in Git.

**5. `KeycloakOrganization` with `spec.domains`**

The operator copies `domains` into `domainsV2` one time and then reads only `domainsV2`. Later edits of
`domains` are ignored. A `domains` entry with an empty name fails the migration, and the operator log shows the
error. Remove the empty entry. Move the list from `domains` to `domainsV2` in the manifests in Git.

**6. CRDs**

1.36.0 needs the new CRDs. With the old `KeycloakOrganization` CRD, the API server drops `domainsV2`, and the
reconcile of each `KeycloakOrganization` with `spec.domains` fails with an error that asks for the CRD
upgrade. The old CRDs also drop `Keycloak` `spec.tlsServerName` and `status.value`. Use the
[operator upgrade procedure](#operator-upgrade-procedure).

**7. Empty or blank `clientId` in `clientRoles` and empty or blank `compositesClientRoles` keys**

The API server rejects an empty `clientId` in `KeycloakRealmUser` and `KeycloakRealmGroup` `clientRoles` and
in `KeycloakClient` `serviceAccount.clientRoles` and `scopeMappings.clientRoles`. It does not reject a blank
`clientId` (spaces only) or an empty or blank `compositesClientRoles` key. 1.36.0 fails the reconcile of such
an entry or key.

1.33.0 to 1.35.0 used the first client of the realm for a user or service-account entry with an empty or blank
`clientId`, and for an empty or blank `compositesClientRoles` key. A user or service-account entry
with `roles` got the roles of that client, and `roles: []` removed all roles of that client. A composite
`KeycloakRealmRole` got composites from that client. The first client has the lowest `clientId` in sort
order. In a default realm it is `account`.

```sh
for k in keycloakrealmusers keycloakrealmgroups; do kubectl get $k -A -o json | jq -r '.items[] | select([.spec.clientRoles[]? | select((.clientId // "") | test("^\\s*$"))] | length > 0) | "\(.kind) \(.metadata.namespace)/\(.metadata.name)"'; done
kubectl get keycloakclients -A -o json | jq -r '.items[] | select([.spec.serviceAccount.clientRoles[]?, .spec.scopeMappings.clientRoles[]? | select((.clientId // "") | test("^\\s*$"))] | length > 0) | "\(.kind) \(.metadata.namespace)/\(.metadata.name)"'
kubectl get keycloakrealmroles -A -o json | jq -r '.items[] | select(.spec.compositesClientRoles // {} | any(keys[]; test("^\\s*$"))) | "\(.kind) \(.metadata.namespace)/\(.metadata.name)"'
```

Fix the listed resources before you upgrade. A stored entry with an empty `clientId` stays stored. Any later
change to the list, including the operator's own update, is rejected while the entry is empty. Set `clientId`
in the listed resources, and set or remove the blank key in the listed roles, in the cluster and in the
manifests in Git. For each listed user and service account, find the first client with
`GET /admin/realms/<realm>/clients?first=0&max=1`. In Keycloak, remove the roles of that client that you did
not assign, and assign again the roles that a `roles: []` entry removed. The operator does neither. For a
listed role, the operator removes the composites from that client after you fix the key. If you set
`composite: false` instead, remove those composites by hand.

**8. `KeycloakRealmIdentityProvider` with `mappers[].identityProviderAlias`**

1.36.0 fails the reconcile before any Keycloak write when a mapper sets `identityProviderAlias` to a value other
than `spec.alias`. The field is deprecated. The operator always uses `spec.alias`. Until the field is fixed,
the operator does not create or update the identity provider. Other spec changes and rotated Secret values in
`spec.config` are not sent.

With Keycloak 26.4 to 26.7, 1.35.0 and earlier created such a mapper under the other alias, also when no
identity provider has that alias. The mapper stays in Keycloak after the upgrade. It applies to logins through
the identity provider of that alias.

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
     Or, if you know the mapper ID, delete it through the path of any existing identity provider:
     `DELETE /admin/realms/<realm>/identity-provider/instances/<any-existing-alias>/mappers/<id>`.
     Keycloak looks up the mapper by ID only.

After the upgrade, a resource with alias `<X>` removes the orphan mapper from `<X>` on its next reconcile when `mappers`
does not list its name. 1.35.0 skipped the removal when `mappers` is `[]`.

**9. `KeycloakRealmIdentityProvider` mapper without `name` or `identityProviderMapper`**

The API server rejects a mapper with an empty or missing `name` or `identityProviderMapper` (the mapper type).
It does not reject blank values (spaces only).

```sh
kubectl get keycloakrealmidentityproviders -A -o json | jq -r '.items[] | select([.spec.mappers[]? | select(((.name // "") | test("^\\s*$")) or ((.identityProviderMapper // "") | test("^\\s*$")))] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"'
```

The query also lists blank values (spaces only), which the operator sends to Keycloak as they are.

Set the name and the mapper type in the listed resources and in the manifests in Git. No manual cleanup in
Keycloak is needed. A stored entry stays stored. Any later change to `mappers` is rejected while the entry is
empty or missing. A Go client that reads and updates the whole resource is rejected while a stored entry has an
explicit empty value. Use a patch or fix the entry.

**10. `KeycloakClient` with duplicate authorization entries**

In `spec.authorization`, `policies`, `permissions` and `resources` are keyed by `name`, and so are
`rolePolicy.roles` and `groupPolicy.groups` of each policy. `scopes`, `clientPolicy.clients`,
`userPolicy.users`, `aggregatedPolicy.policies` and the `policies`, `resources` and `scopes` of each permission
are sets. The API server rejects a repeated name or entry.

```sh
kubectl get keycloakclients -A -o json | jq -r '.items[] | select([.spec.authorization // {} | (.scopes, ((.policies, .permissions, .resources) | select(. != null) | map(.name)), (.policies[]? | ((.rolePolicy.roles, .groupPolicy.groups) | select(. != null) | map(.name)), .clientPolicy.clients, .userPolicy.users, .aggregatedPolicy.policies), (.permissions[]? | .policies, .resources, .scopes)) | select(. != null) | length != (unique | length)] | any) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Remove the duplicate entries from the listed resources and from their manifests in Git. For policies,
permissions and resources that share a name, keep one entry. The operator writes the value of the kept entry
on the next reconcile.

With server-side apply (Flux, or Argo CD with `ServerSideApply=true`), fix the manifests in Git before you
replace the CRDs. With the new CRDs, the API server rejects an apply of a manifest that repeats a name or an
entry with HTTP 500 `duplicate entries for key`, also when the manifest has not changed.

A resource that stores a duplicate accepts updates, including a client-side `kubectl apply` and new
duplicates. The query finds it.

#### After you upgrade

**1. Realm roles that 1.35.0 removed** ([#371](https://github.com/epam/edp-keycloak-operator/issues/371), [#374](https://github.com/epam/edp-keycloak-operator/issues/374))

1.35.0 removed all realm roles from each `KeycloakRealmUser` with no `spec.roles`, unless
`reconciliationStrategy` was `addOnly`. This includes `default-roles-<realm>`, and with it `offline_access`
and `uma_authorization`. 1.36.0 does not manage an omitted list, so the roles stay removed.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.spec.roles == null and .spec.reconciliationStrategy != "addOnly") | "\(.metadata.namespace)/\(.metadata.name)"'
```

The query also lists users that 1.35.0 did not reconcile. Check the realm roles of each listed user in
Keycloak. Give back the missing roles in Keycloak, or add them to `spec.roles`. The same applies to group
memberships (`spec.groups`) and to the realm roles of `KeycloakRealmGroup` (`spec.realmRoles`).

**2. `KeycloakRealmUser` with `attributesV2`** ([#418](https://github.com/epam/edp-keycloak-operator/issues/418))

Keycloak drops attributes that the realm user profile does not allow, and returns success. 1.36.0 reads the
user back. If an attribute is not stored, `status.value` shows "keycloak did not store user attributes".
1.35.0 showed `OK`. Roles, groups and passwords are still applied.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.status.value // "" | contains("did not store user attributes")) | "\(.metadata.namespace)/\(.metadata.name)"'
```

For each listed user, set `spec.userProfileConfig.unmanagedAttributePolicy` of the realm to
`ADMIN_EDIT` or `ENABLED`, or declare the attributes in `spec.userProfileConfig.attributes`. Keycloak
26.4 stores the attributes also with `ADMIN_VIEW`. Keycloak 26.5 and later do not.

**3. `KeycloakRealmUser` with an empty `passwordSecret`** ([#424](https://github.com/epam/edp-keycloak-operator/issues/424))

Before 1.31, the operator stored `passwordSecret: {name: "", key: ""}` in each user. 1.36.0 removes this entry
one time. If the entry is in Git, remove it there. Otherwise GitOps adds it again on each sync, and the
operator removes it again.

**4. `KeycloakRealmUser` without `spec.enabled`** ([#375](https://github.com/epam/edp-keycloak-operator/issues/375))

1.35.0 created and kept a `KeycloakRealmUser` without `spec.enabled` disabled. 1.36.0 does not change
`enabled` of an existing user, so the user stays disabled.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.spec.enabled == null) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Check each listed user in Keycloak. Where the user must be enabled, set `enabled: true` in the spec.

**5. `KeycloakRealmUser` without `spec.emailVerified`**

1.35.0 wrote `emailVerified: false` on each update of a `KeycloakRealmUser` without the field. 1.36.0 does not
change `emailVerified` of an existing user, so the user stays unverified.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.spec.emailVerified == null) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Check each listed user in Keycloak. Where the email must be verified, set `emailVerified: true` in the spec.

#### Behavior changes

| Resource | Keycloak | Change |
|---|---|---|
| `KeycloakRealmUser` | All | `spec.enabled` and `spec.emailVerified` have no default and are not managed when you omit them. Without the field, the operator creates the user enabled and unverified, and does not change the flag of an existing user. With the field, the operator writes the flag on create and when it differs from Keycloak. The operator does not send `enabled` when it equals the Keycloak value. Brute-force login failures and a temporary lockout stay. A change of `email` keeps the flag when `emailVerified` is omitted. Set `emailVerified: false` to make the user unverified. |
| `KeycloakRealmUser`, `KeycloakRealmGroup`, `KeycloakClient`, `KeycloakOrganization` | All | These lists are not managed when you omit them: user `roles` and `groups`, group `realmRoles`, the `roles` of each `clientRoles` entry, client `serviceAccount.realmRoles`, and organization `identityProviders`. `[]` removes all entries. |
| Helm charts that create the resources above | All | Helm drops an empty list. To send `[]`, use `--set-json`, and do not put the value in an `{{ if }}` block. |
| `KeycloakRealmUser` | All | A `passwordSecret` with an empty `name` or `key` fails with a Secret error. 1.35.0 used `spec.password`. |
| `KeycloakClient` | All | `spec.realmRoles` is deprecated. Use `KeycloakRealmRole`. The admission webhook shows a warning for the field and does not block the request. |
| `KeycloakClient` | All | An aggregate policy that names a permission fails with `policy <name> does not exist`. Remove the permission from `aggregatedPolicy.policies`. |
| `KeycloakRealm`, `ClusterKeycloakRealm` | All | `smtpServer.port` has no default. The operator sends the port only when you set it. Keycloak then uses 465 with SSL and 25 without. A resource that the old CRD defaulted to 25 keeps the stored 25. |
| `KeycloakRealm`, `ClusterKeycloakRealm` | All | `displayName`, `displayNameHtml`, `organizationsEnabled` and `realmEventConfig.adminEventsExpiration` are not managed when you omit them. The operator keeps the Keycloak value. An explicit empty string, `false` or `0` is still written. A resource created under the old CRD has `organizationsEnabled: false` stored. It keeps managing the flag until you remove the field from the spec. |
| `KeycloakOrganization` | All | An update sends the `enabled` value that Keycloak returned. An organization disabled in Keycloak stays disabled after an operator write. 1.35.0 enabled it again. |

## Keycloak upgrades

### To Keycloak 26.8

Upgrade the operator to 1.36.0 first. 1.36.0 is the first operator version that supports Keycloak 26.8.

Keycloak 26.8 rejects some requests that 26.7 and earlier accepted. The status of the resource shows the
Keycloak message.

#### Before you upgrade

| Resource | Keycloak 26.8 rule | Action |
|---|---|---|
| `KeycloakRealmIdentityProvider` with a mapper that grants an admin role or an admin group | Returns 403 unless `spec.config.allowAdminRoleMapping` is `"true"`. | Set the key. The operator account needs `manage-realm` on the realm. Keep the key in the spec while the mapper exists. See [security.md](security.md). |
| `KeycloakRealmIdentityProvider` with changed `providerId` | Rejected on an existing identity provider. | Create a new identity provider. |
| `Keycloak`, `ClusterKeycloak` with `spec.tlsServerName` | Returns 421 when all of these are true: the TLS server name is not the `Host` of the request, the server has no proxy headers, and `Host` is not the Keycloak `hostname` or `hostname-admin`. | Set the Keycloak `hostname` option to the host of `spec.url`, or configure proxy headers on the server. |
