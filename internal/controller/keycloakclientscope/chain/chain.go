package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type ClientScopeHandler interface {
	Serve(
		ctx context.Context,
		scope *keycloakApi.KeycloakClientScope,
		realmName string,
	) error
}

type Chain struct {
	handlers []ClientScopeHandler
}

func (ch *Chain) Use(handlers ...ClientScopeHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	scope *keycloakApi.KeycloakClientScope,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakClientScope chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, scope, realmName)
		if err != nil {
			log.Info("KeycloakClientScope chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakClientScope has been finished")

	return nil
}

func MakeChain(keycloakAPIClient *keycloakapi.APIClient) *Chain {
	ch := &Chain{}

	ch.Use(
		NewCreateOrUpdateScope(keycloakAPIClient),
		NewSyncProtocolMappers(keycloakAPIClient),
		NewSetScopeType(keycloakAPIClient),
	)

	return ch
}
