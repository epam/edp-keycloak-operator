package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestProcessIdentityProviders_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		organization   *keycloakApi.KeycloakOrganization
		realm          *gocloak.RealmRepresentation
		keycloakClient func(t *testing.T) keycloak.Client
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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// Mock GetOrganizationIdentityProviders to return empty list
				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]dto.OrganizationIdentityProvider{}, nil).Once()

				// Mock LinkIdentityProviderToOrganization for both identity providers
				client.On("LinkIdentityProviderToOrganization", mock.Anything, "test-realm", "org-123", "idp1").
					Return(nil).Once()
				client.On("LinkIdentityProviderToOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return(nil).Once()

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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// Mock GetOrganizationIdentityProviders to return existing identity providers
				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]dto.OrganizationIdentityProvider{
						{Alias: "idp1"},
						{Alias: "idp2"}, // This should be unlinked
						{Alias: "idp3"}, // This should be unlinked
					}, nil).Once()

				// Mock UnlinkIdentityProviderFromOrganization for removed identity providers
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return(nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp3").
					Return(nil).Once()

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
						{Alias: "idp3"}, // New identity provider
					},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// Mock GetOrganizationIdentityProviders to return existing identity providers
				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]dto.OrganizationIdentityProvider{
						{Alias: "idp1"},
						{Alias: "idp2"}, // This should be unlinked
					}, nil).Once()

				// Mock LinkIdentityProviderToOrganization for new identity provider
				client.On("LinkIdentityProviderToOrganization", mock.Anything, "test-realm", "org-123", "idp3").
					Return(nil).Once()

				// Mock UnlinkIdentityProviderFromOrganization for removed identity provider
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return(nil).Once()

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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// Mock GetOrganizationIdentityProviders to return same identity providers
				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]dto.OrganizationIdentityProvider{
						{Alias: "idp1"},
						{Alias: "idp2"},
					}, nil).Once()

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
					OrganizationID: "", // Empty organization ID
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				// No mock calls expected
				return client
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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return(nil, errors.New("keycloak connection failed")).Once()

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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]dto.OrganizationIdentityProvider{}, nil).Once()

				client.On("LinkIdentityProviderToOrganization", mock.Anything, "test-realm", "org-123", "idp1").
					Return(errors.New("failed to link identity provider")).Once()

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
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]dto.OrganizationIdentityProvider{
						{Alias: "idp1"},
						{Alias: "idp2"}, // This should be unlinked but will fail
					}, nil).Once()

				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return(errors.New("failed to unlink identity provider")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "success - empty identity providers list",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					IdentityProviders: []keycloakApi.OrgIdentityProvider{}, // Empty list
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "org-123",
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				client.On("GetOrganizationIdentityProviders", mock.Anything, "test-realm", "org-123").
					Return([]dto.OrganizationIdentityProvider{
						{Alias: "idp1"},
						{Alias: "idp2"},
					}, nil).Once()

				// Both identity providers should be unlinked
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp1").
					Return(nil).Once()
				client.On("UnlinkIdentityProviderFromOrganization", mock.Anything, "test-realm", "org-123", "idp2").
					Return(nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewProcessIdentityProviders(tt.keycloakClient(t))
			err := handler.ServeRequest(context.Background(), tt.organization, tt.realm)

			tt.wantErr(t, err)
		})
	}
}
