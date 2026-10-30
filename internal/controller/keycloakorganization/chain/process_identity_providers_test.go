package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestProcessIdentityProviders_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organization   *keycloakApi.KeycloakOrganization
		realmName      string
		keycloakClient func(t *testing.T) keycloakapi.OrganizationsClient
		idpClient      func(t *testing.T) keycloakapi.IdentityProvidersClient
		wantErr        require.ErrorAssertionFunc
		wantErrText    string
	}{
		{
			name: "success - link new identity providers",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{
						{Alias: testIdPAlias},
						{Alias: "idp2"},
					}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: testOrgID,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, testOrgID, testIdPAlias).
					Return((*keycloakapi.Response)(nil), nil).Once()
				client.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, testOrgID, "idp2").
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, testIdPAlias).
					Return(&keycloakapi.IdentityProviderRepresentation{Alias: ptr.To(testIdPAlias)}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("GetIdentityProvider", mock.Anything, testRealmName, "idp2").
					Return(&keycloakapi.IdentityProviderRepresentation{Alias: ptr.To("idp2")}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "success - unlink removed identity providers",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{
						{Alias: testIdPAlias},
					}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: testOrgID,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To(testIdPAlias)},
						{Alias: ptr.To("idp2")},
						{Alias: ptr.To("idp3")},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, testOrgID, "idp2").
					Return((*keycloakapi.Response)(nil), nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, testOrgID, "idp3").
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "success - mixed scenario: link new and unlink removed",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{
						{Alias: testIdPAlias},
						{Alias: "idp3"},
					}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: testOrgID,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To(testIdPAlias)},
						{Alias: ptr.To("idp2")},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, testOrgID, "idp3").
					Return((*keycloakapi.Response)(nil), nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, testOrgID, "idp2").
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, "idp3").
					Return(&keycloakapi.IdentityProviderRepresentation{Alias: ptr.To("idp3")}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "success - no changes needed",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{
						{Alias: testIdPAlias},
						{Alias: "idp2"},
					}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: testOrgID,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To(testIdPAlias)},
						{Alias: ptr.To("idp2")},
					}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "error - organization ID not set",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{
						{Alias: testIdPAlias},
					}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				return keycloakapimocks.NewMockOrganizationsClient(t)
			},
			wantErr: require.Error,
		},
		{
			name: "error - GetOrganizationIdentityProviders fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{
						{Alias: testIdPAlias},
					}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: testOrgID,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return(nil, (*keycloakapi.Response)(nil), errors.New("keycloak connection failed")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error - LinkIdentityProviderToOrganization fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{
						{Alias: testIdPAlias},
					}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: testOrgID,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, testOrgID, testIdPAlias).
					Return((*keycloakapi.Response)(nil), errors.New("failed to link identity provider")).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, testIdPAlias).
					Return(&keycloakapi.IdentityProviderRepresentation{Alias: ptr.To(testIdPAlias)}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error - UnlinkIdentityProviderFromOrganization fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{
						{Alias: testIdPAlias},
					}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: testOrgID,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To(testIdPAlias)},
						{Alias: ptr.To("idp2")},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, testOrgID, "idp2").
					Return((*keycloakapi.Response)(nil), errors.New("failed to unlink identity provider")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error - identity provider missing in realm",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{{Alias: testIdPAlias}}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{OrganizationID: testOrgID},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, testIdPAlias).
					Return(nil, (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: http.StatusNotFound, Message: "not found"}).Once()

				return client
			},
			wantErr:     require.Error,
			wantErrText: `identity provider "idp1" not found in realm "test-realm"; create it with a KeycloakRealmIdentityProvider`,
		},
		{
			name: "error - missing new identity provider unlinks nothing",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{{Alias: "idp2"}}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{OrganizationID: testOrgID},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To(testIdPAlias)},
					}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, "idp2").
					Return(nil, (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: http.StatusNotFound, Message: "not found"}).Once()

				return client
			},
			wantErr:     require.Error,
			wantErrText: `identity provider "idp2" not found in realm "test-realm"`,
		},
		{
			name: "error - GetIdentityProvider fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{{Alias: testIdPAlias}}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{OrganizationID: testOrgID},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, testIdPAlias).
					Return(nil, (*keycloakapi.Response)(nil), errors.New("boom")).Once()

				return client
			},
			wantErr:     require.Error,
			wantErrText: `unable to get identity provider "idp1": boom`,
		},
		{
			name: "success - empty identity providers list",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: ptr.To([]keycloakApi.OrgIdentityProvider{}),
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: testOrgID,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To(testIdPAlias)},
						{Alias: ptr.To("idp2")},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, testOrgID, testIdPAlias).
					Return((*keycloakapi.Response)(nil), nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, testOrgID, "idp2").
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "absent identity providers: no Keycloak call",
			organization: &keycloakApi.KeycloakOrganization{
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: testOrgID,
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				return keycloakapimocks.NewMockOrganizationsClient(t)
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgClient := tt.keycloakClient(t)
			kc := &keycloakapi.KeycloakClient{}
			kc.Organizations = orgClient
			kc.IdentityProviders = keycloakapimocks.NewMockIdentityProvidersClient(t)

			if tt.idpClient != nil {
				kc.IdentityProviders = tt.idpClient(t)
			}

			handler := NewProcessIdentityProviders(kc)
			err := handler.ServeRequest(context.Background(), tt.organization, tt.realmName)

			tt.wantErr(t, err)

			if tt.wantErrText != "" {
				require.ErrorContains(t, err, tt.wantErrText)
			}
		})
	}
}

