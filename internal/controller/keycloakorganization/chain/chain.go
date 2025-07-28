package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type Chain interface {
	Serve(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realm *gocloak.RealmRepresentation) error
}

type chain struct {
	handlers []Handler
}

func (c *chain) Serve(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realm *gocloak.RealmRepresentation) error {
	for _, handler := range c.handlers {
		if err := handler.ServeRequest(ctx, organization, realm); err != nil {
			return fmt.Errorf("organization chain handler failed: %w", err)
		}
	}

	return nil
}

type Handler interface {
	ServeRequest(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realm *gocloak.RealmRepresentation) error
}

func MakeChain(keycloakClient keycloak.Client, k8sClient client.Client) Chain {
	return &chain{
		handlers: []Handler{
			NewCreateOrganization(keycloakClient),
			NewProcessIdentityProviders(keycloakClient),
		},
	}
}
