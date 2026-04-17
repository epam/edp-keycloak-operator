package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// AuthFlowHandler is a single step in the KeycloakAuthFlow reconciliation chain.
type AuthFlowHandler interface {
	Serve(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error
}

// Chain sequentially executes a list of AuthFlowHandlers.
type Chain struct {
	handlers []AuthFlowHandler
}

func (ch *Chain) Use(handlers ...AuthFlowHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(ctx context.Context, flow *keycloakApi.KeycloakAuthFlow, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakAuthFlow chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		if err := h.Serve(ctx, flow, realmName); err != nil {
			log.Info("KeycloakAuthFlow chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakAuthFlow has been finished")

	return nil
}

// MakeChain creates the default reconciliation chain for KeycloakAuthFlow.
func MakeChain(keycloakAPIClient *keycloakapi.APIClient) *Chain {
	ch := &Chain{}

	ch.Use(
		NewCreateOrUpdateAuthFlow(keycloakAPIClient),
		NewSyncAuthFlowExecutions(keycloakAPIClient),
	)

	return ch
}
