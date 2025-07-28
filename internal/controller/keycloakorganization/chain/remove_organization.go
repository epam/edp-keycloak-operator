package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

func NewRemoveOrganization(keycloakClient keycloak.Client) Handler {
	return &RemoveOrganization{
		keycloakClient: keycloakClient,
	}
}

type RemoveOrganization struct {
	keycloakClient keycloak.Client
}

func (h *RemoveOrganization) ServeRequest(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realm *gocloak.RealmRepresentation) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start removing organization")

	if organization.Status.OrganizationID == "" {
		log.Info("Organization ID is not set in status, skipping")

		return nil
	}

	if objectmeta.PreserveResourcesOnDeletion(organization) {
		log.Info("Preserve resources on deletion, skipping")

		return nil
	}

	realmName := gocloak.PString(realm.Realm)
	if err := h.keycloakClient.DeleteOrganization(ctx, realmName, organization.Status.OrganizationID); err != nil {
		if adapter.IsErrNotFound(err) {
			log.Info("Organization not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to delete organization %s: %w", organization.Name, err)
	}

	log.Info("Organization deleted successfully")

	return nil
}
