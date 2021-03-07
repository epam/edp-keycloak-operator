package mock

import (
	"github.com/Nerzal/gocloak/v8"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/model"
	"github.com/stretchr/testify/mock"
)

type KeycloakClient struct {
	mock.Mock
}

func (m *KeycloakClient) PutDefaultIdp(realm dto.Realm) error {
	panic("implement me")
}

func (m *KeycloakClient) ExistRealm(realm dto.Realm) (*bool, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res := args.Bool(0)
	return &res, args.Error(1)
}

func (m *KeycloakClient) CreateRealmWithDefaultConfig(realm dto.Realm) error {
	args := m.Called(realm)
	return args.Error(0)
}

func (m *KeycloakClient) ExistCentralIdentityProvider(realm dto.Realm) (*bool, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res := args.Bool(0)
	return &res, args.Error(1)
}

func (m *KeycloakClient) CreateCentralIdentityProvider(realm dto.Realm, client dto.Client) error {
	panic("implement me")
}

func (m *KeycloakClient) ExistClient(client dto.Client) (*bool, error) {
	args := m.Called(client)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res := args.Bool(0)
	return &res, args.Error(1)
}

func (m *KeycloakClient) CreateClient(client dto.Client) error {
	args := m.Called(client)
	return args.Error(0)
}

func (m *KeycloakClient) ExistClientRole(role dto.Client, clientRole string) (*bool, error) {
	panic("implement me")
}

func (m *KeycloakClient) CreateClientRole(role dto.Client, clientRole string) error {
	panic("implement me")
}

func (m *KeycloakClient) ExistRealmRole(realm dto.Realm, role dto.RealmRole) (*bool, error) {
	args := m.Called(realm, role)
	res := args.Bool(0)
	return &res, args.Error(1)
}

func (m *KeycloakClient) CreateRealmRole(realm dto.Realm, role dto.RealmRole) error {
	args := m.Called(realm, role)
	return args.Error(0)
}

func (m *KeycloakClient) ExistRealmUser(realmName string, user dto.User) (*bool, error) {
	panic("implement me")
}

func (m *KeycloakClient) CreateRealmUser(realmName string, user dto.User) error {
	panic("implement me")
}

func (m *KeycloakClient) HasUserClientRole(realmName string, clientId string, user dto.User, role string) (*bool, error) {
	panic("implement me")
}

func (m *KeycloakClient) GetOpenIdConfig(realm dto.Realm) (*string, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res := args.String(0)
	return &res, args.Error(1)
}

func (m *KeycloakClient) AddClientRoleToUser(realmName string, clientId string, user dto.User, role string) error {
	panic("implement me")
}

func (m *KeycloakClient) GetClientId(client dto.Client) (*string, error) {
	args := m.Called(client)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res := args.String(0)
	return &res, args.Error(1)
}

func (m *KeycloakClient) MapRoleToUser(realmName string, user dto.User, role string) error {
	panic("implement me")
}

func (m *KeycloakClient) ExistMapRoleToUser(realmName string, user dto.User, role string) (*bool, error) {
	panic("implement me")
}

func (m *KeycloakClient) AddRealmRoleToUser(realmName string, user dto.User, roleName string) error {
	return m.Called(realmName, user, roleName).Error(0)
}

func (m *KeycloakClient) CreateClientScope(realmName string, scope model.ClientScope) error {
	return m.Called(realmName, scope).Error(0)
}

func (m *KeycloakClient) DeleteClient(kkClientID string, client dto.Client) error {
	return m.Called(kkClientID, client).Error(0)
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

func (m *KeycloakClient) HasUserRealmRole(realmName string, user dto.User, role string) (bool, error) {
	panic("implement me")
}

func (m *KeycloakClient) LinkClientScopeToClient(clientName, scopeId, realmName string) error {
	panic("implement me")
}

func (m *KeycloakClient) PutClientScopeMapper(clientName, scopeId, realmName string) error {
	panic("implement me")
}

func (m *KeycloakClient) SyncClientProtocolMapper(
	client dto.Client, crMappers []gocloak.ProtocolMapperRepresentation) error {
	panic("implement me")
}

func (m *KeycloakClient) SyncRealmRole(realm *dto.Realm, role *dto.RealmRole) error {
	return m.Called(realm, role).Error(0)
}
