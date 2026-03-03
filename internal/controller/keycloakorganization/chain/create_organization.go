package chain

import (
	"context"
	"fmt"
	"maps"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type CreateOrganization struct {
	keycloakClient keycloakv2.OrganizationsClient
}

func NewCreateOrganization(kc *keycloakv2.KeycloakClient) *CreateOrganization {
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
	if err != nil && !keycloakv2.IsNotFound(err) {
		return fmt.Errorf("failed to check if organization exists with alias %s: %w", organization.Spec.Alias, err)
	}

	if err == nil && existingOrg != nil {
		// Organization exists, update it
		orgRepresentation.Id = existingOrg.Id
		if _, updateErr := h.keycloakClient.UpdateOrganization(ctx, realmName, ptr.Deref(existingOrg.Id, ""), orgRepresentation); updateErr != nil {
			return fmt.Errorf("unable to update organization: %w", updateErr)
		}

		organization.Status.OrganizationID = ptr.Deref(existingOrg.Id, "")

		log.Info("Organization updated successfully", "organizationId", organization.Status.OrganizationID)

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
func specToOrganizationRepresentation(org *keycloakApi.KeycloakOrganization) keycloakv2.OrganizationRepresentation {
	rep := keycloakv2.OrganizationRepresentation{
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
		domains := make([]keycloakv2.OrganizationDomainRepresentation, 0, len(org.Spec.Domains))
		for _, d := range org.Spec.Domains {
			domains = append(domains, keycloakv2.OrganizationDomainRepresentation{
				Name: ptr.To(d),
			})
		}

		rep.Domains = &domains
	}

	return rep
}
