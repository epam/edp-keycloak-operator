package mock

import (
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/stretchr/testify/mock"
)

type MockKeycloakClient struct {
	mock.Mock
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
