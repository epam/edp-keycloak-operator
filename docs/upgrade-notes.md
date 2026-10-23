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

## Operator behavior changes

| Resource | Change |
|---|---|
| `KeycloakOrganization` | An update keeps the Keycloak-side settings of domains that the spec still lists: `verified`, and from 26.8 `identityProviderAlias` and `autoRedirect`. A spec edit no longer resets them. |
| `KeycloakOrganization` | Domain names are compared trimmed and in lower case. A spec in mixed case no longer writes to Keycloak on every reconcile. |
| `KeycloakOrganization` | From 26.8 an identity provider can belong to several organizations. |
