package chain

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

// ProcessDomainSettings writes verified, identityProviderAlias and autoRedirect of the spec domains.
// Runs after ProcessIdentityProviders.
type ProcessDomainSettings struct {
	keycloakClient keycloakapi.OrganizationsClient
}

func NewProcessDomainSettings(kc *keycloakapi.KeycloakClient) *ProcessDomainSettings {
	return &ProcessDomainSettings{
		keycloakClient: kc.Organizations,
	}
}

func (h *ProcessDomainSettings) ServeRequest(
	ctx context.Context,
	organization *keycloakApi.KeycloakOrganization,
	realmName string,
) error {
	if !slices.ContainsFunc(organization.Spec.DomainsV2, hasDomainSettings) {
		return nil
	}

	orgID := organization.Status.OrganizationID
	if orgID == "" {
		return fmt.Errorf("organization ID is not set in status")
	}

	fetched, _, err := h.keycloakClient.GetOrganization(ctx, realmName, orgID)
	if err != nil {
		return fmt.Errorf("unable to get organization %s: %w", orgID, err)
	}

	if fetched == nil {
		return fmt.Errorf("organization %s: Keycloak returned no representation", orgID)
	}

	fetchedDomains := ptr.Deref(fetched.Domains, nil)
	stored := maputil.SliceToMapSelf(fetchedDomains, domainKey)

	desired := make([]keycloakapi.OrganizationDomainRepresentation, 0, len(organization.Spec.DomainsV2))
	added := make(map[string]bool, len(organization.Spec.DomainsV2))

	for _, specDomain := range organization.Spec.DomainsV2 {
		key := keycloakapi.OrganizationDomainName(specDomain.Name)
		if added[key] {
			continue
		}

		added[key] = true

		d, ok := stored[key]
		if !ok {
			d = keycloakapi.OrganizationDomainRepresentation{Name: ptr.To(specDomain.Name)}
		}

		desired = append(desired, mergeDomainSettings(d, specDomain))
	}

	if domainsEqual(desired, fetchedDomains) {
		return nil
	}

	body := *fetched
	body.Domains = &desired

	if _, err := h.keycloakClient.UpdateOrganization(ctx, realmName, orgID, body); err != nil {
		return fmt.Errorf("unable to update organization domain settings: %w", err)
	}

	ctrl.LoggerFrom(ctx).Info("Organization domain settings updated", "organizationId", orgID)

	return nil
}

func hasDomainSettings(d keycloakApi.OrgDomain) bool {
	return d.Verified != nil || d.IdentityProviderAlias != nil || d.AutoRedirect != nil
}

func mergeDomainSettings(
	d keycloakapi.OrganizationDomainRepresentation,
	spec keycloakApi.OrgDomain,
) keycloakapi.OrganizationDomainRepresentation {
	if spec.Verified != nil {
		d.Verified = spec.Verified
	}

	if spec.IdentityProviderAlias != nil {
		d.IdentityProviderAlias = nil
		if *spec.IdentityProviderAlias != "" {
			d.IdentityProviderAlias = spec.IdentityProviderAlias
		}
	}

	if spec.AutoRedirect != nil {
		d.AutoRedirect = spec.AutoRedirect
	}

	return d
}

// domainsEqual compares domains by normalized name; nil flags compare as false and nil alias as "".
func domainsEqual(desired, fetched []keycloakapi.OrganizationDomainRepresentation) bool {
	if len(desired) != len(fetched) {
		return false
	}

	byName := maputil.SliceToMapSelf(fetched, domainKey)

	for _, d := range desired {
		f, ok := byName[keycloakapi.OrganizationDomainName(ptr.Deref(d.Name, ""))]
		if !ok ||
			ptr.Deref(d.Verified, false) != ptr.Deref(f.Verified, false) ||
			ptr.Deref(d.IdentityProviderAlias, "") != ptr.Deref(f.IdentityProviderAlias, "") ||
			ptr.Deref(d.AutoRedirect, false) != ptr.Deref(f.AutoRedirect, false) {
			return false
		}
	}

	return true
}
