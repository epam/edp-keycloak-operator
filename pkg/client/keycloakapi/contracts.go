package keycloakapi

import "context"

// UsersClient defines operations for managing Keycloak users including CRUD,
// credentials, role mappings, group membership, and federated identities.
type UsersClient interface {
	// GetUsers lists or searches users in the given realm.
	GetUsers(ctx context.Context, realm string, params *GetUsersParams) ([]UserRepresentation, *Response, error)
	// GetUser retrieves a single user by their Keycloak UUID.
	GetUser(ctx context.Context, realm, userID string) (*UserRepresentation, *Response, error)
	// GetUsersProfile returns the user profile configuration for a realm.
	GetUsersProfile(
		ctx context.Context,
		realm string,
	) (*UserProfileConfig, *Response, error)
	// UpdateUsersProfile updates the user profile configuration for a realm.
	UpdateUsersProfile(
		ctx context.Context,
		realm string,
		userProfile UserProfileConfig,
	) (*UserProfileConfig, *Response, error)
	// FindUserByUsername looks up a user by exact username. Returns ErrNotFound if no match.
	FindUserByUsername(ctx context.Context, realm, username string) (*UserRepresentation, *Response, error)
	// CreateUser creates a new user in the given realm.
	CreateUser(ctx context.Context, realm string, user UserRepresentation) (*Response, error)
	// UpdateUser updates an existing user identified by their Keycloak UUID.
	UpdateUser(ctx context.Context, realm, userID string, user UserRepresentation) (*Response, error)
	// DeleteUser deletes a user identified by their Keycloak UUID.
	DeleteUser(ctx context.Context, realm, userID string) (*Response, error)
	// SetUserPassword sets or resets the password for a user.
	SetUserPassword(ctx context.Context, realm, userID string, cred CredentialRepresentation) (*Response, error)
	// GetUserSessions returns active sessions for a user.
	GetUserSessions(ctx context.Context, realm, userID string) ([]UserSessionRepresentation, *Response, error)
	// LogoutUser terminates all sessions for a user.
	LogoutUser(ctx context.Context, realm, userID string) (*Response, error)
	// GetUserCredentials returns a list of credentials for a user.
	GetUserCredentials(ctx context.Context, realm, userID string) ([]CredentialRepresentation, *Response, error)
	// DeleteUserCredential removes a specific credential (e.g., reset TOTP) from a user.
	DeleteUserCredential(ctx context.Context, realm, userID, credentialID string) (*Response, error)
	// ExecuteActionsEmail triggers email actions (e.g., verify email, update password) for a user.
	ExecuteActionsEmail(ctx context.Context, realm, userID string, actions []string) (*Response, error)
	// SendVerifyEmail sends a verification email to a user.
	SendVerifyEmail(ctx context.Context, realm, userID string) (*Response, error)
	// ImpersonateUser initiates an impersonation session for the given user.
	ImpersonateUser(ctx context.Context, realm, userID string) (map[string]any, *Response, error)
	// GetUserRealmRoleMappings returns realm-level role mappings for a user.
	GetUserRealmRoleMappings(ctx context.Context, realm, userID string) ([]RoleRepresentation, *Response, error)
	// AddUserRealmRoles assigns realm-level roles to a user.
	AddUserRealmRoles(ctx context.Context, realm, userID string, roles []RoleRepresentation) (*Response, error)
	// DeleteUserRealmRoles removes realm-level roles from a user.
	DeleteUserRealmRoles(ctx context.Context, realm, userID string, roles []RoleRepresentation) (*Response, error)
	// GetUserClientRoleMappings returns client-level role mappings for a user; clientID is the Keycloak UUID.
	GetUserClientRoleMappings(ctx context.Context, realm, userID, clientID string) ([]RoleRepresentation, *Response, error)
	// AddUserClientRoles assigns client-level roles to a user; clientID is the Keycloak UUID.
	AddUserClientRoles(ctx context.Context, realm, userID, clientID string, roles []RoleRepresentation) (*Response, error)
	// DeleteUserClientRoles removes client-level roles from a user; clientID is the Keycloak UUID.
	DeleteUserClientRoles(
		ctx context.Context,
		realm, userID, clientID string,
		roles []RoleRepresentation,
	) (*Response, error)
	// GetUserGroups returns the groups a user belongs to.
	GetUserGroups(ctx context.Context, realm, userID string) ([]GroupRepresentation, *Response, error)
	// AddUserToGroup adds a user to a group.
	AddUserToGroup(ctx context.Context, realm, userID, groupID string) (*Response, error)
	// RemoveUserFromGroup removes a user from a group.
	RemoveUserFromGroup(ctx context.Context, realm, userID, groupID string) (*Response, error)
	// GetUserFederatedIdentities returns federated identity links for a user.
	GetUserFederatedIdentities(
		ctx context.Context,
		realm, userID string,
	) ([]FederatedIdentityRepresentation, *Response, error)
	// CreateUserFederatedIdentity links a federated identity provider to a user.
	CreateUserFederatedIdentity(
		ctx context.Context,
		realm, userID, provider string,
		identity FederatedIdentityRepresentation,
	) (*Response, error)
	// DeleteUserFederatedIdentity removes a federated identity link from a user.
	DeleteUserFederatedIdentity(ctx context.Context, realm, userID, provider string) (*Response, error)
}

