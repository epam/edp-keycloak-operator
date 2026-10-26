# Security considerations

## Trust boundary: Secret references

The operator reads Kubernetes Secrets with its own ServiceAccount: a namespaced Role by default, a
ClusterRole when `clusterReconciliationEnabled` is true. A custom resource that names a Secret makes the
operator read it on the author's behalf.

- A `$secretName:secretKey` reference resolves only from the namespace of the custom resource.
- Whoever may create or update a resource that resolves references can send any Secret of that
  namespace to an address the resource names.

Grant create and update on the kinds below only to principals that may already read Secrets in the
namespace. When a GitOps tool applies these resources, every committer to its source repository is an
author.

## Kinds by Secret use

| Use | Kinds | Where the value goes |
|---|---|---|
| No Secret read | KeycloakRealmRole, KeycloakRealmRoleBatch, KeycloakRealmGroup, KeycloakClientScope, KeycloakOrganization; any config without a reference | nowhere |
| Secret stored in Keycloak only | KeycloakClient (`spec.secret`), KeycloakRealmUser (password Secret) | Keycloak admin API; Keycloak does not send it on |
| Secret sent to an address in the spec | Keycloak, ClusterKeycloak (admin credentials → `spec.url`, routed by `spec.tlsServerName`); KeycloakRealm, ClusterKeycloakRealm (SMTP password → `spec.smtp.connection.host`); KeycloakRealmIdentityProvider, KeycloakRealmComponent, KeycloakAuthFlow (references in `config` → addresses in the same config) | the address the author writes |

The chart ships only viewer helper roles for KeycloakRealmIdentityProvider, KeycloakRealmComponent and
KeycloakAuthFlow; write access to them is an explicit grant.

## Destination allowlist

`allowedDestinationHosts` and `enforceDestinationAllowlist` (Helm values) restrict the addresses a
resolved Secret may reach. The list is empty by default: no policy, nothing logged.

- Checked: Keycloak `spec.url` and `spec.tlsServerName`, realm `spec.smtp.connection.host`, and the known address keys
  (`tokenUrl`, `connectionUrl`, ...) in identity provider, realm component and authenticator config.
- Not checked: other config keys, including every key of a custom provider. A reference in a credential key next
  to an address in such a key passes.
- With a non-empty list and `enforceDestinationAllowlist: false`, violations are logged and counted in
  `keycloak_operator_destination_violations_total`; set the list first, read the log, then enforce.

The allowlist narrows the trust boundary above; it does not replace RBAC.

## Keycloak egress

Keycloak itself dials the addresses in identity provider, component and authenticator config. Restrict
Keycloak pod egress with a NetworkPolicy; the operator does not control it.

## Identity provider admin role mappers

Keycloak 26.8 returns 403 on a mapper create or update that grants an admin role or an admin group, unless the
identity provider config has `allowAdminRoleMapping` set to `true`.

- Set `spec.config.allowAdminRoleMapping: "true"` on the `KeycloakRealmIdentityProvider` that carries such a mapper.
- Setting or changing `allowAdminRoleMapping` requires the `manage-realm` role on the realm. Other identity provider
  and mapper writes require `manage-identity-providers`.
- The operator sends `spec.config` as the whole identity provider config. Omitting the key while Keycloak stores
  `true` counts as a change: without `manage-realm` every identity provider update returns 403; with it, the flag is
  disabled and the next mapper write returns 403. Keep the key in the spec while the mapper exists.
- A mapper's `identityProviderAlias` is deprecated. A value other than `spec.alias` fails the reconcile before any
  Keycloak write, so a mapper can no longer be pointed at another identity provider alias through
  `identityProviderAlias`. Omit the field.
- Keycloak 26.8 rejects a change of `providerId` on an existing identity provider.
