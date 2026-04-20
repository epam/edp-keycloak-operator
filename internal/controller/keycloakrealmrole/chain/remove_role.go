package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

type RemoveRole struct {
	kClient *keycloakapi.KeycloakClient
}

func NewRemoveRole(kClient *keycloakapi.KeycloakClient) *RemoveRole {
	return &RemoveRole{kClient: kClient}
}

func (h *RemoveRole) ServeRequest(ctx context.Context, role *keycloakApi.KeycloakRealmRole, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start removing realm role")

	if objectmeta.PreserveResourcesOnDeletion(role) {
		log.Info("Preserve resources on deletion, skipping")

		return nil
	}

	if _, err := h.kClient.Roles.DeleteRealmRole(ctx, realmName, role.Spec.Name); err != nil {
		if keycloakapi.IsNotFound(err) {
			log.Info("Realm role not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to delete realm role %s: %w", role.Spec.Name, err)
	}

	log.Info("Realm role deleted successfully")

	return nil
}
