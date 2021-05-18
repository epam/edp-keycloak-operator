package chain

import (
	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/stretchr/testify/mock"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Serve(keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
	return m.Called(keycloakClient).Error(0)
}
