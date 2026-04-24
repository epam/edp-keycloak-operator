package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

// RemoveComponent deletes a realm component from Keycloak.
type RemoveComponent struct {
	kClient *keycloakapi.KeycloakClient
}

func NewRemoveComponent(kClient *keycloakapi.KeycloakClient) *RemoveComponent {
	return &RemoveComponent{kClient: kClient}
}

func (h *RemoveComponent) Serve(
	ctx context.Context,
	component *keycloakApi.KeycloakRealmComponent,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start removing realm component")

	if objectmeta.PreserveResourcesOnDeletion(component) {
		log.Info("Preserve resources on deletion, skipping")

		return nil
	}

	componentID := component.Status.ID

	if componentID == "" {
		existing, err := h.kClient.RealmComponents.FindComponentByName(ctx, realmName, component.Spec.Name)
		if err != nil {
			return fmt.Errorf("failed to find component for deletion: %w", err)
		}

		if existing == nil || existing.Id == nil {
			log.Info("Realm component not found, skipping deletion")

			return nil
		}

		componentID = *existing.Id
	}

	if _, err := h.kClient.RealmComponents.DeleteComponent(ctx, realmName, componentID); err != nil {
		if keycloakapi.IsNotFound(err) {
			log.Info("Realm component not found, skipping deletion")

			return nil
		}

		return fmt.Errorf("failed to delete realm component: %w", err)
	}

	log.Info("Realm component deleted successfully")

	return nil
}
