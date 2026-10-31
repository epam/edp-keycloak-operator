# Upgrade

## From 1.35.x to 1.36.x

Keycloak 26.4 to 26.8 (1.35: 26.4 to 26.7). Requires Kubernetes 1.32+ or OpenShift 4.19+; `helm upgrade` fails on older clusters.

Do the checks under "Before you upgrade" first.

Scale the operator to 0, then run these commands to update the CRDs before applying the upgrade.
CRDs are cluster-wide: run once per cluster.
Run the scale command for every operator release on the cluster. Default `<name>`: `keycloak-operator`.

```sh
kubectl scale deployment <name> -n <namespace> --replicas=0
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_clusterkeycloakrealms.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_clusterkeycloaks.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakauthflows.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakclients.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakclientscopes.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakorganizations.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakrealmcomponents.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakrealmgroups.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakrealmidentityproviders.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakrealmrolebatches.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakrealmroles.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakrealms.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloakrealmusers.yaml
kubectl replace -f https://raw.githubusercontent.com/epam/edp-keycloak-operator/v1.36.0/deploy-templates/crds/v1.edp.epam.com_keycloaks.yaml
```

A `replace` that reports `NotFound`: run `kubectl create -f` with the same URL. `helm upgrade` sets the replica count from the chart values.

Old CRDs prune new fields, for example `KeycloakOrganization` `spec.domainsV2`, `Keycloak` and `ClusterKeycloak` `spec.tlsServerName` and `status.value`, and `status.observedGeneration`.

Before you upgrade:

