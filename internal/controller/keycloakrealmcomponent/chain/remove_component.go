package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

// RemoveComponent deletes a realm component from Keycloak.
type RemoveComponent struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewRemoveComponent(kClientV2 *keycloakv2.KeycloakClient) *RemoveComponent {
	return &RemoveComponent{kClientV2: kClientV2}
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
		existing, err := h.kClientV2.RealmComponents.FindComponentByName(ctx, realmName, component.Spec.Name)
		if err != nil {
			return fmt.Errorf("failed to find component for deletion: %w", err)
		}

		if existing == nil || existing.Id == nil {
			log.Info("Realm component not found, skipping deletion")

			return nil
		}

		componentID = *existing.Id
	}

	if _, err := h.kClientV2.RealmComponents.DeleteComponent(ctx, realmName, componentID); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Realm component not found, skipping deletion")

			return nil
		}

		return fmt.Errorf("failed to delete realm component: %w", err)
	}

	log.Info("Realm component deleted successfully")

	return nil
}
