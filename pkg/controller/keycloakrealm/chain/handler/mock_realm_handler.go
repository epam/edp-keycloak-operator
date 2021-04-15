package handler

import (
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/stretchr/testify/mock"
)

type MockRealmHandler struct {
	mock.Mock
}

func (m *MockRealmHandler) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	args := m.Called(realm, kClient)
	return args.Error(0)
}
