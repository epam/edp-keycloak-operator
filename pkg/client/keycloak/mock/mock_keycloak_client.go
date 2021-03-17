package mock

import (
	"github.com/Nerzal/gocloak/v8"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/model"
	"github.com/stretchr/testify/mock"
)

type KeycloakClient struct {
	mock.Mock
}

func (m *KeycloakClient) PutDefaultIdp(realm *dto.Realm) error {
	return m.Called(realm).Error(0)
}

func (m *KeycloakClient) ExistRealm(realm string) (bool, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}

	return args.Bool(0), args.Error(1)
}

func (m *KeycloakClient) CreateRealmWithDefaultConfig(realm *dto.Realm) error {
	args := m.Called(realm)
	return args.Error(0)
}

func (m *KeycloakClient) ExistCentralIdentityProvider(realm *dto.Realm) (bool, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}
	res := args.Bool(0)
	return res, args.Error(1)
}

func (m *KeycloakClient) CreateCentralIdentityProvider(realm *dto.Realm, client *dto.Client) error {
	return m.Called(realm, client).Error(0)
}

func (m *KeycloakClient) ExistClient(client *dto.Client) (bool, error) {
	args := m.Called(client)
	if args.Get(0) == nil {
		return false, args.Error(1)
	}
	res := args.Bool(0)
	return res, args.Error(1)
}

func (m *KeycloakClient) CreateClient(client *dto.Client) error {
	args := m.Called(client)
	return args.Error(0)
}

func (m *KeycloakClient) ExistClientRole(role *dto.Client, clientRole string) (bool, error) {
	panic("implement me")
}

func (m *KeycloakClient) CreateClientRole(role *dto.Client, clientRole string) error {
	panic("implement me")
}

func (m *KeycloakClient) ExistRealmRole(realm string, role string) (bool, error) {
	args := m.Called(realm, role)
	return args.Bool(0), args.Error(1)
}

func (m *KeycloakClient) CreateRealmRole(realm string, role *dto.RealmRole) error {
	args := m.Called(realm, role)
	return args.Error(0)
}

func (m *KeycloakClient) ExistRealmUser(realmName string, user *dto.User) (bool, error) {
	called := m.Called(realmName, user)
	return called.Bool(0), called.Error(1)
}

func (m *KeycloakClient) CreateRealmUser(realmName string, user *dto.User) error {
	return m.Called(realmName, user).Error(0)
}

func (m *KeycloakClient) HasUserClientRole(realmName string, clientId string, user *dto.User, role string) (bool, error) {
	called := m.Called(realmName, clientId, user, role)
	return called.Bool(0), called.Error(1)
}

func (m *KeycloakClient) GetOpenIdConfig(realm *dto.Realm) (string, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}

	return args.String(0), args.Error(1)
}

func (m *KeycloakClient) AddClientRoleToUser(realmName string, clientId string, user *dto.User, role string) error {
	return m.Called(realmName, clientId, user, role).Error(0)
}

func (m *KeycloakClient) GetClientID(client *dto.Client) (string, error) {
	args := m.Called(client)
	if args.Get(0) == nil {
		return "", args.Error(1)
	}
	res := args.String(0)
	return res, args.Error(1)
}

func (m *KeycloakClient) MapRoleToUser(realmName string, user dto.User, role string) error {
	panic("implement me")
}

func (m *KeycloakClient) ExistMapRoleToUser(realmName string, user dto.User, role string) (*bool, error) {
	panic("implement me")
}

func (m *KeycloakClient) AddRealmRoleToUser(realmName string, user *dto.User, roleName string) error {
	return m.Called(realmName, user, roleName).Error(0)
}

func (m *KeycloakClient) CreateClientScope(realmName string, scope model.ClientScope) error {
	return m.Called(realmName, scope).Error(0)
}

func (m *KeycloakClient) DeleteClient(kkClientID string, realName string) error {
	return m.Called(kkClientID, realName).Error(0)
}

func (m *KeycloakClient) DeleteRealmRole(realm, roleName string) error {
	return m.Called(realm, roleName).Error(0)
}

func (m *KeycloakClient) DeleteRealm(realmName string) error {
	return m.Called(realmName).Error(0)
}

func (m *KeycloakClient) GetClientScope(scopeName, realmName string) (*model.ClientScope, error) {
	panic("implement me")
}

func (m *KeycloakClient) HasUserRealmRole(realmName string, user *dto.User, role string) (bool, error) {
	called := m.Called(realmName, user, role)
	return called.Bool(0), called.Error(1)
}

func (m *KeycloakClient) LinkClientScopeToClient(clientName, scopeId, realmName string) error {
	panic("implement me")
}

func (m *KeycloakClient) PutClientScopeMapper(clientName, scopeId, realmName string) error {
	panic("implement me")
}

func (m *KeycloakClient) SyncClientProtocolMapper(
	client *dto.Client, crMappers []gocloak.ProtocolMapperRepresentation) error {
	return m.Called(client, crMappers).Error(0)
}

func (m *KeycloakClient) SyncRealmRole(realm *dto.Realm, role *dto.RealmRole) error {
	return m.Called(realm, role).Error(0)
}

func (m *KeycloakClient) SyncServiceAccountRoles(realm, clientID string, realmRoles []string,
	clientRoles map[string][]string) error {
	return m.Called(realm, clientID, realmRoles, clientRoles).Error(0)
}

func (m *KeycloakClient) SyncRealmGroup(realmName string, spec *v1alpha1.KeycloakRealmGroupSpec) (string, error) {
	called := m.Called(realmName, spec)
	return called.String(0), called.Error(1)
}

func (m *KeycloakClient) DeleteGroup(realm, groupName string) error {
	return m.Called(realm, groupName).Error(0)
}
