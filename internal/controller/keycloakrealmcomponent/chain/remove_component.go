package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

// RemoveComponent deletes a realm component from Keycloak.
type RemoveComponent struct {
	k8sClient client.Client
	kClient   *keycloakapi.KeycloakClient
}

func NewRemoveComponent(k8sClient client.Client, kClient *keycloakapi.KeycloakClient) *RemoveComponent {
	return &RemoveComponent{k8sClient: k8sClient, kClient: kClient}
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
	foundByName := componentID == ""

	if foundByName {
		if ref := component.Spec.ParentRef; ref != nil && parentRefKind(ref) == keycloakApi.KeycloakRealmComponentKind {
			log.Info("Realm component has no ID in status and a component parent, skipping deletion")

			return nil
		}

		realm, err := resolveRealmParent(ctx, h.kClient, realmName)
		if err != nil {
			return fmt.Errorf("failed to resolve realm for deletion: %w", err)
		}

		existing, err := findComponentUnderParent(
			ctx, h.kClient, realmName, component.Spec.Name, realm.ID, component.Spec.ProviderType,
		)
		if err != nil {
			return fmt.Errorf("failed to find component for deletion: %w", err)
		}

		if existing == nil || existing.Id == nil {
			log.Info("Realm component not found, skipping deletion")

			return nil
		}

		componentID = *existing.Id
	}

	owner, err := findOwnerCR(ctx, h.k8sClient, component, componentID)
	if err != nil {
		return fmt.Errorf("unable to check realm component ownership for deletion: %w", err)
	}

	if owner != nil && foundByName {
		log.Info("Realm component is managed by another resource, skipping deletion",
			"owner", owner.Namespace+"/"+owner.Name, "componentID", componentID)

		return nil
	}

	if owner != nil {
		return fmt.Errorf(
			"realm component %q (id %s) is managed by KeycloakRealmComponent %s/%s; refusing to delete it - "+
				"remove the duplicate claim, or set annotation %s=true on KeycloakRealmComponent %s/%s "+
				"to remove this resource and keep the Keycloak component",
			component.Spec.Name, componentID, owner.Namespace, owner.Name,
			objectmeta.PreserveResourcesOnDeletionAnnotation, component.Namespace, component.Name,
		)
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
