package chain

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// ValidateDomainSettings rejects an invalid spec before any handler writes to Keycloak.
type ValidateDomainSettings struct{}

func NewValidateDomainSettings() *ValidateDomainSettings {
	return &ValidateDomainSettings{}
}

func (h *ValidateDomainSettings) ServeRequest(
	_ context.Context,
	organization *keycloakApi.KeycloakOrganization,
	_ string,
) error {
	aliases := make([]string, 0, len(organization.Spec.IdentityProviders))
	for _, idp := range organization.Spec.IdentityProviders {
		aliases = append(aliases, idp.Alias)
	}

	seen := make(map[string]keycloakApi.OrgDomain, len(organization.Spec.DomainsV2))

	for i, d := range organization.Spec.DomainsV2 {
		if alias := ptr.Deref(d.IdentityProviderAlias, ""); alias != "" && !slices.Contains(aliases, alias) {
			return fmt.Errorf("domainsV2[%d] %q: identityProviderAlias %q is not in spec.identityProviders",
				i, d.Name, alias)
		}

		key := keycloakapi.OrganizationDomainName(d.Name)

		if prev, ok := seen[key]; ok {
			if !domainSettingsEqual(prev, d) {
				return fmt.Errorf("domainsV2[%d] %q: duplicate domain with different settings", i, d.Name)
			}

			continue
		}

		seen[key] = d
	}

	return nil
}

func domainSettingsEqual(a, b keycloakApi.OrgDomain) bool {
	return ptr.Equal(a.Verified, b.Verified) &&
		ptr.Equal(a.IdentityProviderAlias, b.IdentityProviderAlias) &&
		ptr.Equal(a.AutoRedirect, b.AutoRedirect)
}
