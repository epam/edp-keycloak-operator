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
	DeleteClient(ctx context.Context, realm, clientID string) (*Response, error)
	GetClientRoles(
		ctx context.Context,
		realm string,
		clientID string,
		params *GetClientRolesParams,
	) ([]RoleRepresentation, *Response, error)
	GetClientRole(ctx context.Context, realm, clientID, roleName string) (*RoleRepresentation, *Response, error)
	CreateClientRole(ctx context.Context, realm, clientID string, role RoleRepresentation) (*Response, error)
	DeleteClientRole(ctx context.Context, realm, clientID, roleName string) (*Response, error)
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
