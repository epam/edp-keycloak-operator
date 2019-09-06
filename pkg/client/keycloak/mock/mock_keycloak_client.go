package mock

import (
	"github.com/stretchr/testify/mock"
	"keycloak-operator/pkg/client/keycloak/dto"
)

type MockKeycloakClient struct {
	mock.Mock
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
