package mock

import (
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/stretchr/testify/mock"
)

type MockKeycloakClient struct {
	mock.Mock
}

func (m MockKeycloakClient) MapRoleToUser(realmName string, user dto.User, role string) error {
	panic("implement me")
}

func (m MockKeycloakClient) CreateRealmUser(realmName string, user dto.User) error {
	panic("implement me")
}

func (m MockKeycloakClient) ExistMapRoleToUser(realmName string, user dto.User, role string) (*bool, error) {
	panic("implement me")
}

func (m MockKeycloakClient) ExistRealmUser(realmName string, user dto.User) (*bool, error) {
	panic("implement me")
}

func (m MockKeycloakClient) CreateClientRole(role dto.Client, clientRole string) error {
	panic("implement me")
}

func (m MockKeycloakClient) ExistClientRole(role dto.Client, clientRole string) (*bool, error) {
	panic("implement me")
}

func (m MockKeycloakClient) ExistRealmRole(realm dto.Realm, role dto.RealmRole) (*bool, error) {
	panic("implement me")
}

func (m MockKeycloakClient) CreateRealmRole(realm dto.Realm, role dto.RealmRole) error {
	panic("implement me")
}

func (m MockKeycloakClient) CreateClient(client dto.Client) error {
	panic("implement me")
}

func (m MockKeycloakClient) ExistClient(client dto.Client) (*bool, error) {
	panic("implement me")
}

func (m MockKeycloakClient) CreateCentralIdentityProvider(realm dto.Realm, client dto.Client) error {
	panic("implement me")
}

func (m MockKeycloakClient) ExistCentralIdentityProvider(realm dto.Realm) (*bool, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res := args.Bool(0)
	return &res, args.Error(1)
}

func (m MockKeycloakClient) ExistRealm(realm dto.Realm) (*bool, error) {
	args := m.Called(realm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res := args.Bool(0)
	return &res, args.Error(1)
}

func (m MockKeycloakClient) CreateRealmWithDefaultConfig(realm dto.Realm) error {
	args := m.Called(realm)
	return args.Error(0)
}