// IdentityProvidersClient defines operations for managing Keycloak identity providers (IDPs),
// including CRUD, mapper management, and management permissions.
type IdentityProvidersClient interface {
	// GetIdentityProviders lists all identity providers in the given realm.
	GetIdentityProviders(ctx context.Context, realm string) ([]IdentityProviderRepresentation, *Response, error)
	// GetIdentityProvider retrieves an identity provider by its alias.
	GetIdentityProvider(ctx context.Context, realm, alias string) (*IdentityProviderRepresentation, *Response, error)
	// CreateIdentityProvider creates a new identity provider in the given realm.
	CreateIdentityProvider(ctx context.Context, realm string, idp IdentityProviderRepresentation) (*Response, error)
	// UpdateIdentityProvider updates an existing identity provider identified by alias.
	UpdateIdentityProvider(ctx context.Context, realm, alias string, idp IdentityProviderRepresentation) (*Response, error)
	// DeleteIdentityProvider deletes an identity provider by alias.
	DeleteIdentityProvider(ctx context.Context, realm, alias string) (*Response, error)
	// GetIDPMappers returns all mappers configured for an identity provider.
	GetIDPMappers(ctx context.Context, realm, alias string) ([]IdentityProviderMapperRepresentation, *Response, error)
	// CreateIDPMapper creates a new mapper for an identity provider.
	CreateIDPMapper(
		ctx context.Context, realm, alias string, mapper IdentityProviderMapperRepresentation,
	) (*Response, error)
	// UpdateIDPMapper updates an existing identity provider mapper.
	UpdateIDPMapper(
		ctx context.Context, realm, alias, mapperID string, mapper IdentityProviderMapperRepresentation,
	) (*Response, error)
	// DeleteIDPMapper deletes an identity provider mapper by ID.
	DeleteIDPMapper(ctx context.Context, realm, alias, mapperID string) (*Response, error)
	// ExportBrokerConfig exports the broker configuration (e.g., SAML metadata) for the given IDP alias.
	// The format parameter specifies the export format (e.g., "saml-idp-descriptor").
	ExportBrokerConfig(ctx context.Context, realm, alias, format string) ([]byte, *Response, error)
	// GetIDPManagementPermissions returns fine-grained management permissions for an identity provider.
	GetIDPManagementPermissions(
		ctx context.Context, realm, alias string,
	) (*ManagementPermissionReference, *Response, error)
	// UpdateIDPManagementPermissions enables or disables fine-grained management permissions for an IDP.
	UpdateIDPManagementPermissions(
		ctx context.Context, realm, alias string, permissions ManagementPermissionReference,
	) (*ManagementPermissionReference, *Response, error)
}

