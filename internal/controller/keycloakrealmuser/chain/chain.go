package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// UserContext holds data that is passed between chain handlers.
type UserContext struct {
	// UserID is the Keycloak user ID, set by CreateOrUpdateUser handler.
	UserID string
}

type RealmUserHandler interface {
	Serve(
		ctx context.Context,
		user *keycloakApi.KeycloakRealmUser,
		realmName string,
		userCtx *UserContext,
	) error
}

type Chain struct {
	handlers []RealmUserHandler
}

func (ch *Chain) Use(handlers ...RealmUserHandler) {
	ch.handlers = append(ch.handlers, handlers...)
}

func (ch *Chain) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Starting KeycloakRealmUser chain")

	userCtx := &UserContext{}

	for i := 0; i < len(ch.handlers); i++ {
		h := ch.handlers[i]

		err := h.Serve(ctx, user, realmName, userCtx)
		if err != nil {
			log.Info("KeycloakRealmUser chain finished with error")

			return fmt.Errorf("failed to serve handler: %w", err)
		}
	}

	log.Info("Handling of KeycloakRealmUser has been finished")

	return nil
}

func MakeChain(
	k8sClient client.Client,
	keycloakAPIClient *keycloakapi.APIClient,
) *Chain {
	ch := &Chain{}

	ch.Use(
		NewCreateOrUpdateUser(k8sClient, keycloakAPIClient),
		NewSetUserPassword(k8sClient, keycloakAPIClient),
		NewSyncUserRoles(keycloakAPIClient),
		NewSyncUserGroups(keycloakAPIClient),
		NewSyncUserIdentityProviders(keycloakAPIClient),
		NewCleanupResource(k8sClient),
	)

	return ch
}
