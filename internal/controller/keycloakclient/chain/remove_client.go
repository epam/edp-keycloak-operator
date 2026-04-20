package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

func NewRemoveClient(kc *keycloakapi.KeycloakClient) *RemoveClient {
	return &RemoveClient{
		keycloakClient: kc.Clients,
	}
}

type RemoveClient struct {
	keycloakClient keycloakapi.ClientsClient
}

func (h *RemoveClient) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	log := ctrl.LoggerFrom(ctx).WithValues("client_id", keycloakClient.Status.ClientID)

	log.Info("Start deleting keycloak client")

	if objectmeta.PreserveResourcesOnDeletion(keycloakClient) {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")

		return nil
	}

	if keycloakClient.Status.ClientID == "" {
		log.Info("Client ID is not set in status, skipping deletion.")

		return nil
	}

	if _, err := h.keycloakClient.DeleteClient(ctx, realmName, keycloakClient.Status.ClientID); err != nil {
		if keycloakapi.IsNotFound(err) {
			log.Info("Client not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete keycloak client: %w", err)
	}

	log.Info("Keycloak client has been deleted")

	return nil
}
