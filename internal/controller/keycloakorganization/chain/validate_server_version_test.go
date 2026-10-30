package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func serverInfo(version string) *keycloakapi.ServerInfo {
	return &keycloakapi.ServerInfo{SystemInfo: keycloakapi.SystemInfo{ServerVersion: version}}
}

func TestValidateServerVersion_ServeRequest(t *testing.T) {
	t.Parallel()

	const domain = "d.example"

	aliasDomain := []keycloakApi.OrgDomain{{Name: domain, IdentityProviderAlias: ptr.To(testIdPAlias)}}
	idps := []keycloakApi.OrgIdentityProvider{{Alias: testIdPAlias}}

	tests := []struct {
		name      string
		spec      keycloakApi.KeycloakOrganizationSpec
		version   string
		infoErr   error
		nilInfo   bool
		wantCall  bool
		wantError string
	}{
		{
			name:    "no gated field: no serverinfo call",
			spec:    keycloakApi.KeycloakOrganizationSpec{DomainsV2: []keycloakApi.OrgDomain{{Name: domain, Verified: ptr.To(true)}}, IdentityProviders: idps},
			version: "26.7.5",
		},
		{
			name:    "empty alias is not gated: no serverinfo call",
			spec:    keycloakApi.KeycloakOrganizationSpec{DomainsV2: []keycloakApi.OrgDomain{{Name: domain, IdentityProviderAlias: ptr.To("")}}},
			version: "26.7.5",
		},
		{
			name:    "autoRedirect false is not gated: no serverinfo call",
			spec:    keycloakApi.KeycloakOrganizationSpec{DomainsV2: []keycloakApi.OrgDomain{{Name: domain, AutoRedirect: ptr.To(false)}}},
			version: "26.7.5",
		},
		{
			name:      "alias on 26.7.5",
			spec:      keycloakApi.KeycloakOrganizationSpec{DomainsV2: aliasDomain, IdentityProviders: idps},
			version:   "26.7.5",
			wantCall:  true,
			wantError: `domainsV2[0] "d.example": identityProviderAlias requires Keycloak 26.8 or later, server version is 26.7.5`,
		},
		{
			name:      "autoRedirect true on 26.7.5",
			spec:      keycloakApi.KeycloakOrganizationSpec{DomainsV2: []keycloakApi.OrgDomain{{Name: domain, AutoRedirect: ptr.To(true)}}},
			version:   "26.7.5",
			wantCall:  true,
			wantError: `domainsV2[0] "d.example": autoRedirect requires Keycloak 26.8 or later, server version is 26.7.5`,
		},
		{
			name: "autoRedirect false with alias names the alias",
			spec: keycloakApi.KeycloakOrganizationSpec{
				DomainsV2:         []keycloakApi.OrgDomain{{Name: domain, IdentityProviderAlias: ptr.To(testIdPAlias), AutoRedirect: ptr.To(false)}},
				IdentityProviders: idps,
			},
			version:   "26.4.0",
			wantCall:  true,
			wantError: `domainsV2[0] "d.example": identityProviderAlias requires Keycloak 26.8 or later, server version is 26.4.0`,
		},
		{
			name: "second domain is the first gated one",
			spec: keycloakApi.KeycloakOrganizationSpec{
				DomainsV2: []keycloakApi.OrgDomain{{Name: "a.example"}, {Name: "b.example", AutoRedirect: ptr.To(true)}},
			},
			version:   "26.7.5",
			wantCall:  true,
			wantError: `domainsV2[1] "b.example": autoRedirect requires`,
		},
		{
			name:      "autoMembership false on 26.7.5",
			spec:      keycloakApi.KeycloakOrganizationSpec{IdentityProviders: []keycloakApi.OrgIdentityProvider{{Alias: testIdPAlias, AutoMembership: ptr.To(false)}}},
			version:   "26.7.5",
			wantCall:  true,
			wantError: `identityProviders[0] "idp1": autoMembership requires Keycloak 26.8 or later, server version is 26.7.5`,
		},
		{
			name:      "membershipType on 26.7.5",
			spec:      keycloakApi.KeycloakOrganizationSpec{IdentityProviders: []keycloakApi.OrgIdentityProvider{{Alias: testIdPAlias, MembershipType: ptr.To("UNMANAGED")}}},
			version:   "26.7.5",
			wantCall:  true,
			wantError: `identityProviders[0] "idp1": membershipType requires Keycloak 26.8 or later, server version is 26.7.5`,
		},
		{
			name: "domain field is named before link field",
			spec: keycloakApi.KeycloakOrganizationSpec{
				DomainsV2:         aliasDomain,
				IdentityProviders: []keycloakApi.OrgIdentityProvider{{Alias: testIdPAlias, AutoMembership: ptr.To(true)}},
			},
			version:   "26.7.5",
			wantCall:  true,
			wantError: `domainsV2[0] "d.example": identityProviderAlias requires`,
		},
		{name: "26.8.0 passes", spec: keycloakApi.KeycloakOrganizationSpec{DomainsV2: aliasDomain}, version: "26.8.0", wantCall: true},
		{name: "absent version skips the check", spec: keycloakApi.KeycloakOrganizationSpec{DomainsV2: aliasDomain}, version: "", wantCall: true},
		{name: "unparseable version skips the check", spec: keycloakApi.KeycloakOrganizationSpec{DomainsV2: aliasDomain}, version: "UNKNOWN", wantCall: true},
		{name: "nil serverinfo skips the check", spec: keycloakApi.KeycloakOrganizationSpec{DomainsV2: aliasDomain}, nilInfo: true, wantCall: true},
		{
			name:      "serverinfo error fails",
			spec:      keycloakApi.KeycloakOrganizationSpec{DomainsV2: aliasDomain},
			infoErr:   errors.New("connection refused"),
			wantCall:  true,
			wantError: `domainsV2[0] "d.example": identityProviderAlias: unable to read the Keycloak version to check the 26.8 requirement: connection refused`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := keycloakapimocks.NewMockServerInfoClient(t)

			if tt.wantCall {
				var info *keycloakapi.ServerInfo
				if tt.infoErr == nil && !tt.nilInfo {
					info = serverInfo(tt.version)
				}

				server.On("GetServerInfo", mock.Anything).Return(info, tt.infoErr).Once()
			}

			org := &keycloakApi.KeycloakOrganization{Spec: tt.spec}

			err := NewValidateServerVersion(&keycloakapi.KeycloakClient{Server: server}).
				ServeRequest(context.Background(), org, testRealmName)

			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}

			require.NoError(t, err)
		})
	}
}
