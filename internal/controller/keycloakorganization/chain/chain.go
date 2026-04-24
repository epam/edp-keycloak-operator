package chain

import (
	"context"
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type Chain interface {
	Serve(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realmName string) error
}

type chain struct {
	handlers []Handler
}

func (c *chain) Serve(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realmName string) error {
	for _, handler := range c.handlers {
		if err := handler.ServeRequest(ctx, organization, realmName); err != nil {
			return fmt.Errorf("organization chain handler failed: %w", err)
		}
	}

	return nil
}

type Handler interface {
	ServeRequest(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realmName string) error
}

func MakeChain(kc *keycloakapi.KeycloakClient) Chain {
	return &chain{
		handlers: []Handler{
			NewCreateOrganization(kc),
			NewProcessIdentityProviders(kc),
		},
	}
}
