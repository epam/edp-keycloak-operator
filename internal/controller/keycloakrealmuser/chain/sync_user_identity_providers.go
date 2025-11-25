package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type SyncUserIdentityProviders struct{}

func NewSyncUserIdentityProviders() *SyncUserIdentityProviders {
	return &SyncUserIdentityProviders{}
}

func (h *SyncUserIdentityProviders) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	kClient keycloak.Client,
	realm *gocloak.RealmRepresentation,
	userCtx *UserContext,
) error {
	if user.Spec.IdentityProviders == nil {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing user identity providers")

	if err := kClient.SyncUserIdentityProviders(
		ctx,
		gocloak.PString(realm.Realm),
		userCtx.UserID,
		user.Spec.Username,
		*user.Spec.IdentityProviders,
	); err != nil {
		return fmt.Errorf("unable to sync user identity providers: %w", err)
	}

	log.Info("User identity providers synced successfully")

	return nil
}
