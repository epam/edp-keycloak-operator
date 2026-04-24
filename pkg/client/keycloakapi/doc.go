// Package keycloakapi provides a Go client for the Keycloak Admin REST API.
//
// It wraps auto-generated code (from OpenAPI via oapi-codegen) with ergonomic,
// typed methods organized into domain-specific sub-clients.
//
// # Supported Keycloak Versions
//
// This client targets Keycloak 25+ (including Red Hat build of Keycloak).
// The generated client is produced from the official Keycloak OpenAPI specification.
//
// # Quick Start
//
//	ctx := context.Background()
//
//	kc, err := keycloakapi.NewKeycloakClient(ctx, "http://localhost:8080", "admin-cli",
//	    keycloakapi.WithPasswordGrant("admin", "admin"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List all realms
//	realms, _, err := kc.Realms.GetRealms(ctx)
//
//	// Create a user
//	_, err = kc.Users.CreateUser(ctx, "my-realm", keycloakapi.UserRepresentation{
//	    Username: ptr.To("newuser"),
//	    Enabled:  ptr.To(true),
//	})
//
//	// Get a client by its human-readable clientId
//	client, _, err := kc.Clients.GetClientByClientID(ctx, "my-realm", "my-app")
//
// # Client Architecture
//
// [KeycloakClient] is the main entry point. It holds sub-client fields for each
// Keycloak resource domain:
//
//   - Users       — user CRUD, credentials, roles, groups, federated identities, sessions
//   - Realms      — realm CRUD, authentication flows, event config, keys
//   - Groups      — group CRUD, hierarchy, role mappings, members, management permissions
//   - Roles       — realm role CRUD, composite roles
//   - Clients     — client CRUD, roles, scopes, protocol mappers, secrets, sessions, permissions
//   - Organizations — organization CRUD, identity provider linking, member management
//   - IdentityProviders — IDP CRUD, mappers, broker config export, management permissions
//   - Authorization — scopes, resources, policies, permissions (fine-grained authz)
//   - ClientScopes — client scope CRUD, realm default/optional scopes, protocol mappers
//   - RealmComponents — component CRUD (LDAP, key providers, etc.)
//   - AuthFlows   — authentication flow CRUD, executions, configs, required actions
//   - Server      — server info, feature flags
//   - Events      — user events, admin events, brute-force detection
//
// # Type Aliases
//
// Most representation types (e.g., [UserRepresentation], [ClientRepresentation]) are
// type aliases to the auto-generated types in the generated sub-package. This means
// they have all the fields from the Keycloak OpenAPI specification. See the Keycloak
// Admin REST API documentation for field definitions:
// https://www.keycloak.org/docs-api/latest/rest-api/index.html
//
// # Error Handling
//
// Methods return [ApiError] for non-2xx HTTP responses. Use helper functions:
//
//   - [IsNotFound] — true if the error is a 404 Not Found
//   - [IsConflict] — true if the error is a 409 Conflict
//   - [SkipConflict] — returns nil if the error is a conflict, the error otherwise
//   - [IsClientError] — true for any 4xx error
//   - [IsServerError] — true for any 5xx error
//
// Find* methods return [ErrNotFound] (a sentinel error) when the searched resource does
// not exist, distinct from [ApiError] with a 404 status code.
//
// # Response Type
//
// All methods return a [*Response] alongside the typed result and error. The Response
// is always populated when a Keycloak HTTP response was received (even on error), and
// contains the raw Body bytes and the underlying *http.Response. When the method returns
// a non-nil error and a non-nil Response, the Response provides the raw HTTP details.
// When the error is a transport or connection failure, Response will be nil.
//
// # Thread Safety
//
// [KeycloakClient] is safe for concurrent use from multiple goroutines. It uses internal
// synchronization for token refresh operations.
//
// # Context and Timeouts
//
// All methods accept a context.Context that is passed through to the underlying HTTP
// client. Context cancellation and deadlines are respected. The default HTTP timeout
// is 60 seconds and can be overridden with [WithClientTimeout]. When both a context
// deadline and the client timeout apply, the shorter one wins.
package keycloakapi
