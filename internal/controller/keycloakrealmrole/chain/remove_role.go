package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

type RemoveRole struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewRemoveRole(kClientV2 *keycloakv2.KeycloakClient) *RemoveRole {
	return &RemoveRole{kClientV2: kClientV2}
}

func (h *RemoveRole) ServeRequest(ctx context.Context, role *keycloakApi.KeycloakRealmRole, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start removing realm role")

	if objectmeta.PreserveResourcesOnDeletion(role) {
		log.Info("Preserve resources on deletion, skipping")

		return nil
	}

	if _, err := h.kClientV2.Roles.DeleteRealmRole(ctx, realmName, role.Spec.Name); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Realm role not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to delete realm role %s: %w", role.Spec.Name, err)
	}

	log.Info("Realm role deleted successfully")

	return nil
}
