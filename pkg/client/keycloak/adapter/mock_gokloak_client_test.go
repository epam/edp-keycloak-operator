package adapter

import (
	"github.com/Nerzal/gocloak/v3"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/mock"
	"gopkg.in/resty.v1"
)

type MockGoCloakClient struct {
	mock.Mock
}

func (m *MockGoCloakClient) LoginAdmin(username, password, realm string) (*gocloak.JWT, error) {
	args := m.Called(username, password, realm)
	return args.Get(0).(*gocloak.JWT), args.Error(1)
}

func (m *MockGoCloakClient) RestyClient() *resty.Client {
	panic("implement me")
}

func (m *MockGoCloakClient) GetToken(realm string, options gocloak.TokenOptions) (*gocloak.JWT, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) Login(clientID, clientSecret, realm, username, password string) (*gocloak.JWT, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) Logout(clientID, clientSecret, realm, refreshToken string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) LoginClient(clientID, clientSecret, realm string) (*gocloak.JWT, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) RequestPermission(clientID, clientSecret, realm, username, password, permission string) (*gocloak.JWT, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) RefreshToken(refreshToken string, clientID, clientSecret, realm string) (*gocloak.JWT, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) DecodeAccessToken(accessToken string, realm string) (*jwt.Token, *jwt.MapClaims, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) DecodeAccessTokenCustomClaims(accessToken string, realm string, claims jwt.Claims) (*jwt.Token, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) RetrospectToken(accessToken string, clientID, clientSecret string, realm string) (*gocloak.RetrospecTokenResult, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetIssuer(realm string) (*gocloak.IssuerResponse, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetCerts(realm string) (*gocloak.CertResponse, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetServerInfo(accessToken string) (*gocloak.ServerInfoRepesentation, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetUserInfo(accessToken string, realm string) (*gocloak.UserInfo, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) ExecuteActionsEmail(token string, realm string, params gocloak.ExecuteActionsEmail) error {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateGroup(accessToken string, realm string, group gocloak.Group) error {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateClientRole(accessToken string, realm string, clientID string, role gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateClient(accessToken string, realm string, clientID gocloak.Client) error {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateClientScope(accessToken string, realm string, scope gocloak.ClientScope) error {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateComponent(accessToken string, realm string, component gocloak.Component) error {
	panic("implement me")
}

func (m *MockGoCloakClient) UpdateGroup(accessToken string, realm string, updatedGroup gocloak.Group) error {
	panic("implement me")
}

func (m *MockGoCloakClient) UpdateRole(accessToken string, realm string, clientID string, role gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) UpdateClient(accessToken string, realm string, updatedClient gocloak.Client) error {
	panic("implement me")
}

func (m *MockGoCloakClient) UpdateClientScope(accessToken string, realm string, scope gocloak.ClientScope) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteComponent(accessToken string, realm, componentID string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteGroup(accessToken string, realm, groupID string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteClientRole(accessToken string, realm, clientID, roleName string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteClient(accessToken string, realm, clientID string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteClientScope(accessToken string, realm, scopeID string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClient(accessToken string, realm string, clientID string) (*gocloak.Client, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClientSecret(token string, realm string, clientID string) (*gocloak.CredentialRepresentation, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetKeyStoreConfig(accessToken string, realm string) (*gocloak.KeyStoreConfig, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetComponents(accessToken string, realm string) (*[]gocloak.Component, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetGroups(accessToken string, realm string, params gocloak.GetGroupsParams) (*[]gocloak.Group, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetGroup(accessToken string, realm, groupID string) (*gocloak.Group, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetRoleMappingByGroupID(accessToken string, realm string, groupID string) (*gocloak.MappingsRepresentation, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetRoleMappingByUserID(accessToken string, realm string, userID string) (*gocloak.MappingsRepresentation, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClientRoles(accessToken string, realm string, clientID string) (*[]gocloak.Role, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClientRole(token string, realm string, clientID string, roleName string) (*gocloak.Role, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClients(accessToken string, realm string, params gocloak.GetClientsParams) (*[]gocloak.Client, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClientOfflineSessions(token, realm, clientID string) (*[]gocloak.UserSessionRepresentation, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetClientUserSessions(token, realm, clientID string) (*[]gocloak.UserSessionRepresentation, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) UserAttributeContains(attributes map[string][]string, attribute string, value string) bool {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateRealmRole(token string, realm string, role gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) GetRealmRole(token string, realm string, roleName string) (*gocloak.Role, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetRealmRoles(accessToken string, realm string) (*[]gocloak.Role, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetRealmRolesByUserID(accessToken string, realm string, userID string) (*[]gocloak.Role, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetRealmRolesByGroupID(accessToken string, realm string, groupID string) (*[]gocloak.Role, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) UpdateRealmRole(token string, realm string, roleName string, role gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteRealmRole(token string, realm string, roleName string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) AddRealmRoleToUser(token string, realm string, userID string, roles []gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteRealmRoleFromUser(token string, realm string, userID string, roles []gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) AddRealmRoleComposite(token string, realm string, roleName string, roles []gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteRealmRoleComposite(token string, realm string, roleName string, roles []gocloak.Role) error {
	panic("implement me")
}

func (m *MockGoCloakClient) GetRealm(token string, realm string) (*gocloak.RealmRepresentation, error) {
	args := m.Called(token, realm)
	res := args.Get(0)
	if res == nil {
		return nil, args.Error(1)
	}
	return res.(*gocloak.RealmRepresentation), args.Error(1)
}

func (m *MockGoCloakClient) CreateRealm(token string, realm gocloak.RealmRepresentation) error {
	args := m.Called(token, realm)
	return args.Error(0)
}

func (m *MockGoCloakClient) DeleteRealm(token string, realm string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) CreateUser(token string, realm string, user gocloak.User) (*string, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteUser(accessToken string, realm, userID string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) GetUserByID(accessToken string, realm string, userID string) (*gocloak.User, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetUserCount(accessToken string, realm string) (int, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetUsers(accessToken string, realm string, params gocloak.GetUsersParams) (*[]gocloak.User, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetUserGroups(accessToken string, realm string, userID string) (*[]gocloak.UserGroup, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetUsersByRoleName(token string, realm string, roleName string) (*[]gocloak.User, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) SetPassword(token string, userID string, realm string, password string, temporary bool) error {
	panic("implement me")
}

func (m *MockGoCloakClient) UpdateUser(accessToken string, realm string, user gocloak.User) error {
	panic("implement me")
}

func (m *MockGoCloakClient) AddUserToGroup(token string, realm string, userID string, groupID string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) DeleteUserFromGroup(token string, realm string, userID string, groupID string) error {
	panic("implement me")
}

func (m *MockGoCloakClient) GetUserSessions(token, realm, userID string) (*[]gocloak.UserSessionRepresentation, error) {
	panic("implement me")
}

func (m *MockGoCloakClient) GetUserOfflineSessionsForClient(token, realm, userID, clientID string) (*[]gocloak.UserSessionRepresentation, error) {
	panic("implement me")
}
