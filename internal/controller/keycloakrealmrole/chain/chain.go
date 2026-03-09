package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

// RoleContext holds data that is passed between chain handlers.
type RoleContext struct {
	// RoleID is the Keycloak role ID, set by CreateOrUpdateRole handler.
	RoleID string
}

type RealmRoleHandler interface {
	Serve(
		ctx context.Context,
		role *keycloakApi.KeycloakRealmRole,
		realmName string,
		roleCtx *RoleContext,
	) error
}

type Chain struct {
	handlers []RealmRoleHandler
}

func (ch *Chain) Use(handlers ...RealmRoleHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	role *keycloakApi.KeycloakRealmRole,
	realmName string,
	roleCtx *RoleContext,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakRealmRole chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, role, realmName, roleCtx)
		if err != nil {
			log.Info("KeycloakRealmRole chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakRealmRole has been finished")

	return nil
}

func MakeChain(kClientV2 *keycloakv2.KeycloakClient) *Chain {
	ch := &Chain{}

	ch.Use(
		NewCreateOrUpdateRole(kClientV2),
		NewSyncComposites(kClientV2),
		NewMakeDefault(kClientV2),
	)

	return ch
}
