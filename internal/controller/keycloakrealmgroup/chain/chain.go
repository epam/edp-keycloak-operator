package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

// GroupContext holds data that is passed between chain handlers.
type GroupContext struct {
	// GroupID is the Keycloak group ID, set by CreateOrUpdateGroup handler.
	GroupID string

	// ParentGroupID is the parent group's Keycloak ID (empty if top-level).
	ParentGroupID string

	// RealmName is the Keycloak realm name.
	RealmName string
}

// RealmGroupHandler defines the interface for chain handlers.
type RealmGroupHandler interface {
	Serve(
		ctx context.Context,
		group *keycloakApi.KeycloakRealmGroup,
		kClient *keycloakv2.KeycloakClient,
		groupCtx *GroupContext,
	) error
}

// Chain executes a sequence of RealmGroupHandler handlers.
type Chain struct {
	handlers []RealmGroupHandler
}

func (ch *Chain) Use(handlers ...RealmGroupHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	group *keycloakApi.KeycloakRealmGroup,
	kClient *keycloakv2.KeycloakClient,
	groupCtx *GroupContext,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakRealmGroup chain")

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, group, kClient, groupCtx)
		if err != nil {
			log.Info("KeycloakRealmGroup chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakRealmGroup has been finished")

	return nil
}

func MakeChain() *Chain {
	ch := &Chain{}

	ch.Use(
		NewCreateOrUpdateGroup(),
		NewSyncRealmRoles(),
		NewSyncClientRoles(),
		NewSyncSubGroups(),
	)

	return ch
}