func linkedIdP(autoMembership bool, membershipType string) keycloakapi.IdentityProviderRepresentation {
	return keycloakapi.IdentityProviderRepresentation{
		Alias: ptr.To(testIdPAlias),
		OrganizationLinks: &[]keycloakapi.OrganizationIdentityProviderLinkRepresentation{{
			OrganizationId: ptr.To(testOrgID),
			AutoMembership: ptr.To(autoMembership),
			MembershipType: ptr.To(membershipType),
		}},
	}
}

func linkBody(autoMembership bool, membershipType string) keycloakapi.OrganizationIdentityProviderLinkRepresentation {
	return keycloakapi.OrganizationIdentityProviderLinkRepresentation{
		AutoMembership: ptr.To(autoMembership),
		MembershipType: ptr.To(membershipType),
	}
}

func TestProcessIdentityProviders_LinkSettings(t *testing.T) {
	t.Parallel()

	getIdPs := func(client *keycloakapimocks.MockOrganizationsClient, idps ...keycloakapi.IdentityProviderRepresentation) {
		client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, testOrgID).
			Return(idps, (*keycloakapi.Response)(nil), nil).Once()
	}

	org := func(idps ...keycloakApi.OrgIdentityProvider) *keycloakApi.KeycloakOrganization {
		return &keycloakApi.KeycloakOrganization{
			Spec:   keycloakApi.KeycloakOrganizationSpec{IdentityProviders: &idps},
			Status: keycloakApi.KeycloakOrganizationStatus{OrganizationID: testOrgID},
		}
	}

	tests := []struct {
		name         string
		organization *keycloakApi.KeycloakOrganization
		setup        func(orgs *keycloakapimocks.MockOrganizationsClient, idps *keycloakapimocks.MockIdentityProvidersClient)
		wantErrText  string
	}{
		{
			name:         "new link with settings is re-read and updated",
			organization: org(keycloakApi.OrgIdentityProvider{Alias: testIdPAlias, AutoMembership: ptr.To(false)}),
			setup: func(orgs *keycloakapimocks.MockOrganizationsClient, idps *keycloakapimocks.MockIdentityProvidersClient) {
				getIdPs(orgs)
				idps.On("GetIdentityProvider", mock.Anything, testRealmName, testIdPAlias).
					Return(&keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()
				orgs.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, testOrgID, testIdPAlias).
					Return((*keycloakapi.Response)(nil), nil).Once()
				getIdPs(orgs, linkedIdP(true, "UNMANAGED"))
				orgs.On("UpdateOrganizationIdentityProviderLink", mock.Anything, testRealmName, testOrgID, testIdPAlias,
					linkBody(false, "UNMANAGED")).
					Return((*keycloakapi.Response)(nil), nil).Once()
			},
		},
		{
			name: "settled link is not written",
			organization: org(keycloakApi.OrgIdentityProvider{
				Alias: testIdPAlias, AutoMembership: ptr.To(false), MembershipType: ptr.To("UNMANAGED"),
			}),
			setup: func(orgs *keycloakapimocks.MockOrganizationsClient, _ *keycloakapimocks.MockIdentityProvidersClient) {
				getIdPs(orgs, linkedIdP(false, "UNMANAGED"))
			},
		},
		{
			name:         "drift is repaired and the other field comes from Keycloak",
			organization: org(keycloakApi.OrgIdentityProvider{Alias: testIdPAlias, AutoMembership: ptr.To(false)}),
			setup: func(orgs *keycloakapimocks.MockOrganizationsClient, _ *keycloakapimocks.MockIdentityProvidersClient) {
				getIdPs(orgs, linkedIdP(true, "UNMANAGED"))
				orgs.On("UpdateOrganizationIdentityProviderLink", mock.Anything, testRealmName, testOrgID, testIdPAlias,
					linkBody(false, "UNMANAGED")).
					Return((*keycloakapi.Response)(nil), nil).Once()
			},
		},
		{
			name:         "only membershipType set",
			organization: org(keycloakApi.OrgIdentityProvider{Alias: testIdPAlias, MembershipType: ptr.To("MANAGED")}),
			setup: func(orgs *keycloakapimocks.MockOrganizationsClient, _ *keycloakapimocks.MockIdentityProvidersClient) {
				getIdPs(orgs, linkedIdP(true, "UNMANAGED"))
				orgs.On("UpdateOrganizationIdentityProviderLink", mock.Anything, testRealmName, testOrgID, testIdPAlias,
					linkBody(true, "MANAGED")).
					Return((*keycloakapi.Response)(nil), nil).Once()
			},
		},
		{
			name:         "autoMembership false with fetched MANAGED sends the pair and surfaces the Keycloak error",
			organization: org(keycloakApi.OrgIdentityProvider{Alias: testIdPAlias, AutoMembership: ptr.To(false)}),
			setup: func(orgs *keycloakapimocks.MockOrganizationsClient, _ *keycloakapimocks.MockIdentityProvidersClient) {
				getIdPs(orgs, linkedIdP(true, "MANAGED"))
				orgs.On("UpdateOrganizationIdentityProviderLink", mock.Anything, testRealmName, testOrgID, testIdPAlias,
					linkBody(false, "MANAGED")).
					Return((*keycloakapi.Response)(nil), &keycloakapi.ApiError{
						Code: http.StatusBadRequest, Message: "Auto-membership must be enabled when membership type is MANAGED",
					}).Once()
			},
			wantErrText: `unable to update link settings of identity provider "idp1"`,
		},
		{
			name:         "no organization links is an error without a PUT",
			organization: org(keycloakApi.OrgIdentityProvider{Alias: testIdPAlias, AutoMembership: ptr.To(false)}),
			setup: func(orgs *keycloakapimocks.MockOrganizationsClient, _ *keycloakapimocks.MockIdentityProvidersClient) {
				getIdPs(orgs, keycloakapi.IdentityProviderRepresentation{Alias: ptr.To(testIdPAlias)})
			},
			wantErrText: "require Keycloak 26.8 or later",
		},
		{
			name:         "link of another organization is ignored",
			organization: org(keycloakApi.OrgIdentityProvider{Alias: testIdPAlias, AutoMembership: ptr.To(false)}),
			setup: func(orgs *keycloakapimocks.MockOrganizationsClient, _ *keycloakapimocks.MockIdentityProvidersClient) {
				other := linkedIdP(true, "UNMANAGED")
				(*other.OrganizationLinks)[0].OrganizationId = ptr.To("org-other")
				getIdPs(orgs, other)
			},
			wantErrText: "require Keycloak 26.8 or later",
		},
		{
			name:         "link update returns 404",
			organization: org(keycloakApi.OrgIdentityProvider{Alias: testIdPAlias, AutoMembership: ptr.To(false)}),
			setup: func(orgs *keycloakapimocks.MockOrganizationsClient, _ *keycloakapimocks.MockIdentityProvidersClient) {
				getIdPs(orgs, linkedIdP(true, "UNMANAGED"))
				orgs.On("UpdateOrganizationIdentityProviderLink", mock.Anything, testRealmName, testOrgID, testIdPAlias,
					linkBody(false, "UNMANAGED")).
					Return((*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: http.StatusNotFound, Message: "not found"}).Once()
			},
			wantErrText: "unable to update link settings",
		},
		{
			name:         "no settings in spec: no re-read and no PUT",
			organization: org(keycloakApi.OrgIdentityProvider{Alias: testIdPAlias}),
			setup: func(orgs *keycloakapimocks.MockOrganizationsClient, idps *keycloakapimocks.MockIdentityProvidersClient) {
				getIdPs(orgs)
				idps.On("GetIdentityProvider", mock.Anything, testRealmName, testIdPAlias).
					Return(&keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()
				orgs.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, testOrgID, testIdPAlias).
					Return((*keycloakapi.Response)(nil), nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgs := keycloakapimocks.NewMockOrganizationsClient(t)
			idps := keycloakapimocks.NewMockIdentityProvidersClient(t)
			tt.setup(orgs, idps)

			kc := &keycloakapi.KeycloakClient{}
			kc.Organizations = orgs
			kc.IdentityProviders = idps

			err := NewProcessIdentityProviders(kc).ServeRequest(context.Background(), tt.organization, testRealmName)

			if tt.wantErrText != "" {
				require.ErrorContains(t, err, tt.wantErrText)
				return
			}

			require.NoError(t, err)
		})
	}
}
