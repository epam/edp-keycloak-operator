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
}

type RealmClient interface {
	GetRealm(ctx context.Context, realm string) (*RealmRepresentation, *Response, error)
	CreateRealm(ctx context.Context, realmRep RealmRepresentation) (*Response, error)
	UpdateRealm(ctx context.Context, realm string, realmRep RealmRepresentation) (*Response, error)
	DeleteRealm(ctx context.Context, realm string) (*Response, error)
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
	DeleteRealmRole(ctx context.Context, realm, roleName string) (*Response, error)
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
