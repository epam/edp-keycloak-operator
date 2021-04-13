package handler

import (
	"github.com/epam/keycloak-operator/v2/pkg/apis/v1/v1alpha1"
	"github.com/epam/keycloak-operator/v2/pkg/client/keycloak"
	"github.com/stretchr/testify/mock"
)

type MockRealmHandler struct {
	mock.Mock
}

func (m *MockRealmHandler) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	args := m.Called(realm, kClient)
	return args.Error(0)
}
