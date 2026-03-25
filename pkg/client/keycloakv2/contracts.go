package keycloakv2

import "context"

type UsersClient interface {
	GetUsersProfile(
		ctx context.Context,
		realm string,
	) (*UserProfileConfig, *Response, error)
	UpdateUsersProfile(
		ctx context.Context,
		realm string,
		userProfile UserProfileConfig,
	) (*UserProfileConfig, *Response, error)
	FindUserByUsername(ctx context.Context, realm, username string) (*UserRepresentation, *Response, error)
	CreateUser(ctx context.Context, realm string, user UserRepresentation) (*Response, error)
	UpdateUser(ctx context.Context, realm, userID string, user UserRepresentation) (*Response, error)
	DeleteUser(ctx context.Context, realm, userID string) (*Response, error)
	SetUserPassword(ctx context.Context, realm, userID string, cred CredentialRepresentation) (*Response, error)
	GetUserRealmRoleMappings(ctx context.Context, realm, userID string) ([]RoleRepresentation, *Response, error)
	AddUserRealmRoles(ctx context.Context, realm, userID string, roles []RoleRepresentation) (*Response, error)
	DeleteUserRealmRoles(ctx context.Context, realm, userID string, roles []RoleRepresentation) (*Response, error)
	GetUserClientRoleMappings(ctx context.Context, realm, userID, clientID string) ([]RoleRepresentation, *Response, error)
	AddUserClientRoles(ctx context.Context, realm, userID, clientID string, roles []RoleRepresentation) (*Response, error)
	DeleteUserClientRoles(
		ctx context.Context,
		realm, userID, clientID string,
		roles []RoleRepresentation,
	) (*Response, error)
	GetUserGroups(ctx context.Context, realm, userID string) ([]GroupRepresentation, *Response, error)
	AddUserToGroup(ctx context.Context, realm, userID, groupID string) (*Response, error)
	RemoveUserFromGroup(ctx context.Context, realm, userID, groupID string) (*Response, error)
	GetUserFederatedIdentities(
		ctx context.Context,
		realm, userID string,
	) ([]FederatedIdentityRepresentation, *Response, error)
	CreateUserFederatedIdentity(
		ctx context.Context,
		realm, userID, provider string,
		identity FederatedIdentityRepresentation,
	) (*Response, error)
	DeleteUserFederatedIdentity(ctx context.Context, realm, userID, provider string) (*Response, error)
}

type IdentityProvidersClient interface {
	GetIdentityProvider(ctx context.Context, realm, alias string) (*IdentityProviderRepresentation, *Response, error)
	CreateIdentityProvider(ctx context.Context, realm string, idp IdentityProviderRepresentation) (*Response, error)
	DeleteIdentityProvider(ctx context.Context, realm, alias string) (*Response, error)
}

type RealmClient interface {
	GetRealm(ctx context.Context, realm string) (*RealmRepresentation, *Response, error)
	CreateRealm(ctx context.Context, realmRep RealmRepresentation) (*Response, error)
	UpdateRealm(ctx context.Context, realm string, realmRep RealmRepresentation) (*Response, error)
	DeleteRealm(ctx context.Context, realm string) (*Response, error)
	SetRealmEventConfig(ctx context.Context, realm string, cfg RealmEventsConfigRepresentation) (*Response, error)
	SetRealmBrowserFlow(ctx context.Context, realm string, flowAlias string) (*Response, error)
	GetAuthenticationFlows(ctx context.Context, realm string) ([]AuthenticationFlowRepresentation, *Response, error)
}

type GroupsClient interface {
	GetGroups(ctx context.Context, realm string, params *GetGroupsParams) ([]GroupRepresentation, *Response, error)
	GetGroup(ctx context.Context, realm, groupID string) (*GroupRepresentation, *Response, error)
	CreateGroup(ctx context.Context, realm string, group GroupRepresentation) (*Response, error)
	UpdateGroup(ctx context.Context, realm, groupID string, group GroupRepresentation) (*Response, error)
	DeleteGroup(ctx context.Context, realm, groupID string) (*Response, error)
	GetChildGroups(
		ctx context.Context,
		realm string,
		groupID string,
		params *GetChildGroupsParams,
	) ([]GroupRepresentation, *Response, error)
	CreateChildGroup(ctx context.Context, realm, parentGroupID string, group GroupRepresentation) (*Response, error)
	FindGroupByName(ctx context.Context, realm, groupName string) (*GroupRepresentation, *Response, error)
	GetGroupByPath(ctx context.Context, realm, path string) (*GroupRepresentation, *Response, error)
	FindChildGroupByName(
		ctx context.Context,
		realm string,
		parentGroupID string,
		groupName string,
	) (*GroupRepresentation, *Response, error)
	GetRoleMappings(ctx context.Context, realm, groupID string) (*MappingsRepresentation, *Response, error)
	GetRealmRoleMappings(ctx context.Context, realm, groupID string) ([]RoleRepresentation, *Response, error)
	AddRealmRoleMappings(ctx context.Context, realm, groupID string, roles []RoleRepresentation) (*Response, error)
	DeleteRealmRoleMappings(ctx context.Context, realm, groupID string, roles []RoleRepresentation) (*Response, error)
	GetClientRoleMappings(ctx context.Context, realm, groupID, clientID string) ([]RoleRepresentation, *Response, error)
	AddClientRoleMappings(
		ctx context.Context,
		realm string,
		groupID string,
		clientID string,
		roles []RoleRepresentation,
	) (*Response, error)
	DeleteClientRoleMappings(
		ctx context.Context,
		realm string,
		groupID string,
		clientID string,
		roles []RoleRepresentation,
	) (*Response, error)
}

