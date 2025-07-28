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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestCreateOrganization_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		organization      *keycloakApi.KeycloakOrganization
		realm             *gocloak.RealmRepresentation
		keycloakClient    func(t *testing.T) keycloak.Client
		wantErr           require.ErrorAssertionFunc
		expectedOrgID     string
		expectedStatusMsg string
	}{
		{
			name: "successfully create new organization",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:        "Test Organization",
					Alias:       "test-org",
					Description: "Test organization",
					RedirectURL: "https://example.com/redirect",
					Domains:     []string{"example.com", "test.com"},
					Attributes: map[string][]string{
						"attr1": {"value1"},
						"attr2": {"value2", "value3"},
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return(nil, adapter.NotFoundError("organization not found")).Once()

				// Second call: CreateOrganization succeeds
				client.On("CreateOrganization", mock.Anything, "test-realm", mock.MatchedBy(func(org *dto.Organization) bool {
					return org.Name == "Test Organization" &&
						org.Alias == "test-org" &&
						org.Description == "Test organization" &&
						org.RedirectURL == "https://example.com/redirect" &&
						len(org.Domains) == 2 &&
						org.Domains[0].Name == "example.com" &&
						org.Domains[1].Name == "test.com" &&
						len(org.Attributes) == 2
				})).Return(nil).Once()

				// Third call: GetOrganizationByAlias returns the created organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return(&dto.Organization{
						ID:    "org-123",
						Name:  "Test Organization",
						Alias: "test-org",
					}, nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "org-123",
		},
		{
			name: "successfully update existing organization",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:        "Updated Organization",
					Alias:       "existing-org",
					Description: "Updated organization",
					RedirectURL: "https://updated.com/redirect",
					Domains:     []string{"updated.com"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// First call: GetOrganizationByAlias returns existing organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "existing-org").
					Return(&dto.Organization{
						ID:    "existing-org-456",
						Name:  "Old Organization",
						Alias: "existing-org",
					}, nil).Once()

				// Second call: UpdateOrganization succeeds
				client.On("UpdateOrganization", mock.Anything, "test-realm", mock.MatchedBy(func(org *dto.Organization) bool {
					return org.ID == "existing-org-456" &&
						org.Name == "Updated Organization" &&
						org.Alias == "existing-org" &&
						org.Description == "Updated organization" &&
						org.RedirectURL == "https://updated.com/redirect" &&
						len(org.Domains) == 1 &&
						org.Domains[0].Name == "updated.com"
				})).Return(nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "existing-org-456",
		},
		{
			name: "error when GetOrganizationByAlias fails with non-not-found error",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Test Organization",
					Alias:   "test-org",
					Domains: []string{"example.com"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return(nil, errors.New("network error")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error when CreateOrganization fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Test Organization",
					Alias:   "test-org",
					Domains: []string{"example.com"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return(nil, adapter.NotFoundError("organization not found")).Once()

				// Second call: CreateOrganization fails
				client.On("CreateOrganization", mock.Anything, "test-realm", mock.Anything).
					Return(errors.New("creation failed")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error when UpdateOrganization fails",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Updated Organization",
					Alias:   "existing-org",
					Domains: []string{"example.com"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// First call: GetOrganizationByAlias returns existing organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "existing-org").
					Return(&dto.Organization{
						ID:    "existing-org-456",
						Name:  "Old Organization",
						Alias: "existing-org",
					}, nil).Once()

				// Second call: UpdateOrganization fails
				client.On("UpdateOrganization", mock.Anything, "test-realm", mock.Anything).
					Return(errors.New("update failed")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "error when GetOrganizationByAlias fails after creation",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Test Organization",
					Alias:   "test-org",
					Domains: []string{"example.com"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return(nil, adapter.NotFoundError("organization not found")).Once()

				// Second call: CreateOrganization succeeds
				client.On("CreateOrganization", mock.Anything, "test-realm", mock.Anything).
					Return(nil).Once()

				// Third call: GetOrganizationByAlias fails after creation
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return(nil, errors.New("failed to retrieve created organization")).Once()

				return client
			},
			wantErr: require.Error,
		},
		{
			name: "organization with minimal required fields",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Minimal Org",
					Alias:   "minimal-org",
					Domains: []string{"minimal.com"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "minimal-org").
					Return(nil, adapter.NotFoundError("organization not found")).Once()

				// Second call: CreateOrganization succeeds
				client.On("CreateOrganization", mock.Anything, "test-realm", mock.MatchedBy(func(org *dto.Organization) bool {
					return org.Name == "Minimal Org" &&
						org.Alias == "minimal-org" &&
						len(org.Domains) == 1 &&
						org.Domains[0].Name == "minimal.com"
				})).Return(nil).Once()

				// Third call: GetOrganizationByAlias returns the created organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "minimal-org").
					Return(&dto.Organization{
						ID:    "minimal-org-789",
						Name:  "Minimal Org",
						Alias: "minimal-org",
					}, nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "minimal-org-789",
		},
		{
			name: "organization with existing ID in status",
			organization: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Existing Org",
					Alias:   "existing-org-with-id",
					Domains: []string{"existing.com"},
				},
				Status: keycloakApi.KeycloakOrganizationStatus{
					OrganizationID: "existing-id-123",
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			keycloakClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)

				// First call: GetOrganizationByAlias returns existing organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "existing-org-with-id").
					Return(&dto.Organization{
						ID:    "existing-id-123",
						Name:  "Old Existing Org",
						Alias: "existing-org-with-id",
					}, nil).Once()

				// Second call: UpdateOrganization succeeds
				client.On("UpdateOrganization", mock.Anything, "test-realm", mock.MatchedBy(func(org *dto.Organization) bool {
					return org.ID == "existing-id-123" &&
						org.Name == "Existing Org" &&
						org.Alias == "existing-org-with-id"
				})).Return(nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "existing-id-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewCreateOrganization(tt.keycloakClient(t))
			err := handler.ServeRequest(context.Background(), tt.organization, tt.realm)

			tt.wantErr(t, err)

			if err == nil {
				require.Equal(t, tt.expectedOrgID, tt.organization.Status.OrganizationID)
			}
		})
	}
}

func TestNewCreateOrganization(t *testing.T) {
	t.Parallel()

	keycloakClient := mocks.NewMockClient(t)

	handler := NewCreateOrganization(keycloakClient)

	require.NotNil(t, handler)
	require.Equal(t, keycloakClient, handler.keycloakClient)
}
