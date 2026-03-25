package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

type RemoveScope struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewRemoveScope(kClientV2 *keycloakv2.KeycloakClient) *RemoveScope {
	return &RemoveScope{kClientV2: kClientV2}
}

func (h *RemoveScope) Serve(ctx context.Context, scope *keycloakApi.KeycloakClientScope, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start removing client scope")

	if objectmeta.PreserveResourcesOnDeletion(scope) {
		log.Info("Preserve resources on deletion, skipping")

		return nil
	}

	scopesClient := h.kClientV2.ClientScopes
	scopeID := scope.Status.ID

	// Remove from realm scope lists before deletion
	if _, err := scopesClient.RemoveRealmDefaultClientScope(ctx, realmName, scopeID); err != nil && !keycloakv2.IsNotFound(err) {
		return fmt.Errorf("failed to remove scope from default list: %w", err)
	}

	if _, err := scopesClient.RemoveRealmOptionalClientScope(ctx, realmName, scopeID); err != nil && !keycloakv2.IsNotFound(err) {
		return fmt.Errorf("failed to remove scope from optional list: %w", err)
	}

	if _, err := scopesClient.DeleteClientScope(ctx, realmName, scopeID); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Client scope not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to delete client scope: %w", err)
	}

	log.Info("Client scope deleted successfully")

	return nil
}
