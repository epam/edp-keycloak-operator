package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

type RemoveIDP struct {
	kClient *keycloakv2.KeycloakClient
}

func NewRemoveIDP(kClient *keycloakv2.KeycloakClient) *RemoveIDP {
	return &RemoveIDP{kClient: kClient}
}

func (h *RemoveIDP) Serve(ctx context.Context, idp *keycloakApi.KeycloakRealmIdentityProvider, realmName string) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start removing identity provider")

	if objectmeta.PreserveResourcesOnDeletion(idp) {
		log.Info("Preserve resources on deletion, skipping")

		return nil
	}

	if _, err := h.kClient.IdentityProviders.DeleteIdentityProvider(ctx, realmName, idp.Spec.Alias); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Identity provider not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to delete identity provider: %w", err)
	}

	log.Info("Identity provider deleted successfully")

	return nil
}