// RealmClient defines operations for managing Keycloak realms including CRUD,
// authentication flow settings, event configuration, and key management.
type RealmClient interface {
	// GetRealms lists all realms visible to the authenticated user.
	GetRealms(ctx context.Context) ([]RealmRepresentation, *Response, error)
	// GetRealm retrieves a realm by name.
	GetRealm(ctx context.Context, realm string) (*RealmRepresentation, *Response, error)
	// CreateRealm creates a new realm.
	CreateRealm(ctx context.Context, realmRep RealmRepresentation) (*Response, error)
	// UpdateRealm updates an existing realm.
	UpdateRealm(ctx context.Context, realm string, realmRep RealmRepresentation) (*Response, error)
	// DeleteRealm deletes a realm by name.
	DeleteRealm(ctx context.Context, realm string) (*Response, error)
	// SetRealmEventConfig updates the events configuration for a realm.
	SetRealmEventConfig(ctx context.Context, realm string, cfg RealmEventsConfigRepresentation) (*Response, error)
	// SetRealmBrowserFlow sets the browser authentication flow for a realm.
	SetRealmBrowserFlow(ctx context.Context, realm string, flowAlias string) (*Response, error)
	// GetAuthenticationFlows returns all authentication flows for a realm.
	GetAuthenticationFlows(ctx context.Context, realm string) ([]AuthenticationFlowRepresentation, *Response, error)
	// GetRealmKeys returns the key metadata for a realm (signing, encryption keys).
	GetRealmKeys(ctx context.Context, realm string) (*KeysMetadataRepresentation, *Response, error)
	// GetRealmLocalization returns localization strings for a realm and locale as a key-value map.
	GetRealmLocalization(ctx context.Context, realm, locale string) (map[string]string, *Response, error)
	// PostRealmLocalization sets or merges localization strings for a realm locale (POST /admin/realms/{realm}/localization/{locale}).
	// Keycloak ignores localizationTexts on realm update; runtime message bundles must use this API per locale.
	PostRealmLocalization(ctx context.Context, realm, locale string, texts map[string]string) (*Response, error)
}

// GroupsClient defines operations for managing Keycloak groups including CRUD,
// hierarchy, role mappings, and member management.
type GroupsClient interface {
	// GetGroups lists or searches groups in the given realm.
	GetGroups(ctx context.Context, realm string, params *GetGroupsParams) ([]GroupRepresentation, *Response, error)
	// GetGroup retrieves a single group by its Keycloak UUID.
	GetGroup(ctx context.Context, realm, groupID string) (*GroupRepresentation, *Response, error)
	// CreateGroup creates a new top-level group in the given realm.
	CreateGroup(ctx context.Context, realm string, group GroupRepresentation) (*Response, error)
	// UpdateGroup updates an existing group identified by its Keycloak UUID.
	UpdateGroup(ctx context.Context, realm, groupID string, group GroupRepresentation) (*Response, error)
	// DeleteGroup deletes a group by its Keycloak UUID.
	DeleteGroup(ctx context.Context, realm, groupID string) (*Response, error)
	// GetChildGroups returns child groups of the given parent group.
	GetChildGroups(
		ctx context.Context,
		realm string,
		groupID string,
		params *GetChildGroupsParams,
	) ([]GroupRepresentation, *Response, error)
	// CreateChildGroup creates a child group under the given parent group.
	CreateChildGroup(ctx context.Context, realm, parentGroupID string, group GroupRepresentation) (*Response, error)
	// FindGroupByName searches for a group by exact name. Returns ErrNotFound if no match.
	FindGroupByName(ctx context.Context, realm, groupName string) (*GroupRepresentation, *Response, error)
	// GetGroupByPath retrieves a group by its path (e.g., "/parent/child").
	GetGroupByPath(ctx context.Context, realm, path string) (*GroupRepresentation, *Response, error)
	// FindChildGroupByName searches for a child group by name under the given parent. Returns ErrNotFound if no match.
	FindChildGroupByName(
		ctx context.Context,
		realm string,
		parentGroupID string,
		groupName string,
	) (*GroupRepresentation, *Response, error)
	// GetGroupMembers lists users that are members of the given group.
	GetGroupMembers(
		ctx context.Context, realm, groupID string, params *GetGroupMembersParams,
	) ([]UserRepresentation, *Response, error)
	// GetGroupManagementPermissions returns fine-grained management permissions for a group.
	GetGroupManagementPermissions(
		ctx context.Context, realm, groupID string,
	) (*ManagementPermissionReference, *Response, error)
	// UpdateGroupManagementPermissions enables or disables fine-grained management permissions for a group.
	UpdateGroupManagementPermissions(
		ctx context.Context, realm, groupID string, permissions ManagementPermissionReference,
	) (*ManagementPermissionReference, *Response, error)
	// GetRoleMappings returns all role mappings (realm and client) for a group.
	GetRoleMappings(ctx context.Context, realm, groupID string) (*MappingsRepresentation, *Response, error)
	// GetRealmRoleMappings returns realm-level role mappings for a group.
	GetRealmRoleMappings(ctx context.Context, realm, groupID string) ([]RoleRepresentation, *Response, error)
	// AddRealmRoleMappings assigns realm-level roles to a group.
	AddRealmRoleMappings(ctx context.Context, realm, groupID string, roles []RoleRepresentation) (*Response, error)
	// DeleteRealmRoleMappings removes realm-level roles from a group.
	DeleteRealmRoleMappings(ctx context.Context, realm, groupID string, roles []RoleRepresentation) (*Response, error)
	// GetClientRoleMappings returns client-level role mappings for a group; clientID is the Keycloak UUID.
	GetClientRoleMappings(ctx context.Context, realm, groupID, clientID string) ([]RoleRepresentation, *Response, error)
	// AddClientRoleMappings assigns client-level roles to a group; clientID is the Keycloak UUID.
	AddClientRoleMappings(
		ctx context.Context,
		realm string,
		groupID string,
		clientID string,
		roles []RoleRepresentation,
	) (*Response, error)
	// DeleteClientRoleMappings removes client-level roles from a group; clientID is the Keycloak UUID.
	DeleteClientRoleMappings(
		ctx context.Context,
		realm string,
		groupID string,
		clientID string,
		roles []RoleRepresentation,
	) (*Response, error)
	// CountGroups returns the count of top-level groups in a realm.
	CountGroups(ctx context.Context, realm string, params *CountGroupsParams) (map[string]int64, *Response, error)
}

