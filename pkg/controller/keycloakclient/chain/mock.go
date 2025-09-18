package chain

import (
	"context"

	"github.com/stretchr/testify/mock"

	v1v1alpha1 "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Serve(ctx context.Context, keycloakClient *v1v1alpha1.KeycloakClient, adapterClient keycloak.Client) error {
	return m.Called(keycloakClient).Error(0)
}
