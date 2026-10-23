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
					IdentityProviders: []keycloakApi.OrgIdentityProvider{
						{Alias: "idp1"},
						{Alias: "idp2"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return([]keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, "org-123", "idp1").
					Return((*keycloakapi.Response)(nil), nil).Once()
				client.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, "org-123", "idp2").
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, "idp1").
					Return(&keycloakapi.IdentityProviderRepresentation{Alias: ptr.To("idp1")}, (*keycloakapi.Response)(nil), nil).Once()

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
					IdentityProviders: []keycloakApi.OrgIdentityProvider{
						{Alias: "idp1"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
						{Alias: ptr.To("idp2")},
						{Alias: ptr.To("idp3")},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, "org-123", "idp2").
					Return((*keycloakapi.Response)(nil), nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, "org-123", "idp3").
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "success - mixed scenario: link new and unlink removed",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: []keycloakApi.OrgIdentityProvider{
						{Alias: "idp1"},
						{Alias: "idp3"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
						{Alias: ptr.To("idp2")},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, "org-123", "idp3").
					Return((*keycloakapi.Response)(nil), nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, "org-123", "idp2").
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
					IdentityProviders: []keycloakApi.OrgIdentityProvider{
						{Alias: "idp1"},
						{Alias: "idp2"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
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
					IdentityProviders: []keycloakApi.OrgIdentityProvider{
						{Alias: "idp1"},
					},
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
					IdentityProviders: []keycloakApi.OrgIdentityProvider{
						{Alias: "idp1"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return(nil, (*keycloakapi.Response)(nil), errors.New("keycloak connection failed")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error - LinkIdentityProviderToOrganization fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: []keycloakApi.OrgIdentityProvider{
						{Alias: "idp1"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return([]keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, testRealmName, "org-123", "idp1").
					Return((*keycloakapi.Response)(nil), errors.New("failed to link identity provider")).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, "idp1").
					Return(&keycloakapi.IdentityProviderRepresentation{Alias: ptr.To("idp1")}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error - UnlinkIdentityProviderFromOrganization fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: []keycloakApi.OrgIdentityProvider{
						{Alias: "idp1"},
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
						{Alias: ptr.To("idp2")},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, "org-123", "idp2").
					Return((*keycloakapi.Response)(nil), errors.New("failed to unlink identity provider")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error - identity provider missing in realm",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: []keycloakApi.OrgIdentityProvider{{Alias: "idp1"}},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{OrganizationID: "org-123"},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return([]keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, "idp1").
					Return(nil, (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: http.StatusNotFound, Message: "not found"}).Once()

				return client
			},
			wantErr:     require.Error,
			wantErrText: `identity provider "idp1" not found in realm "test-realm"; create it with a KeycloakRealmIdentityProvider`,
		},
		{
			name: "error - GetIdentityProvider fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: []keycloakApi.OrgIdentityProvider{{Alias: "idp1"}},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{OrganizationID: "org-123"},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return([]keycloakapi.IdentityProviderRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			idpClient: func(t *testing.T) keycloakapi.IdentityProvidersClient {
				client := keycloakapimocks.NewMockIdentityProvidersClient(t)

				client.On("GetIdentityProvider", mock.Anything, testRealmName, "idp1").
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
					IdentityProviders: []keycloakApi.OrgIdentityProvider{},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realmName: testRealmName,
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakapimocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, testRealmName, "org-123").
					Return([]keycloakapi.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
						{Alias: ptr.To("idp2")},
					}, (*keycloakapi.Response)(nil), nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, "org-123", "idp1").
					Return((*keycloakapi.Response)(nil), nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, testRealmName, "org-123", "idp2").
					Return((*keycloakapi.Response)(nil), nil).Once()

				return client
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
