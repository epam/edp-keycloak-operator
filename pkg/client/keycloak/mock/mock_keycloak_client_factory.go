package mock

import (
	"github.com/stretchr/testify/mock"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
	"keycloak-operator/pkg/client/keycloak"
)

type MockGoCloakFactory struct {
	mock.Mock
}

func (m MockGoCloakFactory) New(spec v1alpha1.KeycloakSpec) (keycloak.Client, error) {
	args := m.Called(spec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(keycloak.Client), args.Error(1)
}
