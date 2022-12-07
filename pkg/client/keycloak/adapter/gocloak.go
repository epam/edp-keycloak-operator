package adapter

import (
	"context"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-resty/resty/v2"
)

type GoCloak interface {
	RestyClient() *resty.Client
	LoginAdmin(ctx context.Context, username, password, realm string) (*gocloak.JWT, error)

	GoCloakRealms
	GoCloakClients
	GoCloakUsers
	GoCloakClientRoles
	GoCloakRealmRoles
	GoCloakGroups
}

type GoCloakRealms interface {
	GetRealm(ctx context.Context, token, realm string) (*gocloak.RealmRepresentation, error)
	DeleteRealm(ctx context.Context, token, realm string) error
	CreateRealm(ctx context.Context, token string, realm gocloak.RealmRepresentation) (string, error)
	UpdateRealm(ctx context.Context, token string, realm gocloak.RealmRepresentation) error
}

type GoCloakClients interface {
	GetClients(ctx context.Context, accessToken, realm string,
		params gocloak.GetClientsParams) ([]*gocloak.Client, error)
	DeleteClient(ctx context.Context, accessToken, realm, clientID string) error
	CreateClient(ctx context.Context, accessToken, realm string, clientID gocloak.Client) (string, error)
	UpdateClient(ctx context.Context, accessToken, realm string, updatedClient gocloak.Client) error
	CreateClientProtocolMapper(ctx context.Context, token, realm, clientID string,
		mapper gocloak.ProtocolMapperRepresentation) (string, error)
	UpdateClientProtocolMapper(ctx context.Context, token, realm, clientID, mapperID string,
		mapper gocloak.ProtocolMapperRepresentation) error
	DeleteClientProtocolMapper(ctx context.Context, token, realm, clientID, mapperID string) error
	GetClientServiceAccount(ctx context.Context, token, realm, clientID string) (*gocloak.User, error)
	DeleteClientScope(ctx context.Context, accessToken, realm, scopeID string) error
	GetClientScope(ctx context.Context, token, realm, scopeID string) (*gocloak.ClientScope, error)
	GetClientsDefaultScopes(ctx context.Context, token, realm, clientID string) ([]*gocloak.ClientScope, error)
	AddDefaultScopeToClient(ctx context.Context, token, realm, clientID, scopeID string) error
}

type GoCloakUsers interface {
	CreateUser(ctx context.Context, token, realm string, user gocloak.User) (string, error)
	GetUsers(ctx context.Context, accessToken, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error)
	GetRoleMappingByUserID(ctx context.Context, accessToken, realm,
		userID string) (*gocloak.MappingsRepresentation, error)
	UpdateUser(ctx context.Context, accessToken, realm string, user gocloak.User) error
}

type GoCloakClientRoles interface {
	GetClientRoles(ctx context.Context, accessToken, realm, clientID string, params gocloak.GetRoleParams) ([]*gocloak.Role, error)
	CreateClientRole(ctx context.Context, accessToken, realm, clientID string, role gocloak.Role) (string, error)
	GetClientRole(ctx context.Context, token, realm, clientID, roleName string) (*gocloak.Role, error)
	AddClientRoleToUser(ctx context.Context, token, realm, clientID, userID string, roles []gocloak.Role) error
	DeleteClientRoleFromUser(ctx context.Context, token, realm, clientID, userID string, roles []gocloak.Role) error
	AddClientRoleToGroup(ctx context.Context, token, realm, clientID, groupID string, roles []gocloak.Role) error
	DeleteClientRoleFromGroup(ctx context.Context, token, realm, clientID, groupID string, roles []gocloak.Role) error
}

type GoCloakRealmRoles interface {
	CreateRealmRole(ctx context.Context, token, realm string, role gocloak.Role) (string, error)
	GetRealmRole(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error)
	AddRealmRoleToUser(ctx context.Context, token, realm, userID string, roles []gocloak.Role) error
	UpdateRealmRole(ctx context.Context, token, realm, roleName string, role gocloak.Role) error
	DeleteRealmRole(ctx context.Context, token, realm, roleName string) error
	AddRealmRoleComposite(ctx context.Context, token, realm, roleName string, roles []gocloak.Role) error
	DeleteRealmRoleComposite(ctx context.Context, token, realm, roleName string, roles []gocloak.Role) error
	GetCompositeRealmRolesByRoleID(ctx context.Context, token, realm, roleID string) ([]*gocloak.Role, error)
	DeleteRealmRoleFromUser(ctx context.Context, token, realm, userID string, roles []gocloak.Role) error
	AddRealmRoleToGroup(ctx context.Context, token, realm, groupID string, roles []gocloak.Role) error
	DeleteRealmRoleFromGroup(ctx context.Context, token, realm, groupID string, roles []gocloak.Role) error
}

type GoCloakGroups interface {
	CreateGroup(ctx context.Context, accessToken, realm string, group gocloak.Group) (string, error)
	CreateChildGroup(ctx context.Context, token, realm, groupID string, group gocloak.Group) (string, error)
	UpdateGroup(ctx context.Context, accessToken, realm string, updatedGroup gocloak.Group) error
	DeleteGroup(ctx context.Context, accessToken, realm, groupID string) error
	GetGroups(ctx context.Context, accessToken, realm string, params gocloak.GetGroupsParams) ([]*gocloak.Group, error)
	GetRoleMappingByGroupID(ctx context.Context, accessToken, realm,
		groupID string) (*gocloak.MappingsRepresentation, error)
}
