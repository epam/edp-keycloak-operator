package mock

import (
	"github.com/stretchr/testify/mock"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
)

type MockKeycloakClient struct {
	mock.Mock
}

func (m MockKeycloakClient) ExistRealm(spec v1alpha1.KeycloakRealmSpec) (*bool, error) {
	args := m.Called(spec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	res := args.Bool(0)
	return &res, args.Error(1)
}

func (m MockKeycloakClient) CreateRealmWithDefaultConfig(spec v1alpha1.KeycloakRealmSpec) error {
	args := m.Called(spec)
	return args.Error(0)
}
