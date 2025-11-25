package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type SyncUserGroups struct{}

func NewSyncUserGroups() *SyncUserGroups {
	return &SyncUserGroups{}
}

func (h *SyncUserGroups) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	kClient keycloak.Client,
	realm *gocloak.RealmRepresentation,
	userCtx *UserContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing user groups")

	if err := kClient.SyncUserGroups(
		ctx,
		gocloak.PString(realm.Realm),
		userCtx.UserID,
		user.Spec.Groups,
		user.IsReconciliationStrategyAddOnly(),
	); err != nil {
		return fmt.Errorf("unable to sync user groups: %w", err)
	}

	log.Info("User groups synced successfully")

	return nil
}
