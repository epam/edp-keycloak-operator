package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type ProcessIdentityProviders struct {
	keycloakClient keycloakapi.OrganizationsClient
}

func NewProcessIdentityProviders(kc *keycloakapi.KeycloakClient) *ProcessIdentityProviders {
	return &ProcessIdentityProviders{
		keycloakClient: kc.Organizations,
	}
}

func (h *ProcessIdentityProviders) ServeRequest(
	ctx context.Context,
	organization *keycloakApi.KeycloakOrganization,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)

	if organization.Status.OrganizationID == "" {
		return fmt.Errorf("organization ID is not set in status")
	}

	log.Info("Start processing identity providers for organization")

	// Get current identity providers linked to the organization
	currentIdPs, _, err := h.keycloakClient.GetOrganizationIdentityProviders(ctx, realmName, organization.Status.OrganizationID)
	if err != nil {
		return fmt.Errorf("unable to get current organization identity providers: %w", err)
	}

	// Build a set of identity providers defined in the spec.
	specIdPMap := make(map[string]bool, len(organization.Spec.IdentityProviders))
	for _, specIdP := range organization.Spec.IdentityProviders {
		specIdPMap[specIdP.Alias] = true
	}

	// Build a set of currently linked identity providers and unlink any that are no longer in the spec.
	currentIdPMap := make(map[string]bool, len(currentIdPs))

	for _, idp := range currentIdPs {
		if idp.Alias == nil {
			continue
		}

		currentIdPMap[*idp.Alias] = true

		if !specIdPMap[*idp.Alias] {
			if _, err := h.keycloakClient.UnlinkIdentityProviderFromOrganization(ctx, realmName, organization.Status.OrganizationID, *idp.Alias); err != nil {
				return fmt.Errorf("unable to unlink identity provider %s from organization: %w", *idp.Alias, err)
			}

			log.Info("Identity provider unlinked from organization", "alias", *idp.Alias)
		}
	}

	// Link identity providers from the spec that are not yet linked.
	for _, specIdP := range organization.Spec.IdentityProviders {
		if currentIdPMap[specIdP.Alias] {
			log.Info("Identity provider already linked to organization", "alias", specIdP.Alias)
			continue
		}

		if _, err := h.keycloakClient.LinkIdentityProviderToOrganization(ctx, realmName, organization.Status.OrganizationID, specIdP.Alias); err != nil {
			return fmt.Errorf("unable to link identity provider %s to organization: %w", specIdP.Alias, err)
		}

		log.Info("Identity provider linked to organization successfully", "alias", specIdP.Alias)
	}

	log.Info("Processing identity providers completed successfully")

	return nil
}