type RolesClient interface {
	GetRealmRoles(ctx context.Context, realm string, params *GetRealmRolesParams) ([]RoleRepresentation, *Response, error)
	GetRealmRole(ctx context.Context, realm, roleName string) (*RoleRepresentation, *Response, error)
	CreateRealmRole(ctx context.Context, realm string, role RoleRepresentation) (*Response, error)
	UpdateRealmRole(ctx context.Context, realm, roleName string, role RoleRepresentation) (*Response, error)
	DeleteRealmRole(ctx context.Context, realm, roleName string) (*Response, error)
	GetRealmRoleComposites(ctx context.Context, realm, roleName string) ([]RoleRepresentation, *Response, error)
	AddRealmRoleComposites(ctx context.Context, realm, roleName string, roles []RoleRepresentation) (*Response, error)
	DeleteRealmRoleComposites(ctx context.Context, realm, roleName string, roles []RoleRepresentation) (*Response, error)
}

type ClientsClient interface {
	GetClients(ctx context.Context, realm string, params *GetClientsParams) ([]ClientRepresentation, *Response, error)
	GetClient(ctx context.Context, realm, clientID string) (*ClientRepresentation, *Response, error)
	CreateClient(ctx context.Context, realm string, client ClientRepresentation) (*Response, error)
	UpdateClient(ctx context.Context, realm, clientUUID string, client ClientRepresentation) (*Response, error)
	DeleteClient(ctx context.Context, realm, clientID string) (*Response, error)
	GetClientByClientID(ctx context.Context, realm, clientID string) (*ClientRepresentation, *Response, error)
	GetClientUUID(ctx context.Context, realm, clientID string) (string, error)
	GetClientRoles(
		ctx context.Context,
		realm string,
		clientID string,
		params *GetClientRolesParams,
	) ([]RoleRepresentation, *Response, error)
	GetClientRole(ctx context.Context, realm, clientID, roleName string) (*RoleRepresentation, *Response, error)
	CreateClientRole(ctx context.Context, realm, clientID string, role RoleRepresentation) (*Response, error)
	UpdateClientRole(ctx context.Context, realm, clientID, roleName string, role RoleRepresentation) (*Response, error)
	DeleteClientRole(ctx context.Context, realm, clientID, roleName string) (*Response, error)
	GetClientRoleComposites(
		ctx context.Context, realm, clientUUID, roleName string,
	) ([]RoleRepresentation, *Response, error)
	AddClientRoleComposites(
		ctx context.Context, realm, clientUUID, roleName string, roles []RoleRepresentation,
	) (*Response, error)
	DeleteClientRoleComposites(
		ctx context.Context, realm, clientUUID, roleName string, roles []RoleRepresentation,
	) (*Response, error)
	GetDefaultClientScopes(ctx context.Context, realm, clientUUID string) ([]ClientScopeRepresentation, *Response, error)
	AddDefaultClientScope(ctx context.Context, realm, clientUUID, scopeID string) (*Response, error)
	GetOptionalClientScopes(ctx context.Context, realm, clientUUID string) ([]ClientScopeRepresentation, *Response, error)
	AddOptionalClientScope(ctx context.Context, realm, clientUUID, scopeID string) (*Response, error)
	GetRealmClientScopes(ctx context.Context, realm string) ([]ClientScopeRepresentation, *Response, error)
	GetServiceAccountUser(ctx context.Context, realm, clientUUID string) (*UserRepresentation, *Response, error)
	GetClientProtocolMappers(
		ctx context.Context, realm, clientUUID string,
	) ([]ProtocolMapperRepresentation, *Response, error)
	CreateClientProtocolMapper(
		ctx context.Context, realm, clientUUID string, mapper ProtocolMapperRepresentation,
	) (*Response, error)
	UpdateClientProtocolMapper(
		ctx context.Context, realm, clientUUID, mapperID string, mapper ProtocolMapperRepresentation,
	) (*Response, error)
	DeleteClientProtocolMapper(ctx context.Context, realm, clientUUID, mapperID string) (*Response, error)
	GetClientManagementPermissions(
		ctx context.Context, realm, clientUUID string,
	) (*ManagementPermissionReference, *Response, error)
	UpdateClientManagementPermissions(
		ctx context.Context, realm, clientUUID string, permissions ManagementPermissionReference,
	) (*ManagementPermissionReference, *Response, error)
}

