package adapter

import (
	"context"

	"github.com/Nerzal/gocloak/v8"
	"github.com/go-resty/resty/v2"
)

type GoCloak interface {
	GetRealm(ctx context.Context, token, realm string) (*gocloak.RealmRepresentation, error)
	DeleteRealm(ctx context.Context, token, realm string) error
	CreateRealm(ctx context.Context, token string, realm gocloak.RealmRepresentation) (string, error)
	RestyClient() *resty.Client
	GetClients(ctx context.Context, accessToken, realm string,
		params gocloak.GetClientsParams) ([]*gocloak.Client, error)
	GetClientRoles(ctx context.Context, accessToken, realm, clientID string) ([]*gocloak.Role, error)
	CreateClientRole(ctx context.Context, accessToken, realm, clientID string, role gocloak.Role) (string, error)
	DeleteClient(ctx context.Context, accessToken, realm, clientID string) error
	CreateClient(ctx context.Context, accessToken, realm string, clientID gocloak.Client) (string, error)
	CreateUser(ctx context.Context, token, realm string, user gocloak.User) (string, error)
	GetUsers(ctx context.Context, accessToken, realm string, params gocloak.GetUsersParams) ([]*gocloak.User, error)
	GetRoleMappingByUserID(ctx context.Context, accessToken, realm,
		userID string) (*gocloak.MappingsRepresentation, error)
	GetRealmRole(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error)
	AddRealmRoleToUser(ctx context.Context, token, realm, userID string, roles []gocloak.Role) error
	GetClientRole(ctx context.Context, token, realm, clientID, roleName string) (*gocloak.Role, error)
	AddClientRoleToUser(ctx context.Context, token, realm, clientID, userID string, roles []gocloak.Role) error
	CreateRealmRole(ctx context.Context, token, realm string, role gocloak.Role) (string, error)
	AddRealmRoleComposite(ctx context.Context, token, realm, roleName string, roles []gocloak.Role) error
	LoginAdmin(ctx context.Context, username, password, realm string) (*gocloak.JWT, error)
	CreateClientProtocolMapper(ctx context.Context, token, realm, clientID string,
		mapper gocloak.ProtocolMapperRepresentation) (string, error)
	UpdateClientProtocolMapper(ctx context.Context, token, realm, clientID, mapperID string,
		mapper gocloak.ProtocolMapperRepresentation) error
	DeleteClientProtocolMapper(ctx context.Context, token, realm, clientID, mapperID string) error
}