// RolesClient defines operations for managing Keycloak realm-level roles
// including CRUD and composite role management.
type RolesClient interface {
	// GetRealmRoles lists realm-level roles, optionally filtered by params.
	GetRealmRoles(ctx context.Context, realm string, params *GetRealmRolesParams) ([]RoleRepresentation, *Response, error)
	// GetRealmRole retrieves a realm-level role by name.
	GetRealmRole(ctx context.Context, realm, roleName string) (*RoleRepresentation, *Response, error)
	// CreateRealmRole creates a new realm-level role.
	CreateRealmRole(ctx context.Context, realm string, role RoleRepresentation) (*Response, error)
	// UpdateRealmRole updates an existing realm-level role by name.
	UpdateRealmRole(ctx context.Context, realm, roleName string, role RoleRepresentation) (*Response, error)
	// DeleteRealmRole deletes a realm-level role by name.
	DeleteRealmRole(ctx context.Context, realm, roleName string) (*Response, error)
	// GetRealmRoleComposites returns composite roles for a realm-level role.
	GetRealmRoleComposites(ctx context.Context, realm, roleName string) ([]RoleRepresentation, *Response, error)
	// AddRealmRoleComposites adds composite roles to a realm-level role.
	AddRealmRoleComposites(ctx context.Context, realm, roleName string, roles []RoleRepresentation) (*Response, error)
	// DeleteRealmRoleComposites removes composite roles from a realm-level role.
	DeleteRealmRoleComposites(ctx context.Context, realm, roleName string, roles []RoleRepresentation) (*Response, error)
}