type ClientScopesClient interface {
	GetClientScopes(ctx context.Context, realm string) ([]ClientScopeRepresentation, *Response, error)
	GetClientScope(ctx context.Context, realm, scopeID string) (*ClientScopeRepresentation, *Response, error)
	CreateClientScope(ctx context.Context, realm string, scope ClientScopeRepresentation) (*Response, error)
	UpdateClientScope(ctx context.Context, realm, scopeID string, scope ClientScopeRepresentation) (*Response, error)
	DeleteClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	GetRealmDefaultClientScopes(ctx context.Context, realm string) ([]ClientScopeRepresentation, *Response, error)
	AddRealmDefaultClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	RemoveRealmDefaultClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	GetRealmOptionalClientScopes(ctx context.Context, realm string) ([]ClientScopeRepresentation, *Response, error)
	AddRealmOptionalClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	RemoveRealmOptionalClientScope(ctx context.Context, realm, scopeID string) (*Response, error)
	GetClientScopeProtocolMappers(
		ctx context.Context, realm, scopeID string,
	) ([]ProtocolMapperRepresentation, *Response, error)
	CreateClientScopeProtocolMapper(
		ctx context.Context, realm, scopeID string, mapper ProtocolMapperRepresentation,
	) (*Response, error)
	DeleteClientScopeProtocolMapper(ctx context.Context, realm, scopeID, mapperID string) (*Response, error)
}

type AuthorizationClient interface {
	// Scopes
	GetScopes(ctx context.Context, realm, clientUUID string) ([]ScopeRepresentation, *Response, error)
	CreateScope(ctx context.Context, realm, clientUUID string, scope ScopeRepresentation) (*Response, error)
	DeleteScope(ctx context.Context, realm, clientUUID, scopeID string) (*Response, error)
	// Resources
	GetResources(ctx context.Context, realm, clientUUID string) ([]ResourceRepresentation, *Response, error)
	CreateResource(
		ctx context.Context, realm, clientUUID string, resource ResourceRepresentation,
	) (*ResourceRepresentation, *Response, error)
	UpdateResource(
		ctx context.Context, realm, clientUUID, resourceID string, resource ResourceRepresentation,
	) (*Response, error)
	DeleteResource(ctx context.Context, realm, clientUUID, resourceID string) (*Response, error)
	// Policies
	GetPolicies(ctx context.Context, realm, clientUUID string) ([]AbstractPolicyRepresentation, *Response, error)
	CreatePolicy(
		ctx context.Context, realm, clientUUID, policyType string, policy any,
	) (*PolicyRepresentation, *Response, error)
	UpdatePolicy(
		ctx context.Context, realm, clientUUID, policyType, policyID string, policy any,
	) (*Response, error)
	GetPolicy(ctx context.Context, realm, clientUUID, policyType, policyID string) (*Response, error)
	DeletePolicy(ctx context.Context, realm, clientUUID, policyID string) (*Response, error)
	// Permissions
	GetPermissions(ctx context.Context, realm, clientUUID string) ([]AbstractPolicyRepresentation, *Response, error)
	CreatePermission(
		ctx context.Context, realm, clientUUID, permType string, perm PolicyRepresentation,
	) (*PolicyRepresentation, *Response, error)
	UpdatePermission(
		ctx context.Context, realm, clientUUID, permType, permID string, perm PolicyRepresentation,
	) (*Response, error)
	DeletePermission(ctx context.Context, realm, clientUUID, permID string) (*Response, error)
}

type ServerInfoClient interface {
	GetServerInfo(ctx context.Context) (*ServerInfo, error)
	FeatureFlagEnabled(ctx context.Context, featureFlag string) (bool, error)
}

type OrganizationsClient interface {
	GetOrganizations(
		ctx context.Context,
		realm string,
		params *GetOrganizationsParams,
	) ([]OrganizationRepresentation, *Response, error)
	GetOrganizationByAlias(ctx context.Context, realm, alias string) (*OrganizationRepresentation, *Response, error)
	CreateOrganization(ctx context.Context, realm string, org OrganizationRepresentation) (*Response, error)
	UpdateOrganization(ctx context.Context, realm, orgID string, org OrganizationRepresentation) (*Response, error)
	DeleteOrganization(ctx context.Context, realm, orgID string) (*Response, error)
	GetOrganizationIdentityProviders(
		ctx context.Context,
		realm, orgID string,
	) ([]IdentityProviderRepresentation, *Response, error)
	LinkIdentityProviderToOrganization(ctx context.Context, realm, orgID, alias string) (*Response, error)
	UnlinkIdentityProviderFromOrganization(ctx context.Context, realm, orgID, alias string) (*Response, error)
}
