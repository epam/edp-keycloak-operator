package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

func NewRemoveOrganization(kc *keycloakv2.KeycloakClient) Handler {
	return &RemoveOrganization{
		keycloakClient: kc.Organizations,
	}
}

type RemoveOrganization struct {
	keycloakClient keycloakv2.OrganizationsClient
}

func (h *RemoveOrganization) ServeRequest(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realmName string) error {
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

	if _, err := h.keycloakClient.DeleteOrganization(ctx, realmName, organization.Status.OrganizationID); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Organization not found, skipping")

			return nil
		}

		return fmt.Errorf("failed to delete organization %s: %w", organization.Name, err)
	}

	log.Info("Organization deleted successfully")

	return nil
}