// ClientsClient defines operations for managing Keycloak clients (OAuth/OIDC/SAML applications)
// including CRUD, roles, scopes, protocol mappers, service accounts, and permissions.
// Note: methods named clientID refer to the Keycloak UUID, while clientId refers to the
// human-readable client_id string. Use GetClientByClientID to look up by the human-readable identifier.
type ClientsClient interface {
	// GetClients lists clients in the given realm, optionally filtered by params.
	GetClients(ctx context.Context, realm string, params *GetClientsParams) ([]ClientRepresentation, *Response, error)
	// GetClient retrieves a client by its Keycloak UUID.
	GetClient(ctx context.Context, realm, clientUUID string) (*ClientRepresentation, *Response, error)
	// CreateClient creates a new client in the given realm.
	CreateClient(ctx context.Context, realm string, client ClientRepresentation) (*Response, error)
	// UpdateClient updates a client identified by its Keycloak UUID.
	UpdateClient(ctx context.Context, realm, clientUUID string, client ClientRepresentation) (*Response, error)
	// DeleteClient deletes a client identified by its Keycloak UUID.
	DeleteClient(ctx context.Context, realm, clientUUID string) (*Response, error)
	// GetClientByClientID looks up a client by its human-readable clientId string.
	GetClientByClientID(ctx context.Context, realm, clientID string) (*ClientRepresentation, *Response, error)
	// GetClientUUID returns the Keycloak UUID for the client with the given human-readable clientId.
	GetClientUUID(ctx context.Context, realm, clientID string) (string, error)
	// GetClientSecret retrieves the client secret for a confidential client.
	GetClientSecret(ctx context.Context, realm, clientUUID string) (*CredentialRepresentation, *Response, error)
	// RegenerateClientSecret rotates the client secret and returns the new credential.
	RegenerateClientSecret(ctx context.Context, realm, clientUUID string) (*CredentialRepresentation, *Response, error)
	// GetClientSessions returns active user sessions for a client.
	GetClientSessions(
		ctx context.Context, realm, clientUUID string, params *GetClientSessionsParams,
	) ([]UserSessionRepresentation, *Response, error)
	// GetClientRoles lists roles for a client; clientID is the Keycloak UUID.
	GetClientRoles(
		ctx context.Context,
		realm string,
		clientID string,
		params *GetClientRolesParams,
	) ([]RoleRepresentation, *Response, error)
	// GetClientRole retrieves a single client role by name; clientID is the Keycloak UUID.
	GetClientRole(ctx context.Context, realm, clientID, roleName string) (*RoleRepresentation, *Response, error)
	// CreateClientRole creates a new role for a client; clientID is the Keycloak UUID.
	CreateClientRole(ctx context.Context, realm, clientID string, role RoleRepresentation) (*Response, error)
	// UpdateClientRole updates a client role by name; clientID is the Keycloak UUID.
	UpdateClientRole(ctx context.Context, realm, clientID, roleName string, role RoleRepresentation) (*Response, error)
	// DeleteClientRole deletes a client role by name; clientID is the Keycloak UUID.
	DeleteClientRole(ctx context.Context, realm, clientID, roleName string) (*Response, error)
	// GetClientRoleComposites returns composite roles for a client role.
	GetClientRoleComposites(
		ctx context.Context, realm, clientUUID, roleName string,
	) ([]RoleRepresentation, *Response, error)
	// AddClientRoleComposites adds composite roles to a client role.
	AddClientRoleComposites(
		ctx context.Context, realm, clientUUID, roleName string, roles []RoleRepresentation,
	) (*Response, error)
	// DeleteClientRoleComposites removes composite roles from a client role.
	DeleteClientRoleComposites(
		ctx context.Context, realm, clientUUID, roleName string, roles []RoleRepresentation,
	) (*Response, error)
	// GetDefaultClientScopes returns the default client scopes assigned to a client.
	GetDefaultClientScopes(ctx context.Context, realm, clientUUID string) ([]ClientScopeRepresentation, *Response, error)
	// AddDefaultClientScope assigns a default client scope to a client.
	AddDefaultClientScope(ctx context.Context, realm, clientUUID, scopeID string) (*Response, error)
	// GetOptionalClientScopes returns the optional client scopes assigned to a client.
	GetOptionalClientScopes(ctx context.Context, realm, clientUUID string) ([]ClientScopeRepresentation, *Response, error)
	// AddOptionalClientScope assigns an optional client scope to a client.
	AddOptionalClientScope(ctx context.Context, realm, clientUUID, scopeID string) (*Response, error)
	// GetRealmClientScopes returns all client scopes available in the realm.
	GetRealmClientScopes(ctx context.Context, realm string) ([]ClientScopeRepresentation, *Response, error)
	// GetServiceAccountUser returns the service account user associated with a client.
	GetServiceAccountUser(ctx context.Context, realm, clientUUID string) (*UserRepresentation, *Response, error)
	// GetClientProtocolMappers returns all protocol mappers for a client.
	GetClientProtocolMappers(
		ctx context.Context, realm, clientUUID string,
	) ([]ProtocolMapperRepresentation, *Response, error)
	// CreateClientProtocolMapper creates a new protocol mapper for a client.
	CreateClientProtocolMapper(
		ctx context.Context, realm, clientUUID string, mapper ProtocolMapperRepresentation,
	) (*Response, error)
	// UpdateClientProtocolMapper updates an existing protocol mapper for a client.
	UpdateClientProtocolMapper(
		ctx context.Context, realm, clientUUID, mapperID string, mapper ProtocolMapperRepresentation,
	) (*Response, error)
	// DeleteClientProtocolMapper deletes a protocol mapper from a client.
	DeleteClientProtocolMapper(ctx context.Context, realm, clientUUID, mapperID string) (*Response, error)
	// GetClientManagementPermissions returns fine-grained management permissions for a client.
	GetClientManagementPermissions(
		ctx context.Context, realm, clientUUID string,
	) (*ManagementPermissionReference, *Response, error)
	// UpdateClientManagementPermissions enables or disables fine-grained management permissions for a client.
	UpdateClientManagementPermissions(
		ctx context.Context, realm, clientUUID string, permissions ManagementPermissionReference,
	) (*ManagementPermissionReference, *Response, error)
	// GetClientInstallationProvider returns the installation configuration for a client
	// using the specified provider (e.g., "keycloak-oidc-keycloak-json", "saml-idp-descriptor").
	GetClientInstallationProvider(
		ctx context.Context, realm, clientUUID, providerID string,
	) ([]byte, *Response, error)
}

