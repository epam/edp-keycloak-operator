package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type SyncUserRoles struct{}

func NewSyncUserRoles() *SyncUserRoles {
	return &SyncUserRoles{}
}

func (h *SyncUserRoles) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	kClient keycloak.Client,
	realm *gocloak.RealmRepresentation,
	userCtx *UserContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing user roles")

	if err := kClient.SyncUserRoles(
		ctx,
		gocloak.PString(realm.Realm),
		userCtx.UserID,
		user.Spec.Roles,
		convertClientRoles(user.Spec.ClientRoles),
		user.IsReconciliationStrategyAddOnly(),
	); err != nil {
		return fmt.Errorf("unable to sync user roles: %w", err)
	}

	log.Info("User roles synced successfully")

	return nil
}

func convertClientRoles(apiClientRoles []keycloakApi.UserClientRole) map[string][]string {
	if apiClientRoles == nil {
		return nil
	}

	clientRolesMap := make(map[string][]string)
	for _, apiRole := range apiClientRoles {
		clientRolesMap[apiRole.ClientID] = apiRole.Roles
	}

	return clientRolesMap
}
