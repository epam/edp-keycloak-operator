package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

func TestValidateDomainSettings_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		domains []keycloakApi.OrgDomain
		idps    *[]keycloakApi.OrgIdentityProvider
		wantErr string
	}{
		{
			name:    "alias in identityProviders",
			domains: []keycloakApi.OrgDomain{{Name: "a.com", IdentityProviderAlias: ptr.To(testIdPAlias)}},
			idps:    ptr.To([]keycloakApi.OrgIdentityProvider{{Alias: testIdPAlias}}),
		},
		{
			name:    "alias with absent identityProviders",
			domains: []keycloakApi.OrgDomain{{Name: "a.com", IdentityProviderAlias: ptr.To(testIdPAlias)}},
			wantErr: `domainsV2[0] "a.com": identityProviderAlias "idp1" is not in spec.identityProviders`,
		},
		{
			name:    "alias with empty identityProviders",
			domains: []keycloakApi.OrgDomain{{Name: testDomain, IdentityProviderAlias: ptr.To(testIdPAlias)}},
			idps:    ptr.To([]keycloakApi.OrgIdentityProvider{}),
			wantErr: `domainsV2[0] "a.com": identityProviderAlias "idp1" is not in spec.identityProviders`,
		},
		{
			name:    "empty alias needs no provider",
			domains: []keycloakApi.OrgDomain{{Name: "a.com", IdentityProviderAlias: ptr.To("")}},
		},
		{
			name:    "alias missing in identityProviders",
			domains: []keycloakApi.OrgDomain{{Name: "x.com"}, {Name: "a.com", IdentityProviderAlias: ptr.To(testIdPAlias)}},
			idps:    ptr.To([]keycloakApi.OrgIdentityProvider{{Alias: "idp2"}}),
			wantErr: `domainsV2[1] "a.com": identityProviderAlias "idp1" is not in spec.identityProviders`,
		},
		{
			name: "duplicate with different settings",
			domains: []keycloakApi.OrgDomain{
				{Name: "a.com", Verified: ptr.To(true)},
				{Name: " A.com"},
			},
			wantErr: `domainsV2[1] " A.com": duplicate domain with different settings`,
		},
		{
			name: "duplicate with identical settings",
			domains: []keycloakApi.OrgDomain{
				{Name: "a.com", Verified: ptr.To(true)},
				{Name: "A.com", Verified: ptr.To(true)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			org := &keycloakApi.KeycloakOrganization{Spec: keycloakApi.KeycloakOrganizationSpec{
				DomainsV2: tt.domains, IdentityProviders: tt.idps,
			}}

			err := NewValidateDomainSettings().ServeRequest(context.Background(), org, testRealmName)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