// ClientScopesClient defines operations for managing Keycloak client scopes,
// realm default/optional scope assignments, and protocol mappers within scopes.
type ClientScopesClient interface {
	// GetClientScopes lists all client scopes in the given realm.
	GetClientScopes(ctx context.Context, realm string) ([]ClientScopeRepresentation, *Response, error)
	// GetClientScope retrieves a client scope by its Keycloak UUID.
	GetClientScope(ctx context.Context, realm, scopeID string) (*ClientScopeRepresentation, *Response, error)
	// CreateClientScope creates a new client scope in the given realm.
	CreateClientScope(ctx context.Context, realm string, scope ClientScopeRepresentation) (*Response, error)
	// UpdateClientScope updates an existing client scope.
	UpdateClientScope(ctx context.Context, realm, scopeID string, scope ClientScopeRepresentation) (*Response, error)
	// DeleteClientScope deletes a client scope by its Keycloak UUID.
	DeleteClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	// GetRealmDefaultClientScopes returns client scopes assigned as realm defaults.
	GetRealmDefaultClientScopes(ctx context.Context, realm string) ([]ClientScopeRepresentation, *Response, error)
	// AddRealmDefaultClientScope assigns a client scope as a realm default.
	AddRealmDefaultClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	// RemoveRealmDefaultClientScope removes a client scope from realm defaults.
	RemoveRealmDefaultClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	// GetRealmOptionalClientScopes returns client scopes assigned as realm optionals.
	GetRealmOptionalClientScopes(ctx context.Context, realm string) ([]ClientScopeRepresentation, *Response, error)
	// AddRealmOptionalClientScope assigns a client scope as a realm optional.
	AddRealmOptionalClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	// RemoveRealmOptionalClientScope removes a client scope from realm optionals.
	RemoveRealmOptionalClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	// GetClientScopeProtocolMappers returns all protocol mappers for a client scope.
	GetClientScopeProtocolMappers(
		ctx context.Context, realm, scopeID string,
	) ([]ProtocolMapperRepresentation, *Response, error)
	// CreateClientScopeProtocolMapper creates a new protocol mapper within a client scope.
	CreateClientScopeProtocolMapper(
		ctx context.Context, realm, scopeID string, mapper ProtocolMapperRepresentation,
	) (*Response, error)
	// UpdateClientScopeProtocolMapper updates an existing protocol mapper within a client scope.
	UpdateClientScopeProtocolMapper(
		ctx context.Context, realm, scopeID, mapperID string, mapper ProtocolMapperRepresentation,
	) (*Response, error)
	// DeleteClientScopeProtocolMapper deletes a protocol mapper from a client scope.
	DeleteClientScopeProtocolMapper(ctx context.Context, realm, scopeID, mapperID string) (*Response, error)
}

