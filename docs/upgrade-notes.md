# Upgrade notes

Supported Keycloak versions: 26.4 to 26.8.

## Keycloak 26.8

Keycloak 26.8 rejects some requests that 26.7 and earlier accepted. Existing resources can start to fail
after the Keycloak upgrade. The operator reports the Keycloak message in the resource status.

Check before you upgrade Keycloak:

| Resource | Keycloak 26.8 rule | Action |
|---|---|---|
| `KeycloakRealmIdentityProvider` with a mapper that grants an admin role or an admin group | Returns 403 unless `spec.config.allowAdminRoleMapping` is `"true"`. | Set the key. The operator account needs `manage-realm` on the realm. Keep the key in the spec while the mapper exists. |
| `KeycloakRealmIdentityProvider` mapper with `identityProviderAlias` | Returns 400 when the alias differs from the alias of the identity provider. | Remove the field or set it to the identity provider alias. |
| `KeycloakRealmIdentityProvider` with changed `providerId` | Rejected on an existing identity provider. | Create a new identity provider. |
| `Keycloak`, `ClusterKeycloak` with `spec.tlsServerName` | Returns 421 when the TLS server name differs from the `Host` of the request, the server has no proxy headers configured, and `Host` is not the Keycloak `hostname` or `hostname-admin`. | Set the Keycloak `hostname` option to the host of `spec.url`, or configure proxy headers on the server. |

See [security.md](security.md) for details on `allowAdminRoleMapping`.

## Check before you upgrade the operator

| Resource | Change | Action |
|---|---|---|
| `KeycloakOrganization` with `spec.domains` | The operator copies `domains` into `domainsV2` once and then reads only `domainsV2`. Later edits of `domains` are ignored. | Move the list from `domains` to `domainsV2` in the Git source of the resource (GitOps) at upgrade time. |
| `KeycloakOrganization` CRD | The operator needs the updated `KeycloakOrganization` CRD (new `domainsV2`, new identity provider link fields). Helm does not upgrade CRDs. With the old CRD the API server drops `domainsV2`, and the reconcile of every `KeycloakOrganization` that has `spec.domains` fails with an error that asks for the CRD upgrade. | Apply `deploy-templates/crds/` (or the CRDs of the chart version) before or together with the operator update. |

## Operator behavior changes

| Resource | Change |
|---|---|
| `KeycloakOrganization` | An update keeps the Keycloak-side settings of domains that the spec still lists: `verified`, and from 26.8 `identityProviderAlias` and `autoRedirect`. A spec edit no longer resets them. |
| `KeycloakOrganization` | Domain names are compared trimmed and in lower case. A spec in mixed case no longer writes to Keycloak on every reconcile. |
| `KeycloakOrganization` | From 26.8 an identity provider can belong to several organizations. |
| `KeycloakOrganization` | `identityProviders[].autoMembership` and `membershipType` (Keycloak 26.8 or later) are applied to the organization link and repaired on drift. Fields left out keep the Keycloak value, and a change sends both fields. |
| `KeycloakOrganization` | `domains` is deprecated in favor of `domainsV2`. The operator copies `domains` into `domainsV2` once, which updates the resource (one generation bump), and then keeps every Keycloak-side domain setting. |
| `KeycloakOrganization` | `domainsV2[].verified`, `identityProviderAlias` and `autoRedirect` (the last two need Keycloak 26.8 or later) are written after the identity provider links and repaired on drift. Settings left out keep the Keycloak value. An empty `identityProviderAlias` clears the routing. |
| `KeycloakOrganization` | A `domainsV2[].identityProviderAlias` that is not in `identityProviders`, or the same domain listed twice with different settings, fails the reconcile before any Keycloak write. |
| `KeycloakOrganization` | An organization update sends the `enabled` value Keycloak returned. An organization disabled in Keycloak stays disabled after an operator write. Earlier versions enabled it again. |
| `KeycloakOrganization` | Linking an identity provider that does not exist in the realm fails before the link request, and the error names `KeycloakRealmIdentityProvider`. The operator checks every new alias before it unlinks any identity provider. Each newly linked alias costs one extra GET request; a reconcile without new links sends none. |
| `KeycloakOrganization` on Keycloak 26.4 to 26.7 | `identityProviders[].autoMembership` and `membershipType` fail the reconcile with an error that names the 26.8 requirement, and the operator sends no link update for that identity provider. The plain link request still happens, so the identity provider is linked without the settings. Keycloak 26.7 rejects `domainsV2[].identityProviderAlias` and `autoRedirect` with HTTP 400 (unknown field); the operator shows the Keycloak message in the status. Leave these fields out until Keycloak is 26.8 or later. `domainsV2[].verified` works on all supported versions. |
| `KeycloakOrganization` | Migration limit: the migration update drops an explicit `identityProviders: []` and `attributes: {}`, with the same meaning, and also drops empty strings from the stored spec (`description: ""`, `redirectUrl: ""`). A legacy `domains` list with an empty name fails the migration; the error appears in the operator log. Remove the empty entry from `domains`. |
