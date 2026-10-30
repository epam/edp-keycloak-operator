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

func TestProcessDomainSettings_ServeRequest(t *testing.T) {
	t.Parallel()

	fetchedOrg := func(domains ...keycloakapi.OrganizationDomainRepresentation) *keycloakapi.OrganizationRepresentation {
		return &keycloakapi.OrganizationRepresentation{
			Id:          ptr.To(testOrgID),
			Name:        ptr.To("Org"),
			Alias:       ptr.To("org"),
			Attributes:  &map[string][]string{"out-of-band": {"kept"}},
			Domains:     &domains,
			Description: ptr.To("d"),
		}
	}

	domain := func(name string, verified bool, alias string, autoRedirect bool) keycloakapi.OrganizationDomainRepresentation {
		d := keycloakapi.OrganizationDomainRepresentation{
			Name: ptr.To(name), Verified: ptr.To(verified), AutoRedirect: ptr.To(autoRedirect),
		}
		if alias != "" {
			d.IdentityProviderAlias = ptr.To(alias)
		}

		return d
	}

	tests := []struct {
		name      string
		domains   []keycloakApi.OrgDomain
		fetched   *keycloakapi.OrganizationRepresentation
		getErr    error
		wantGet   bool
		wantPut   []keycloakapi.OrganizationDomainRepresentation
		putErr    error
		wantError string
		orgID     string
	}{
		{
			name:    "no settings: no Keycloak call",
			domains: []keycloakApi.OrgDomain{{Name: "a.com"}},
			orgID:   testOrgID,
		},
		{
			name: "settings applied to the fetched representation",
			domains: []keycloakApi.OrgDomain{{
				Name: "a.com", Verified: ptr.To(true), IdentityProviderAlias: ptr.To(testIdPAlias), AutoRedirect: ptr.To(true),
			}},
			fetched: fetchedOrg(domain("a.com", false, "", false)),
			wantGet: true,
			wantPut: []keycloakapi.OrganizationDomainRepresentation{domain("a.com", true, testIdPAlias, true)},
			orgID:   testOrgID,
		},
		{
			name:    "settled: no PUT",
			domains: []keycloakApi.OrgDomain{{Name: "A.com", Verified: ptr.To(true), IdentityProviderAlias: ptr.To(testIdPAlias)}},
			fetched: fetchedOrg(domain("a.com", true, testIdPAlias, false)),
			wantGet: true,
			orgID:   testOrgID,
		},
		{
			name:    "empty alias clears the routing",
			domains: []keycloakApi.OrgDomain{{Name: "a.com", IdentityProviderAlias: ptr.To("")}},
			fetched: fetchedOrg(domain("a.com", true, testIdPAlias, true)),
			wantGet: true,
			wantPut: []keycloakapi.OrganizationDomainRepresentation{domain("a.com", true, "", true)},
			orgID:   testOrgID,
		},
		{
			name:    "omitted settings keep the Keycloak value",
			domains: []keycloakApi.OrgDomain{{Name: "a.com"}, {Name: "b.com", Verified: ptr.To(true)}},
			fetched: fetchedOrg(domain("a.com", true, testIdPAlias, true), domain("b.com", false, "", false)),
			wantGet: true,
			wantPut: []keycloakapi.OrganizationDomainRepresentation{
				domain("a.com", true, testIdPAlias, true), domain("b.com", true, "", false),
			},
			orgID: testOrgID,
		},
		{
			name:    "drift in a set field is repaired",
			domains: []keycloakApi.OrgDomain{{Name: "a.com", IdentityProviderAlias: ptr.To(testIdPAlias)}},
			fetched: fetchedOrg(domain("a.com", false, "", false)),
			wantGet: true,
			wantPut: []keycloakapi.OrganizationDomainRepresentation{domain("a.com", false, testIdPAlias, false)},
			orgID:   testOrgID,
		},
		{
			name: "identical duplicates are sent once",
			domains: []keycloakApi.OrgDomain{
				{Name: "a.com", Verified: ptr.To(true)}, {Name: "A.com", Verified: ptr.To(true)},
			},
			fetched: fetchedOrg(domain("a.com", false, "", false)),
			wantGet: true,
			wantPut: []keycloakapi.OrganizationDomainRepresentation{domain("a.com", true, "", false)},
			orgID:   testOrgID,
		},
		{
			name:    "autoRedirect false is omitted from the body",
			domains: []keycloakApi.OrgDomain{{Name: "a.com", AutoRedirect: ptr.To(false)}},
			fetched: fetchedOrg(domain("a.com", true, "", true)),
			wantGet: true,
			wantPut: []keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.com"), Verified: ptr.To(true)},
			},
			orgID: testOrgID,
		},
		{
			name:    "autoRedirect false matches a stored false: no PUT",
			domains: []keycloakApi.OrgDomain{{Name: "a.com", AutoRedirect: ptr.To(false)}},
			fetched: fetchedOrg(domain("a.com", true, "", false)),
			wantGet: true,
			orgID:   testOrgID,
		},
		{
			name:    "autoRedirect false with verified on a server without the key",
			domains: []keycloakApi.OrgDomain{{Name: "a.com", Verified: ptr.To(true), AutoRedirect: ptr.To(false)}},
			fetched: fetchedOrg(keycloakapi.OrganizationDomainRepresentation{Name: ptr.To("a.com")}),
			wantGet: true,
			wantPut: []keycloakapi.OrganizationDomainRepresentation{
				{Name: ptr.To("a.com"), Verified: ptr.To(true)},
			},
			orgID: testOrgID,
		},
		{
			name:      "GET fails",
			domains:   []keycloakApi.OrgDomain{{Name: "a.com", Verified: ptr.To(true)}},
			getErr:    errors.New("boom"),
			wantGet:   true,
			wantError: "unable to get organization org-123",
			orgID:     testOrgID,
		},
		{
			name:      "organization deleted in between",
			domains:   []keycloakApi.OrgDomain{{Name: "a.com", Verified: ptr.To(true)}},
			getErr:    &keycloakapi.ApiError{Code: 404, Message: "not found"},
			wantGet:   true,
			wantError: "unable to get organization org-123",
			orgID:     testOrgID,
		},
		{
			name:      "GET returns no representation",
			domains:   []keycloakApi.OrgDomain{{Name: "a.com", Verified: ptr.To(true)}},
			wantGet:   true,
			wantError: "Keycloak returned no representation",
			orgID:     testOrgID,
		},
		{
			name:      "PUT fails",
			domains:   []keycloakApi.OrgDomain{{Name: "a.com", Verified: ptr.To(true)}},
			fetched:   fetchedOrg(domain("a.com", false, "", false)),
			wantGet:   true,
			wantPut:   []keycloakapi.OrganizationDomainRepresentation{domain("a.com", true, "", false)},
			putErr:    errors.New("400"),
			wantError: "unable to update organization domain settings",
			orgID:     testOrgID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			orgs := keycloakapimocks.NewMockOrganizationsClient(t)

			if tt.wantGet {
				orgs.On("GetOrganization", mock.Anything, testRealmName, testOrgID).
					Return(tt.fetched, (*keycloakapi.Response)(nil), tt.getErr).Once()
			}

			if tt.wantPut != nil {
				orgs.On("UpdateOrganization", mock.Anything, testRealmName, testOrgID, mock.Anything).
					Run(func(args mock.Arguments) {
						sent, ok := args.Get(3).(keycloakapi.OrganizationRepresentation)
						require.True(t, ok)
						require.Equal(t, tt.wantPut, ptr.Deref(sent.Domains, nil))
						require.Equal(t, tt.fetched.Attributes, sent.Attributes)
						require.Equal(t, tt.fetched.Name, sent.Name)
					}).
					Return((*keycloakapi.Response)(nil), tt.putErr).Once()
			}

			kc := &keycloakapi.KeycloakClient{}
			kc.Organizations = orgs

			org := &keycloakApi.KeycloakOrganization{
				Spec:   keycloakApi.KeycloakOrganizationSpec{DomainsV2: tt.domains},
				Status: keycloakApi.KeycloakOrganizationStatus{OrganizationID: tt.orgID},
			}

			err := NewProcessDomainSettings(kc).ServeRequest(context.Background(), org, testRealmName)

			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGatedDomainField(t *testing.T) {
	t.Parallel()

	const domainName = "gate.example"

	tests := []struct {
		name   string
		domain keycloakApi.OrgDomain
		want   string
	}{
		{name: "alias set", domain: keycloakApi.OrgDomain{Name: domainName, IdentityProviderAlias: ptr.To("x")}, want: "identityProviderAlias"},
		{name: "alias empty", domain: keycloakApi.OrgDomain{Name: domainName, IdentityProviderAlias: ptr.To("")}},
		{name: "alias nil", domain: keycloakApi.OrgDomain{Name: domainName}},
		{name: "autoRedirect true", domain: keycloakApi.OrgDomain{Name: domainName, AutoRedirect: ptr.To(true)}, want: "autoRedirect"},
		{name: "autoRedirect false", domain: keycloakApi.OrgDomain{Name: domainName, AutoRedirect: ptr.To(false)}},
		{name: "autoRedirect nil", domain: keycloakApi.OrgDomain{Name: domainName}},
		{name: "verified only", domain: keycloakApi.OrgDomain{Name: domainName, Verified: ptr.To(false)}},
		{
			name:   "alias takes precedence over autoRedirect",
			domain: keycloakApi.OrgDomain{Name: domainName, IdentityProviderAlias: ptr.To("x"), AutoRedirect: ptr.To(true)},
			want:   "identityProviderAlias",
		},
		{
			name:   "autoRedirect false with alias names the alias",
			domain: keycloakApi.OrgDomain{Name: domainName, IdentityProviderAlias: ptr.To("x"), AutoRedirect: ptr.To(false)},
			want:   "identityProviderAlias",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, gatedDomainField(tt.domain))
		})
	}
}