// AuthorizationClient defines operations for managing Keycloak fine-grained authorization:
// scopes, resources, policies, and permissions on a resource server (client).
type AuthorizationClient interface {
	// Scopes
	// GetScopes returns all authorization scopes for a resource server (client).
	GetScopes(ctx context.Context, realm, clientUUID string) ([]ScopeRepresentation, *Response, error)
	// GetScope retrieves a single authorization scope by ID.
	GetScope(ctx context.Context, realm, clientUUID, scopeID string) (*ScopeRepresentation, *Response, error)
	// CreateScope creates a new authorization scope.
	CreateScope(ctx context.Context, realm, clientUUID string, scope ScopeRepresentation) (*Response, error)
	// UpdateScope updates an existing authorization scope.
	UpdateScope(ctx context.Context, realm, clientUUID, scopeID string, scope ScopeRepresentation) (*Response, error)
	// DeleteScope deletes an authorization scope by ID.
	DeleteScope(ctx context.Context, realm, clientUUID, scopeID string) (*Response, error)
	// Resources
	// GetResources returns all resources for a resource server (client).
	GetResources(ctx context.Context, realm, clientUUID string) ([]ResourceRepresentation, *Response, error)
	// GetResource retrieves a single resource by ID.
	GetResource(
		ctx context.Context, realm, clientUUID, resourceID string,
	) (*ResourceRepresentation, *Response, error)
	// CreateResource creates a new authorization resource.
	CreateResource(
		ctx context.Context, realm, clientUUID string, resource ResourceRepresentation,
	) (*ResourceRepresentation, *Response, error)
	// UpdateResource updates an existing authorization resource.
	UpdateResource(
		ctx context.Context, realm, clientUUID, resourceID string, resource ResourceRepresentation,
	) (*Response, error)
	// DeleteResource deletes an authorization resource by ID.
	DeleteResource(ctx context.Context, realm, clientUUID, resourceID string) (*Response, error)
	// Policies
	// GetPolicies returns all authorization policies for a resource server.
	GetPolicies(ctx context.Context, realm, clientUUID string) ([]AbstractPolicyRepresentation, *Response, error)
	// CreatePolicy creates a new authorization policy of the given type.
	CreatePolicy(
		ctx context.Context, realm, clientUUID, policyType string, policy any,
	) (*PolicyRepresentation, *Response, error)
	// UpdatePolicy updates an existing authorization policy.
	UpdatePolicy(
		ctx context.Context, realm, clientUUID, policyType, policyID string, policy any,
	) (*Response, error)
	// GetPolicy retrieves an authorization policy by type and ID.
	GetPolicy(ctx context.Context, realm, clientUUID, policyType, policyID string) (*Response, error)
	// DeletePolicy deletes an authorization policy by ID.
	DeletePolicy(ctx context.Context, realm, clientUUID, policyID string) (*Response, error)
	// Permissions
	// GetPermissions returns all authorization permissions for a resource server.
	GetPermissions(ctx context.Context, realm, clientUUID string) ([]AbstractPolicyRepresentation, *Response, error)
	// CreatePermission creates a new authorization permission of the given type.
	CreatePermission(
		ctx context.Context, realm, clientUUID, permType string, perm PolicyRepresentation,
	) (*PolicyRepresentation, *Response, error)
	// UpdatePermission updates an existing authorization permission.
	UpdatePermission(
		ctx context.Context, realm, clientUUID, permType, permID string, perm PolicyRepresentation,
	) (*Response, error)
	// DeletePermission deletes an authorization permission by ID.
	DeletePermission(ctx context.Context, realm, clientUUID, permID string) (*Response, error)
}

