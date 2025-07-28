package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type CreateOrganization struct {
	keycloakClient keycloak.Client
}

func NewCreateOrganization(keycloakClient keycloak.Client) *CreateOrganization {
	return &CreateOrganization{
		keycloakClient: keycloakClient,
	}
}

func (h *CreateOrganization) ServeRequest(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realm *gocloak.RealmRepresentation) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start creating/updating Keycloak organization")

	realmName := gocloak.PString(realm.Realm)
	orgAdapter := dto.ConvertSpecToOrganization(organization)

	// Check if organization already exists by alias
	existingOrg, err := h.keycloakClient.GetOrganizationByAlias(ctx, realmName, organization.Spec.Alias)
	if err != nil && !adapter.IsErrNotFound(err) {
		return fmt.Errorf("failed to check if organization exists with alias %s: %w", organization.Spec.Alias, err)
	}

	if err == nil && existingOrg != nil {
		// Organization exists, update it
		orgAdapter.ID = existingOrg.ID
		if updateErr := h.keycloakClient.UpdateOrganization(ctx, realmName, orgAdapter); updateErr != nil {
			return fmt.Errorf("unable to update organization: %w", updateErr)
		}

		organization.Status.OrganizationID = existingOrg.ID

		log.Info("Organization updated successfully", "organizationId", organization.Status.OrganizationID)

		return nil
	}

	// Organization doesn't exist, create new one
	if createErr := h.keycloakClient.CreateOrganization(ctx, realmName, orgAdapter); createErr != nil {
		return fmt.Errorf("unable to create organization: %w", createErr)
	}

	// Get the created organization by alias to retrieve its ID
	org, err := h.keycloakClient.GetOrganizationByAlias(ctx, realmName, organization.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to find created organization with alias %s: %w", organization.Spec.Alias, err)
	}

	organization.Status.OrganizationID = org.ID

	log.Info("Organization created successfully", "organizationId", organization.Status.OrganizationID)

	return nil
}
