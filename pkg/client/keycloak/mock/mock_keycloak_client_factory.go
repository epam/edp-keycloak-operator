package mock

import (
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/stretchr/testify/mock"
)

type GoCloakFactory struct {
	mock.Mock
}

func (m *GoCloakFactory) New(dto dto.Keycloak) (keycloak.Client, error) {
	args := m.Called(dto)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(keycloak.Client), args.Error(1)
}