// ServerInfoClient provides access to Keycloak server metadata and feature flags.
type ServerInfoClient interface {
	// GetServerInfo returns Keycloak server metadata including version and provider info.
	GetServerInfo(ctx context.Context) (*ServerInfo, error)
	// FeatureFlagEnabled checks whether a named feature flag is enabled on the Keycloak server.
	FeatureFlagEnabled(ctx context.Context, featureFlag string) (bool, error)
}

// RealmComponentsClient defines operations for managing Keycloak realm components
// (e.g., LDAP user federation, key providers).
type RealmComponentsClient interface {
	// GetComponents returns realm components, optionally filtered by params (e.g., type, parent).
	GetComponents(
		ctx context.Context, realm string, params *GetComponentsParams,
	) ([]ComponentRepresentation, *Response, error)
	// GetComponent retrieves a single realm component by its Keycloak UUID.
	GetComponent(ctx context.Context, realm, componentID string) (*ComponentRepresentation, *Response, error)
	// FindComponentByName searches for a realm component by name. Returns ErrNotFound if no match.
	FindComponentByName(ctx context.Context, realm, componentName string) (*ComponentRepresentation, error)
	// CreateComponent creates a new realm component.
	CreateComponent(ctx context.Context, realm string, component ComponentRepresentation) (*Response, error)
	// UpdateComponent updates an existing realm component.
	UpdateComponent(ctx context.Context, realm, componentID string, component ComponentRepresentation) (*Response, error)
	// DeleteComponent deletes a realm component by its Keycloak UUID.
	DeleteComponent(ctx context.Context, realm, componentID string) (*Response, error)
}

// OrganizationsClient defines operations for managing Keycloak organizations
// including CRUD, identity provider linking, and member management.
type OrganizationsClient interface {
	// GetOrganizations lists organizations in a realm, optionally filtered by params.
	GetOrganizations(
		ctx context.Context,
		realm string,
		params *GetOrganizationsParams,
	) ([]OrganizationRepresentation, *Response, error)
	// GetOrganizationByAlias looks up an organization by its alias. Returns ErrNotFound if no match.
	GetOrganizationByAlias(ctx context.Context, realm, alias string) (*OrganizationRepresentation, *Response, error)
	// CreateOrganization creates a new organization in the given realm.
	CreateOrganization(ctx context.Context, realm string, org OrganizationRepresentation) (*Response, error)
	// UpdateOrganization updates an existing organization.
	UpdateOrganization(ctx context.Context, realm, orgID string, org OrganizationRepresentation) (*Response, error)
	// DeleteOrganization deletes an organization by its Keycloak UUID.
	DeleteOrganization(ctx context.Context, realm, orgID string) (*Response, error)
	// GetOrganizationIdentityProviders returns identity providers linked to an organization.
	GetOrganizationIdentityProviders(
		ctx context.Context,
		realm, orgID string,
	) ([]IdentityProviderRepresentation, *Response, error)
	// LinkIdentityProviderToOrganization links an identity provider to an organization by alias.
	LinkIdentityProviderToOrganization(ctx context.Context, realm, orgID, alias string) (*Response, error)
	// UnlinkIdentityProviderFromOrganization unlinks an identity provider from an organization.
	UnlinkIdentityProviderFromOrganization(ctx context.Context, realm, orgID, alias string) (*Response, error)
	// GetOrganizationMembers lists members of an organization.
	GetOrganizationMembers(
		ctx context.Context, realm, orgID string, params *GetOrganizationMembersParams,
	) ([]MemberRepresentation, *Response, error)
	// AddOrganizationMember adds a user (by user ID) as a member of an organization.
	AddOrganizationMember(ctx context.Context, realm, orgID, userID string) (*Response, error)
	// RemoveOrganizationMember removes a member from an organization.
	RemoveOrganizationMember(ctx context.Context, realm, orgID, memberID string) (*Response, error)
	// InviteExistingOrganizationMember sends an invitation to an existing Keycloak user to join an organization.
	InviteExistingOrganizationMember(ctx context.Context, realm, orgID, userID string) (*Response, error)
	// InviteNewOrganizationMember sends an invitation to a new user (by email) to join an organization.
	InviteNewOrganizationMember(ctx context.Context, realm, orgID, email, firstName, lastName string) (*Response, error)
}
