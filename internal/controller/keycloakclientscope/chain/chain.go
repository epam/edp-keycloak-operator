package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
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

func MakeChain(kClientV2 *keycloakv2.KeycloakClient) *Chain {
	ch := &Chain{}

	ch.Use(
		NewCreateOrUpdateScope(kClientV2),
		NewSyncProtocolMappers(kClientV2),
		NewSetScopeType(kClientV2),
	)

	return ch
}
