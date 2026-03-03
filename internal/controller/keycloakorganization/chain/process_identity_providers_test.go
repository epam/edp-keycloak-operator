package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestProcessIdentityProviders_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organization   *keycloakApi.KeycloakOrganization
		realmName      string
		keycloakClient func(t *testing.T) keycloakv2.OrganizationsClient
		wantErr        require.ErrorAssertionFunc
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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakv2.OrganizationsClient {
				client := keycloakv2mocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]keycloakv2.IdentityProviderRepresentation{}, (*keycloakv2.Response)(nil), nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, "test-realm", "org-123", "idp1").
					Return((*keycloakv2.Response)(nil), nil).Once()
				client.On("LinkIdentityProviderToOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return((*keycloakv2.Response)(nil), nil).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakv2.OrganizationsClient {
				client := keycloakv2mocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]keycloakv2.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
						{Alias: ptr.To("idp2")},
						{Alias: ptr.To("idp3")},
					}, (*keycloakv2.Response)(nil), nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return((*keycloakv2.Response)(nil), nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp3").
					Return((*keycloakv2.Response)(nil), nil).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakv2.OrganizationsClient {
				client := keycloakv2mocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]keycloakv2.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
						{Alias: ptr.To("idp2")},
					}, (*keycloakv2.Response)(nil), nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, "test-realm", "org-123", "idp3").
					Return((*keycloakv2.Response)(nil), nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return((*keycloakv2.Response)(nil), nil).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakv2.OrganizationsClient {
				client := keycloakv2mocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]keycloakv2.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
						{Alias: ptr.To("idp2")},
					}, (*keycloakv2.Response)(nil), nil).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakv2.OrganizationsClient {
				return keycloakv2mocks.NewMockOrganizationsClient(t)
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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakv2.OrganizationsClient {
				client := keycloakv2mocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return(nil, (*keycloakv2.Response)(nil), errors.New("keycloak connection failed")).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakv2.OrganizationsClient {
				client := keycloakv2mocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]keycloakv2.IdentityProviderRepresentation{}, (*keycloakv2.Response)(nil), nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, "test-realm", "org-123", "idp1").
					Return((*keycloakv2.Response)(nil), errors.New("failed to link identity provider")).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakv2.OrganizationsClient {
				client := keycloakv2mocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]keycloakv2.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
						{Alias: ptr.To("idp2")},
					}, (*keycloakv2.Response)(nil), nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return((*keycloakv2.Response)(nil), errors.New("failed to unlink identity provider")).Once()

				return client
			},
			wantErr: require.Error,
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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakv2.OrganizationsClient {
				client := keycloakv2mocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]keycloakv2.IdentityProviderRepresentation{
						{Alias: ptr.To("idp1")},
						{Alias: ptr.To("idp2")},
					}, (*keycloakv2.Response)(nil), nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp1").
					Return((*keycloakv2.Response)(nil), nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return((*keycloakv2.Response)(nil), nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgClient := tt.keycloakClient(t)
			kc := &keycloakv2.KeycloakClient{}
			kc.Organizations = orgClient

			handler := NewProcessIdentityProviders(kc)
			err := handler.ServeRequest(context.Background(), tt.organization, tt.realmName)

			tt.wantErr(t, err)
		})
	}
}
