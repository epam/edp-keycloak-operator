package chain

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

type CreateOrganization struct {
	keycloakClient keycloakapi.OrganizationsClient
}

func NewCreateOrganization(kc *keycloakapi.KeycloakClient) *CreateOrganization {
	return &CreateOrganization{
		keycloakClient: kc.Organizations,
	}
}

func (h *CreateOrganization) ServeRequest(ctx context.Context, organization *keycloakApi.KeycloakOrganization, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start creating/updating Keycloak organization")

	orgRepresentation := specToOrganizationRepresentation(organization)

	// Check if organization already exists by alias
	existingOrg, _, err := h.keycloakClient.GetOrganizationByAlias(ctx, realmName, organization.Spec.Alias)
	if err != nil && !keycloakapi.IsNotFound(err) {
		return fmt.Errorf("failed to check if organization exists with alias %s: %w", organization.Spec.Alias, err)
	}

	if err == nil && existingOrg != nil {
		// Organization exists, update it
		orgRepresentation.Id = existingOrg.Id
		organization.Status.OrganizationID = ptr.Deref(existingOrg.Id, "")

		needsUpdate, matchErr := h.orgNeedsUpdate(ctx, realmName, existingOrg, organization)
		if matchErr != nil {
			return matchErr
		}

		if needsUpdate {
			if _, updateErr := h.keycloakClient.UpdateOrganization(ctx, realmName, ptr.Deref(existingOrg.Id, ""), orgRepresentation); updateErr != nil {
				return fmt.Errorf("unable to update organization: %w", updateErr)
			}

			log.Info("Organization updated successfully", "organizationId", organization.Status.OrganizationID)
		}

		return nil
	}

	// Organization doesn't exist, create new one
	if _, createErr := h.keycloakClient.CreateOrganization(ctx, realmName, orgRepresentation); createErr != nil {
		return fmt.Errorf("unable to create organization: %w", createErr)
	}

	// Get the created organization by alias to retrieve its ID
	org, _, err := h.keycloakClient.GetOrganizationByAlias(ctx, realmName, organization.Spec.Alias)
	if err != nil {
		return fmt.Errorf("failed to find created organization with alias %s: %w", organization.Spec.Alias, err)
	}

	organization.Status.OrganizationID = ptr.Deref(org.Id, "")

	log.Info("Organization created successfully", "organizationId", organization.Status.OrganizationID)

	return nil
}

// specToOrganizationRepresentation converts a KeycloakOrganization spec to an OrganizationRepresentation.
func specToOrganizationRepresentation(org *keycloakApi.KeycloakOrganization) keycloakapi.OrganizationRepresentation {
	rep := keycloakapi.OrganizationRepresentation{
		Name:        ptr.To(org.Spec.Name),
		Alias:       ptr.To(org.Spec.Alias),
		Description: ptr.To(org.Spec.Description),
		RedirectUrl: ptr.To(org.Spec.RedirectURL),
	}

	if len(org.Spec.Attributes) > 0 {
		attrs := make(map[string][]string, len(org.Spec.Attributes))
		maps.Copy(attrs, org.Spec.Attributes)

		rep.Attributes = &attrs
	}

	if len(org.Spec.Domains) > 0 {
		domains := make([]keycloakapi.OrganizationDomainRepresentation, 0, len(org.Spec.Domains))
		for _, d := range org.Spec.Domains {
			domains = append(domains, keycloakapi.OrganizationDomainRepresentation{
				Name: ptr.To(d),
			})
		}

		rep.Domains = &domains
	}

	return rep
}

// orgNeedsUpdate reports whether organization must be written to Keycloak. A spec edit
// (generation bump) always forces the write. Otherwise the brief fields already available from
// existing (the list representation backing GetOrganizationByAlias) are compared first; only
// when those match and the spec declares attributes does it fetch the full representation to
// compare attributes, since the list endpoint omits them.
func (h *CreateOrganization) orgNeedsUpdate(
	ctx context.Context,
	realmName string,
	existing *keycloakapi.OrganizationRepresentation,
	organization *keycloakApi.KeycloakOrganization,
) (bool, error) {
	if helper.SpecChanged(organization.Generation, organization.Status.ObservedGeneration) {
		return true, nil
	}

	if !briefOrgMatchesSpec(existing, organization) {
		return true, nil
	}

	if len(organization.Spec.Attributes) == 0 {
		return false, nil
	}

	orgID := ptr.Deref(existing.Id, "")

	// The list endpoint returns a brief representation without attributes; only the by-ID GET
	// includes them.
	full, _, err := h.keycloakClient.GetOrganization(ctx, realmName, orgID)
	if err != nil {
		return false, fmt.Errorf("failed to get organization %s by id: %w", orgID, err)
	}

	return !maputil.ContainsSubsetMulti(ptr.Deref(full.Attributes, nil), organization.Spec.Attributes), nil
}

// briefOrgMatchesSpec reports whether the fields present in the brief (list) representation
// already match the desired spec. Enabled, Groups and Members are server-populated;
// IdentityProviders is reconciled separately by ProcessIdentityProviders.
func briefOrgMatchesSpec(existing *keycloakapi.OrganizationRepresentation, org *keycloakApi.KeycloakOrganization) bool {
	if ptr.Deref(existing.Name, "") != org.Spec.Name ||
		ptr.Deref(existing.Alias, "") != org.Spec.Alias ||
		ptr.Deref(existing.Description, "") != org.Spec.Description ||
		ptr.Deref(existing.RedirectUrl, "") != org.Spec.RedirectURL {
		return false
	}

	existingDomains := ptr.Deref(existing.Domains, nil)
	existingDomainNames := make([]string, 0, len(existingDomains))

	for _, d := range existingDomains {
		existingDomainNames = append(existingDomainNames, ptr.Deref(d.Name, ""))
	}

	// Keycloak dedups domains server-side: dedup the spec list too so accidental spec
	// duplicates don't permanently look like drift.
	specDomains := slices.Clone(org.Spec.Domains)
	slices.Sort(specDomains)
	specDomains = slices.Compact(specDomains)

	slices.Sort(existingDomainNames)

	return slices.Equal(existingDomainNames, specDomains)
}
