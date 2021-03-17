package adapter

import (
	"context"
	"sort"

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
	return m.Called(realm, clientID, userID, roles).Error(0)
}

func (m *MockGoCloakClient) AddRealmRoleComposite(ctx context.Context, token, realm, roleName string,
	roles []gocloak.Role) error {
	return m.Called(realm, roleName, roles).Error(0)
}

func (m *MockGoCloakClient) AddRealmRoleToUser(ctx context.Context, token, realm, userID string,
	roles []gocloak.Role) error {
	return m.Called(realm, userID, roles).Error(0)
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
	called := m.Called(realm, clientID, roleName)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*gocloak.Role), nil
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
	called := m.Called(realm, roleName)
	if err := called.Error(1); err != nil {
		return nil, err
	}
	return called.Get(0).(*gocloak.Role), nil
}

func (m *MockGoCloakClient) GetRoleMappingByUserID(ctx context.Context, accessToken, realm,
	userID string) (*gocloak.MappingsRepresentation, error) {
	called := m.Called(realm, userID)
	err := called.Error(1)
	if err != nil {
		return nil, err
	}

	return called.Get(0).(*gocloak.MappingsRepresentation), nil
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

func (m *MockGoCloakClient) DeleteRealmRole(ctx context.Context, token, realm, roleName string) error {
	return m.Called(realm, roleName).Error(0)
}

func (m *MockGoCloakClient) DeleteRealmRoleComposite(ctx context.Context, token, realm, roleName string,
	roles []gocloak.Role) error {
	return m.Called(realm, roleName, roles).Error(0)
}

func (m *MockGoCloakClient) GetCompositeRealmRolesByRoleID(ctx context.Context, token, realm,
	roleID string) ([]*gocloak.Role, error) {
	called := m.Called(realm, roleID)
	return called.Get(0).([]*gocloak.Role), called.Error(1)
}

func (m *MockGoCloakClient) UpdateRealmRole(ctx context.Context, token, realm, roleName string,
	role gocloak.Role) error {
	return m.Called(realm, roleName, role).Error(0)
}

type RoleSorter []gocloak.Role

func (a RoleSorter) Len() int           { return len(a) }
func (a RoleSorter) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a RoleSorter) Less(i, j int) bool { return *a[i].Name < *a[j].Name }

func (m *MockGoCloakClient) DeleteClientRoleFromUser(ctx context.Context, token, realm, clientID, userID string,
	roles []gocloak.Role) error {

	sort.Sort(RoleSorter(roles))
	return m.Called(realm, clientID, userID, roles).Error(0)
}

func (m *MockGoCloakClient) DeleteRealmRoleFromUser(ctx context.Context, token, realm, userID string,
	roles []gocloak.Role) error {

	sort.Sort(RoleSorter(roles))
	return m.Called(realm, userID, roles).Error(0)
}

func (m *MockGoCloakClient) GetClientServiceAccount(ctx context.Context, token, realm,
	clientID string) (*gocloak.User, error) {
	called := m.Called(realm, clientID)
	err := called.Error(1)
	if err != nil {
		return nil, err
	}
	return called.Get(0).(*gocloak.User), nil
}

func (m *MockGoCloakClient) DeleteClientRoleFromGroup(ctx context.Context, token, realm, clientID, groupID string,
	roles []gocloak.Role) error {
	sort.Sort(RoleSorter(roles))

	return m.Called(realm, clientID, groupID, roles).Error(0)
}

func (m *MockGoCloakClient) AddClientRoleToGroup(ctx context.Context, token, realm, clientID, groupID string,
	roles []gocloak.Role) error {
	sort.Sort(RoleSorter(roles))

	return m.Called(realm, clientID, groupID, roles).Error(0)
}

func (m *MockGoCloakClient) DeleteRealmRoleFromGroup(ctx context.Context, token, realm, groupID string,
	roles []gocloak.Role) error {
	sort.Sort(RoleSorter(roles))
	return m.Called(realm, groupID, roles).Error(0)
}

func (m *MockGoCloakClient) AddRealmRoleToGroup(ctx context.Context, token, realm, groupID string,
	roles []gocloak.Role) error {
	sort.Sort(RoleSorter(roles))
	return m.Called(realm, groupID, roles).Error(0)
}

func (m *MockGoCloakClient) GetRoleMappingByGroupID(ctx context.Context, accessToken, realm,
	groupID string) (*gocloak.MappingsRepresentation, error) {
	called := m.Called(realm, groupID)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).(*gocloak.MappingsRepresentation), nil
}

func (m *MockGoCloakClient) GetGroups(ctx context.Context, accessToken, realm string,
	params gocloak.GetGroupsParams) ([]*gocloak.Group, error) {
	called := m.Called(realm, params)
	if err := called.Error(1); err != nil {
		return nil, err
	}

	return called.Get(0).([]*gocloak.Group), nil
}

func (m *MockGoCloakClient) DeleteGroup(ctx context.Context, accessToken, realm, groupID string) error {
	return m.Called(realm, groupID).Error(0)
}

func (m *MockGoCloakClient) UpdateGroup(ctx context.Context, accessToken, realm string,
	updatedGroup gocloak.Group) error {
	return m.Called(realm, updatedGroup).Error(0)
}

func (m *MockGoCloakClient) CreateChildGroup(ctx context.Context, token, realm, groupID string,
	group gocloak.Group) (string, error) {
	called := m.Called(realm, groupID, group)
	return called.String(0), called.Error(1)
}

func (m *MockGoCloakClient) CreateGroup(ctx context.Context, accessToken, realm string,
	group gocloak.Group) (string, error) {
	called := m.Called(realm, group)
	return called.String(0), called.Error(1)
}
