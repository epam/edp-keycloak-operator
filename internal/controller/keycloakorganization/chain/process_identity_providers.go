package chain

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type ProcessIdentityProviders struct {
	keycloakClient keycloakapi.OrganizationsClient
	idpClient      keycloakapi.IdentityProvidersClient
}

func NewProcessIdentityProviders(kc *keycloakapi.KeycloakClient) *ProcessIdentityProviders {
	return &ProcessIdentityProviders{
		keycloakClient: kc.Organizations,
		idpClient:      kc.IdentityProviders,
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

	if organization.Spec.IdentityProviders == nil {
		log.Info("Identity providers are not managed by the resource, skipping")

		return nil
	}

	specIdPs := *organization.Spec.IdentityProviders

	// Get current identity providers linked to the organization
	currentIdPs, _, err := h.keycloakClient.GetOrganizationIdentityProviders(ctx, realmName, organization.Status.OrganizationID)
	if err != nil {
		return fmt.Errorf("unable to get current organization identity providers: %w", err)
	}

	// Build a set of identity providers defined in the spec.
	specIdPMap := make(map[string]bool, len(specIdPs))
	for _, specIdP := range specIdPs {
		specIdPMap[specIdP.Alias] = true
	}

	currentIdPMap := make(map[string]bool, len(currentIdPs))

	for _, idp := range currentIdPs {
		if idp.Alias != nil {
			currentIdPMap[*idp.Alias] = true
		}
	}

	for _, specIdP := range specIdPs {
		if currentIdPMap[specIdP.Alias] {
			continue
		}

		if err := h.requireIdentityProvider(ctx, realmName, specIdP.Alias); err != nil {
			return err
		}
	}

	for _, idp := range currentIdPs {
		if idp.Alias == nil || specIdPMap[*idp.Alias] {
			continue
		}

		if _, err := h.keycloakClient.UnlinkIdentityProviderFromOrganization(ctx, realmName, organization.Status.OrganizationID, *idp.Alias); err != nil {
			return fmt.Errorf("unable to unlink identity provider %s from organization: %w", *idp.Alias, err)
		}

		log.Info("Identity provider unlinked from organization", "alias", *idp.Alias)
	}

	// Link identity providers from the spec that are not yet linked.
	linkedWithSettings := false

	for _, specIdP := range specIdPs {
		if currentIdPMap[specIdP.Alias] {
			log.Info("Identity provider already linked to organization", "alias", specIdP.Alias)
			continue
		}

		if _, err := h.keycloakClient.LinkIdentityProviderToOrganization(ctx, realmName, organization.Status.OrganizationID, specIdP.Alias); err != nil {
			return fmt.Errorf("unable to link identity provider %s to organization: %w", specIdP.Alias, err)
		}

		linkedWithSettings = linkedWithSettings || hasLinkSettings(specIdP)

		log.Info("Identity provider linked to organization successfully", "alias", specIdP.Alias)
	}

	if err := h.syncLinkSettings(ctx, organization, specIdPs, realmName, currentIdPs, linkedWithSettings); err != nil {
		return err
	}

	log.Info("Processing identity providers completed successfully")

	return nil
}

func (h *ProcessIdentityProviders) requireIdentityProvider(ctx context.Context, realmName, alias string) error {
	if _, _, err := h.idpClient.GetIdentityProvider(ctx, realmName, alias); err != nil {
		if keycloakapi.IsNotFound(err) {
			return fmt.Errorf(
				"identity provider %q not found in realm %q; create it with a KeycloakRealmIdentityProvider: %w",
				alias, realmName, err)
		}

		return fmt.Errorf("unable to get identity provider %q: %w", alias, err)
	}

	return nil
}

func hasLinkSettings(idp keycloakApi.OrgIdentityProvider) bool {
	return idp.AutoMembership != nil || idp.MembershipType != nil
}

// syncLinkSettings writes autoMembership and membershipType of links whose spec sets either field.
// The PUT carries both fields.
func (h *ProcessIdentityProviders) syncLinkSettings(
	ctx context.Context,
	organization *keycloakApi.KeycloakOrganization,
	specIdPs []keycloakApi.OrgIdentityProvider,
	realmName string,
	currentIdPs []keycloakapi.IdentityProviderRepresentation,
	reread bool,
) error {
	if !slices.ContainsFunc(specIdPs, hasLinkSettings) {
		return nil
	}

	orgID := organization.Status.OrganizationID

	if reread {
		var err error

		currentIdPs, _, err = h.keycloakClient.GetOrganizationIdentityProviders(ctx, realmName, orgID)
		if err != nil {
			return fmt.Errorf("unable to get current organization identity providers: %w", err)
		}
	}

	fetchedLinks := make(map[string]keycloakapi.OrganizationIdentityProviderLinkRepresentation, len(currentIdPs))

	for _, idp := range currentIdPs {
		if idp.Alias == nil || idp.OrganizationLinks == nil {
			continue
		}

		for _, link := range *idp.OrganizationLinks {
			if ptr.Deref(link.OrganizationId, "") == orgID {
				fetchedLinks[*idp.Alias] = link
			}
		}
	}

	for _, specIdP := range specIdPs {
		if !hasLinkSettings(specIdP) {
			continue
		}

		fetched, ok := fetchedLinks[specIdP.Alias]
		if !ok {
			return fmt.Errorf(
				"identity provider %q: Keycloak returned no organization link settings; "+
					"autoMembership and membershipType require Keycloak 26.8 or later", specIdP.Alias)
		}

		desired := keycloakapi.OrganizationIdentityProviderLinkRepresentation{
			AutoMembership: fetched.AutoMembership,
			MembershipType: fetched.MembershipType,
		}

		if specIdP.AutoMembership != nil {
			desired.AutoMembership = specIdP.AutoMembership
		}

		if specIdP.MembershipType != nil {
			desired.MembershipType = specIdP.MembershipType
		}

		if ptr.Equal(desired.AutoMembership, fetched.AutoMembership) &&
			ptr.Equal(desired.MembershipType, fetched.MembershipType) {
			continue
		}

		if _, err := h.keycloakClient.UpdateOrganizationIdentityProviderLink(
			ctx, realmName, orgID, specIdP.Alias, desired); err != nil {
			return fmt.Errorf("unable to update link settings of identity provider %q: %w", specIdP.Alias, err)
		}

		ctrl.LoggerFrom(ctx).Info("Identity provider link settings updated", "alias", specIdP.Alias)
	}

	return nil
}
