package handler

import (
	"context"

	"github.com/stretchr/testify/mock"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type MockRealmHandler struct {
	mock.Mock
}

func (m *MockRealmHandler) ServeRequest(_ context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
	args := m.Called(realm, kClient)
	return args.Error(0)
}
