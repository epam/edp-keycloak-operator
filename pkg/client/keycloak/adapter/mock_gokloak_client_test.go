package adapter

import (
	"context"

	"github.com/Nerzal/gocloak/v8"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/mock"
)

type MockGoCloakClient struct {
	mock.Mock
}

func (m *MockGoCloakClient) LoginAdmin(ctx context.Context, username, password, realm string) (*gocloak.JWT, error) {
	args := m.Called(username, password, realm)
	return args.Get(0).(*gocloak.JWT), args.Error(1)
}

func (m *MockGoCloakClient) GetRealm(ctx context.Context, token, realm string) (*gocloak.RealmRepresentation, error) {
	args := m.Called(token, realm)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*gocloak.RealmRepresentation), args.Error(1)
}

func (m *MockGoCloakClient) CreateRealm(ctx context.Context, token string, realm gocloak.RealmRepresentation) (string, error) {
	args := m.Called(token, realm)
	return "", args.Error(0)
}

func (m *MockGoCloakClient) AddClientRoleToUser(ctx context.Context, token, realm, clientID, userID string,
	roles []gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) AddRealmRoleComposite(ctx context.Context, token, realm, roleName string,
	roles []gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) AddRealmRoleToUser(ctx context.Context, token, realm, userID string,
	roles []gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateClient(ctx context.Context, accessToken, realm string,
	clientID gocloak.Client) (string, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateClientRole(ctx context.Context, accessToken, realm, clientID string,
	role gocloak.Role) (string, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateRealmRole(ctx context.Context, token, realm string,
	role gocloak.Role) (string, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateUser(ctx context.Context, token, realm string, user gocloak.User) (string, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteClient(ctx context.Context, accessToken, realm, clientID string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteRealm(ctx context.Context, token, realm string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClientRole(ctx context.Context, token, realm, clientID,
	roleName string) (*gocloak.Role, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClientRoles(ctx context.Context, accessToken, realm,
	clientID string) ([]*gocloak.Role, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClients(ctx context.Context, accessToken, realm string,
	params gocloak.GetClientsParams) ([]*gocloak.Client, error) {
	args := m.Called(realm, params)
	return args.Get(0).([]*gocloak.Client), args.Error(1)
}

func (m *MockGoCloakClient) GetRealmRole(ctx context.Context, token, realm, roleName string) (*gocloak.Role, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetRoleMappingByUserID(ctx context.Context, accessToken, realm,
	userID string) (*gocloak.MappingsRepresentation, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetUsers(ctx context.Context, accessToken, realm string,
	params gocloak.GetUsersParams) ([]*gocloak.User, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) RestyClient() *resty.Client {
	args := m.Called()
	return args.Get(0).(*resty.Client)
}

func (m *MockGoCloakClient) CreateClientProtocolMapper(ctx context.Context, token, realm, clientID string,
	mapper gocloak.ProtocolMapperRepresentation) (string, error) {
	args := m.Called(realm, clientID, mapper)
	return args.String(0), args.Error(1)
}
func (m *MockGoCloakClient) UpdateClientProtocolMapper(ctx context.Context, token, realm, clientID, mapperID string,
	mapper gocloak.ProtocolMapperRepresentation) error {
	args := m.Called(realm, clientID, mapperID, mapper)
	return args.Error(0)
}
func (m *MockGoCloakClient) DeleteClientProtocolMapper(ctx context.Context, token, realm, clientID,
	mapperID string) error {
	args := m.Called(realm, clientID, mapperID)
	return args.Error(0)
}
