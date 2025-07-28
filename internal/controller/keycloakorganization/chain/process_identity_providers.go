package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type ProcessIdentityProviders struct {
	keycloakClient keycloak.Client
}

func NewProcessIdentityProviders(keycloakClient keycloak.Client) *ProcessIdentityProviders {
	return &ProcessIdentityProviders{
		keycloakClient: keycloakClient,
	}
}

func (h *ProcessIdentityProviders) ServeRequest(
	ctx context.Context,
	organization *keycloakApi.KeycloakOrganization,
	realm *gocloak.RealmRepresentation,
) error {
	log := ctrl.LoggerFrom(ctx)

	if organization.Status.OrganizationID == "" {
		return fmt.Errorf("organization ID is not set in status")
	}

	log.Info("Start processing identity providers for organization")

	realmName := gocloak.PString(realm.Realm)

	// Get current identity providers linked to the organization
	currentIdPs, err := h.keycloakClient.GetOrganizationIdentityProviders(ctx, realmName, organization.Status.OrganizationID)
	if err != nil {
		return fmt.Errorf("unable to get current organization identity providers: %w", err)
	}

	// Create a map of current identity providers for easier lookup
	currentIdPMap := make(map[string]bool)
	for _, idp := range currentIdPs {
		currentIdPMap[idp.Alias] = true
	}

	// Process each identity provider in the spec
	for _, specIdP := range organization.Spec.IdentityProviders {
		if currentIdPMap[specIdP.Alias] {
			log.Info("Identity provider already linked to organization", "alias", specIdP.Alias)
			continue
		}

		// Link the identity provider to the organization
		if err := h.keycloakClient.LinkIdentityProviderToOrganization(ctx, realmName, organization.Status.OrganizationID, specIdP.Alias); err != nil {
			return fmt.Errorf("unable to link identity provider %s to organization: %w", specIdP.Alias, err)
		}

		log.Info("Identity provider linked to organization successfully", "alias", specIdP.Alias)
	}

	// Unlink identity providers that are no longer in the spec
	specIdPMap := make(map[string]bool, len(organization.Spec.IdentityProviders))
	for _, specIdP := range organization.Spec.IdentityProviders {
		specIdPMap[specIdP.Alias] = true
	}

	for _, currentIdP := range currentIdPs {
		if !specIdPMap[currentIdP.Alias] {
			if err := h.keycloakClient.UnlinkIdentityProviderFromOrganization(ctx, realmName, organization.Status.OrganizationID, currentIdP.Alias); err != nil {
				return fmt.Errorf("unable to unlink identity provider %s from organization: %w", currentIdP.Alias, err)
			}

			log.Info("Identity provider unlinked from organization", "alias", currentIdP.Alias)
		}
	}

	log.Info("Processing identity providers completed successfully")

	return nil
}