**`KeycloakClient` with `redirectUris: []` or `webOrigins: []`** ([#413](https://github.com/epam/edp-keycloak-operator/issues/413))

1.36.0 sends `[]` to Keycloak and clears the list. Login and CORS fail for the client. 1.35.0 read `[]` as a missing field. 1.35.0 can remove `[]` from the stored resource. Also check manifests in Git and `helm template` output.

```sh
kubectl get keycloakclients -A -o json | jq -r '.items[] | select(.spec.redirectUris == [] or .spec.webOrigins == []) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Remove the key to derive the list from `webUrl`, write the list to keep a fixed list, keep `[]` to clear it.

**`KeycloakRealmComponent` with `parentRef`** ([#423](https://github.com/epam/edp-keycloak-operator/issues/423))

Kind `KeycloakRealmComponent`: `parentRef.name` is the `metadata.name` of the parent CR in the same namespace. 1.33.0 to 1.35.0 used the Keycloak name of the parent component.
Kind `KeycloakRealm` (default): `parentRef.name` must equal `spec.realmRef.name`.
A child that does not resolve shows the error in `status.value` and is retried every 30 seconds.

```sh
kubectl get keycloakrealmcomponents -A -o json | jq -r '(.items | map("\(.metadata.namespace)/\(.metadata.name)")) as $crs | .items[] | select(.spec.parentRef != null) | select(if (.spec.parentRef.kind // "KeycloakRealm") == "KeycloakRealm" then .spec.parentRef.name != .spec.realmRef.name else ("\(.metadata.namespace)/\(.spec.parentRef.name)" | IN($crs[]) | not) end) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Set `parentRef.name` of each listed component to the parent CR name, or to `spec.realmRef.name`.

**`KeycloakAuthFlow` config values that start with `$`** ([#407](https://github.com/epam/edp-keycloak-operator/issues/407))

`authenticatorConfig.config` values that start with `$` are Secret references (`$secretName:secretKey`). A value such as `$abc` fails the reconcile and Keycloak keeps the old value. There is no escape syntax. 1.35.0 sent it as text.

```sh
kubectl get keycloakauthflows -A -o json | jq -r '.items[] | select([.spec.authenticationExecutions[]?.authenticatorConfig.config // {} | .[] | select(startswith("$") and (startswith("${") | not))] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Change the value of each listed flow to a Secret reference.
With `allowedDestinationHosts` set, the operator also checks destination values in `authenticatorConfig.config`: add the hosts of your auth flows to the list. See [security.md](security.md).

**`KeycloakRealm` and `ClusterKeycloakRealm` with duplicate list entries**

`localization.supportedLocales`, `realmEventConfig.enabledEventTypes` and `realmEventConfig.eventsListeners` are sets. The API server rejects a duplicate entry. Stored duplicates stay accepted.

```sh
for k in keycloakrealms clusterkeycloakrealms; do kubectl get $k -A -o json | jq -r '.items[] | select([.spec.localization.supportedLocales, .spec.realmEventConfig.enabledEventTypes, .spec.realmEventConfig.eventsListeners | select(. != null) | length != (unique | length)] | any) | "\(.kind) \(.metadata.namespace // "-")/\(.metadata.name)"'; done
```

Remove the duplicates from the listed resources and from their manifests in Git.

**`KeycloakOrganization` with `spec.domains`**

The operator copies `domains` to `domainsV2` once and reads only `domainsV2` afterwards. An entry with an empty name fails the migration.
Remove the empty entry. Move the list to `domainsV2` in the manifests in Git.

**Empty or blank `clientId` in `clientRoles`, empty or blank `compositesClientRoles` keys**

The API server rejects an empty `clientId` in `KeycloakRealmUser` and `KeycloakRealmGroup` `clientRoles` and in `KeycloakClient` `serviceAccount.clientRoles` and `scopeMappings.clientRoles`. It accepts a blank `clientId` (spaces only) and an empty or blank `compositesClientRoles` key. 1.36.0 fails the reconcile of such an entry or key.
1.33.0 to 1.35.0 used the first client of the realm (lowest `clientId`) for them.

```sh
for k in keycloakrealmusers keycloakrealmgroups; do kubectl get $k -A -o json | jq -r '.items[] | select([.spec.clientRoles[]? | select((.clientId // "") | test("^\\s*$"))] | length > 0) | "\(.kind) \(.metadata.namespace)/\(.metadata.name)"'; done
kubectl get keycloakclients -A -o json | jq -r '.items[] | select([.spec.serviceAccount.clientRoles[]?, .spec.scopeMappings.clientRoles[]? | select((.clientId // "") | test("^\\s*$"))] | length > 0) | "\(.kind) \(.metadata.namespace)/\(.metadata.name)"'
kubectl get keycloakrealmroles -A -o json | jq -r '.items[] | select(.spec.compositesClientRoles // {} | any(keys[]; test("^\\s*$"))) | "\(.kind) \(.metadata.namespace)/\(.metadata.name)"'
```

Set `clientId` in the listed resources, and set or remove the blank key in the listed roles, in the cluster and in Git. A later change to the list is rejected while the entry is empty.
For each listed user and service account, find the first client with `GET /admin/realms/<realm>/clients?first=0&max=1`. In Keycloak, remove the roles of that client that you did not assign, and assign again the roles that a `roles: []` entry removed. The operator does neither.
For a listed role, the operator removes the composites from that client after you fix the key. With `composite: false` instead, remove them by hand.

**`KeycloakRealmIdentityProvider` with `mappers[].identityProviderAlias`**

1.36.0 fails the reconcile before any Keycloak write when a mapper sets `identityProviderAlias` other than `spec.alias`. The field is deprecated. The identity provider is not created or updated while the field is set.
On Keycloak 26.4 to 26.7, 1.35.0 and earlier created such a mapper under the other alias `<X>`. It stays in Keycloak.

```sh
kubectl get keycloakrealmidentityproviders -A -o json | jq -r '.items[] | select(.spec.alias as $a | [.spec.mappers[]? | select((.identityProviderAlias // "") != "" and .identityProviderAlias != $a)] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"'
```

```sh
kubectl get keycloakrealmidentityproviders -A -o json | jq -r '.items[] | select([.spec.mappers[]? | select((.identityProviderAlias // "") != "")] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"'
```

First query: affected resources. Second query: every resource that sets the field.
Step 1, both lists: remove `identityProviderAlias` from each mapper in the resource and in Git, before step 2.
Step 2, first list, Keycloak 26.4 to 26.7: delete the orphan mapper under `<X>`.
If an identity provider with alias `<X>` exists, list its mappers with `GET /admin/realms/<realm>/identity-provider/instances/<X>/mappers`, then delete the one with the name from the resource with `DELETE /admin/realms/<realm>/identity-provider/instances/<X>/mappers/<id>`.
If none has alias `<X>`, create a disabled identity provider with alias `<X>`, list its mappers, delete the mapper, delete the temporary identity provider. Create it with `POST /admin/realms/<realm>/identity-provider/instances`:

```json
{"alias":"<X>","providerId":"keycloak-oidc","enabled":false,"config":{"clientId":"x","clientSecret":"y","authorizationUrl":"https://example.invalid/auth","tokenUrl":"https://example.invalid/token"}}
```

If you know the mapper ID, use `DELETE /admin/realms/<realm>/identity-provider/instances/<any-existing-alias>/mappers/<id>`.

**`KeycloakRealmIdentityProvider` mapper without `name` or `identityProviderMapper`**

The API server rejects a mapper with an empty or missing `name` or `identityProviderMapper` (mapper type). It accepts blank values (spaces only).

```sh
kubectl get keycloakrealmidentityproviders -A -o json | jq -r '.items[] | select([.spec.mappers[]? | select(((.name // "") | test("^\\s*$")) or ((.identityProviderMapper // "") | test("^\\s*$")))] | length > 0) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Set the name and the mapper type in the listed resources and in Git. A later change to `mappers` is rejected while a stored entry is empty or missing.

**`KeycloakClient` with duplicate authorization entries**

In `spec.authorization`, `policies`, `permissions`, `resources`, `rolePolicy.roles` and `groupPolicy.groups` are keyed by `name`. `scopes`, `clientPolicy.clients`, `userPolicy.users`, `aggregatedPolicy.policies` and the `policies`, `resources`, `scopes` of each permission are sets. The API server rejects a repeated name or entry.

```sh
kubectl get keycloakclients -A -o json | jq -r '.items[] | select([.spec.authorization // {} | (.scopes, ((.policies, .permissions, .resources) | select(. != null) | map(.name)), (.policies[]? | ((.rolePolicy.roles, .groupPolicy.groups) | select(. != null) | map(.name)), .clientPolicy.clients, .userPolicy.users, .aggregatedPolicy.policies), (.permissions[]? | .policies, .resources, .scopes)) | select(. != null) | length != (unique | length)] | any) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Remove the duplicates from the listed resources and from Git. Keep one entry per shared name.
With server-side apply (Flux, Argo CD `ServerSideApply=true`), fix Git before the CRDs: the API server rejects an apply of a manifest with duplicates with HTTP 500 `duplicate entries for key`, also when the manifest has not changed. A resource with stored duplicates accepts updates.

**`KeycloakRealmRole` and `KeycloakRealmRoleBatch`: realm default roles**

1.36.0 makes `default-roles-<realm>` match `isDefault` of each `KeycloakRealmRole`. Omitted `isDefault` is `false`: the role is removed from the default roles, for all users of the realm, on the first reconcile. A `KeycloakRealmRole` for `offline_access` without `isDefault: true` removes `offline_access` from all users.
`isDefault: true` works in realms with upper-case letters in the name and adds the role to the default roles.

1. List the roles without `isDefault: true`. Columns: Keycloak realm, role name, resource. The list includes the roles of a `KeycloakRealmRoleBatch`.

   ```sh
   { kubectl get keycloakrealms,clusterkeycloakrealms -A -o json; kubectl get keycloakrealmroles -A -o json; } | jq -rs '(.[0].items | map({key: "\(.kind)/\(.metadata.namespace // "")/\(.metadata.name)", value: .spec.realmName}) | from_entries) as $realms | .[1].items[] | select(.spec.isDefault != true) | (.spec.realmRef.kind // "KeycloakRealm") as $k | ($realms["\($k)/\(if $k == "KeycloakRealm" then .metadata.namespace else "" end)/\(.spec.realmRef.name)"] // "<realm not found>") as $realm | "\($realm) \(.spec.name) \(.metadata.namespace)/\(.metadata.name)"'
   ```

2. For each realm, `GET /admin/realms/<realm>` returns the default role name in `defaultRole.name`; `GET /admin/realms/<realm>/roles/<defaultRole.name>/composites/realm` returns the default roles.
3. A role in both lists is affected.

To keep an affected role a default role, set `isDefault: true` in the resource and in Git.
For a `KeycloakRealmRoleBatch` role, set it in the batch entry and in the child `KeycloakRealmRole`: an edit of a batch entry does not reach an existing child.
To remove the role from the default roles, do nothing.
To prevent a role with `isDefault: true` in a realm with upper-case letters from becoming a default role for all users, remove `isDefault`.

**`Keycloak` and `ClusterKeycloak` with a `spec.url` that answers with a redirect**

1.36.0 follows no HTTP redirect. A `spec.url` that answers with 3xx sets `status.connected` to `false` and `status.value` to `redirect refused: <method> <url> answered <code>; ...`; resources that use this Keycloak stop reconciling.

```sh
curl -sS -o /dev/null -w '%{http_code}\n' -X POST <spec.url>/realms/master/protocol/openid-connect/token
```

The command must print a status other than 3xx (`400` and `401` are fine). Otherwise set `spec.url` to the address that serves Keycloak without a redirect: remove a trailing slash, add `/auth` for a legacy base path.

After you upgrade:

**Realm roles that 1.35.0 removed** ([#371](https://github.com/epam/edp-keycloak-operator/issues/371), [#374](https://github.com/epam/edp-keycloak-operator/issues/374))

1.35.0 removed all realm roles, including `default-roles-<realm>`, from each `KeycloakRealmUser` with no `spec.roles` unless `reconciliationStrategy` was `addOnly`. 1.36.0 does not manage an omitted list: the roles stay removed.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.spec.roles == null and .spec.reconciliationStrategy != "addOnly") | "\(.metadata.namespace)/\(.metadata.name)"'
```

The query also lists users that 1.35.0 did not reconcile. Check the realm roles of each listed user in Keycloak. Give the roles back in Keycloak or add them to `spec.roles`. Same for `spec.groups` and `KeycloakRealmGroup` `spec.realmRoles`.
1.35.0 and earlier also removed all client roles from each `KeycloakRealmGroup` with no `spec.clientRoles`. The roles stay removed.

```sh
kubectl get keycloakrealmgroups -A -o json | jq -r '.items[] | select(.spec.clientRoles == null) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Check the client roles of each listed group in Keycloak. Give the roles back in Keycloak or add them to `spec.clientRoles`.

**`KeycloakRealmUser` with `attributesV2`** ([#418](https://github.com/epam/edp-keycloak-operator/issues/418))

Keycloak drops attributes that the realm user profile does not allow. 1.36.0 reads the user back and shows "keycloak did not store user attributes" in `status.value`. 1.35.0 showed `OK`.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.status.value // "" | contains("did not store user attributes")) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Set `spec.userProfileConfig.unmanagedAttributePolicy` of the realm to `ADMIN_EDIT` or `ENABLED`, or declare the attributes in `spec.userProfileConfig.attributes`.

**`KeycloakRealmUser` with an empty `passwordSecret`** ([#424](https://github.com/epam/edp-keycloak-operator/issues/424))

Before 1.31 the operator stored `passwordSecret: {name: "", key: ""}` in each user. 1.36.0 removes the entry once. Remove it from Git, or GitOps adds it again on each sync.

**`KeycloakRealmUser` without `spec.enabled`** ([#375](https://github.com/epam/edp-keycloak-operator/issues/375))

1.35.0 created and kept such a user disabled. 1.36.0 does not change `enabled` of an existing user: it stays disabled.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.spec.enabled == null) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Check each listed user in Keycloak. Set `enabled: true` in the spec where the user must be enabled.

**`KeycloakRealmUser` without `spec.emailVerified`**

1.35.0 wrote `emailVerified: false` on each update. 1.36.0 does not change `emailVerified` of an existing user: it stays unverified.

```sh
kubectl get keycloakrealmusers -A -o json | jq -r '.items[] | select(.spec.emailVerified == null) | "\(.metadata.namespace)/\(.metadata.name)"'
```

Check each listed user in Keycloak. Set `emailVerified: true` in the spec where the email must be verified.

Other changes:

- `KeycloakRealmUser`: `spec.enabled` and `spec.emailVerified` have no default and are not managed when omitted. Create: enabled, unverified. An existing user keeps the flag. Brute-force login failures and a temporary lockout stay. A change of `email` keeps the flag when `emailVerified` is omitted. Set `emailVerified: false` to make the user unverified.
- `KeycloakRealmUser`, `KeycloakRealmGroup`, `KeycloakClient`, `KeycloakOrganization`: an omitted list is not managed (user `roles`, `groups`; group `realmRoles`, `clientRoles` and their `roles`; client `serviceAccount.realmRoles`; organization `identityProviders`). `[]` removes all entries.
- Helm charts that create these resources: Helm drops an empty list. Use `--set-json` to send `[]`; keep the value out of an `{{ if }}` block.
- `KeycloakRealmUser`: a `passwordSecret` with an empty `name` or `key` fails with a Secret error. 1.35.0 used `spec.password`.
- `KeycloakClient`: `spec.realmRoles` is deprecated, use `KeycloakRealmRole`. The admission webhook warns.
- `KeycloakClient`: an aggregate policy that names a permission fails with `policy <name> does not exist`. Remove the permission from `aggregatedPolicy.policies`.
- `KeycloakRealm`, `ClusterKeycloakRealm`: `smtpServer.port` has no default. Keycloak uses 465 with SSL and 25 without. A resource that the old CRD defaulted to 25 keeps the stored 25.
- `KeycloakRealm`, `ClusterKeycloakRealm`: `displayName`, `displayNameHtml`, `organizationsEnabled` and `realmEventConfig.adminEventsExpiration` are not managed when omitted. An explicit empty string, `false` or `0` is written. `organizationsEnabled: false` stored under the old CRD stays managed until you remove the field.
- `KeycloakRealmIdentityProvider`: with `mappers: []`, 1.36.0 removes every mapper the list does not name; 1.35.0 skipped the removal.
- `KeycloakOrganization`: an update sends the `enabled` value that Keycloak returned. An organization disabled in Keycloak stays disabled. 1.35.0 enabled it again.

## From Keycloak 26.7 to 26.8

Upgrade the operator to 1.36 first.

- `KeycloakRealmIdentityProvider` with a mapper that grants an admin role or group: Keycloak returns 403 unless `spec.config.allowAdminRoleMapping` is `"true"`. Set the key and keep it while the mapper exists. The operator account needs `manage-realm`. See [security.md](security.md).
- `KeycloakRealmIdentityProvider` with changed `providerId`: Keycloak rejects it. Create a new identity provider.
- `Keycloak`, `ClusterKeycloak` with `spec.tlsServerName`: Keycloak returns 421 when the TLS server name is not the request `Host`, the server has no proxy headers and `Host` is not `hostname` or `hostname-admin`. Set the Keycloak `hostname` option to the host of `spec.url`, or configure proxy headers.
