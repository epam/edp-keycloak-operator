package chain

import (
	"context"

	"github.com/stretchr/testify/mock"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type Mock struct {
	mock.Mock
}

func (m *Mock) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, adapterClient keycloak.Client) error {
	return m.Called(keycloakClient).Error(0)
}
