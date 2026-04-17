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
	keycloakmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestCreateOrganization_ServeRequest(t *testing.T) {
	tests := []struct {
		name           string
		organization   *keycloakApi.KeycloakOrganization
		realmName      string
		keycloakClient func(t *testing.T) keycloakapi.OrganizationsClient
		wantErr        require.ErrorAssertionFunc
		expectedOrgID  string
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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404, Message: "organization not found"}).Once()

				// Second call: CreateOrganization succeeds
				client.On("CreateOrganization", mock.Anything, "test-realm", mock.MatchedBy(func(org keycloakapi.OrganizationRepresentation) bool {
					return ptr.Deref(org.Name, "") == "Test Organization" &&
						ptr.Deref(org.Alias, "") == "test-org" &&
						ptr.Deref(org.Description, "") == "Test organization" &&
						ptr.Deref(org.RedirectUrl, "") == "https://example.com/redirect" &&
						len(ptr.Deref(org.Domains, nil)) == 2 &&
						len(ptr.Deref(org.Attributes, nil)) == 2
				})).Return((*keycloakapi.Response)(nil), nil).Once()

				// Third call: GetOrganizationByAlias returns the created organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("org-123"),
						Alias: ptr.To("test-org"),
					}, (*keycloakapi.Response)(nil), nil).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns existing organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "existing-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("existing-org-456"),
						Alias: ptr.To("existing-org"),
					}, (*keycloakapi.Response)(nil), nil).Once()

				// Second call: UpdateOrganization succeeds
				client.On("UpdateOrganization", mock.Anything, "test-realm", "existing-org-456", mock.MatchedBy(func(org keycloakapi.OrganizationRepresentation) bool {
					return ptr.Deref(org.Name, "") == "Updated Organization" &&
						ptr.Deref(org.Alias, "") == "existing-org" &&
						ptr.Deref(org.Description, "") == "Updated organization" &&
						ptr.Deref(org.RedirectUrl, "") == "https://updated.com/redirect" &&
						len(ptr.Deref(org.Domains, nil)) == 1
				})).Return((*keycloakapi.Response)(nil), nil).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)

				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("network error")).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404, Message: "organization not found"}).Once()

				// Second call: CreateOrganization fails
				client.On("CreateOrganization", mock.Anything, "test-realm", mock.Anything).
					Return((*keycloakapi.Response)(nil), errors.New("creation failed")).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns existing organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "existing-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("existing-org-456"),
						Alias: ptr.To("existing-org"),
					}, (*keycloakapi.Response)(nil), nil).Once()

				// Second call: UpdateOrganization fails
				client.On("UpdateOrganization", mock.Anything, "test-realm", "existing-org-456", mock.Anything).
					Return((*keycloakapi.Response)(nil), errors.New("update failed")).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404, Message: "organization not found"}).Once()

				// Second call: CreateOrganization succeeds
				client.On("CreateOrganization", mock.Anything, "test-realm", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Once()

				// Third call: GetOrganizationByAlias fails after creation
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "test-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("failed to retrieve created organization")).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns not found
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "minimal-org").
					Return((*keycloakapi.OrganizationRepresentation)(nil), (*keycloakapi.Response)(nil), &keycloakapi.ApiError{Code: 404, Message: "organization not found"}).Once()

				// Second call: CreateOrganization succeeds
				client.On("CreateOrganization", mock.Anything, "test-realm", mock.MatchedBy(func(org keycloakapi.OrganizationRepresentation) bool {
					return ptr.Deref(org.Name, "") == "Minimal Org" &&
						ptr.Deref(org.Alias, "") == "minimal-org" &&
						len(ptr.Deref(org.Domains, nil)) == 1
				})).Return((*keycloakapi.Response)(nil), nil).Once()

				// Third call: GetOrganizationByAlias returns the created organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "minimal-org").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("minimal-org-789"),
						Alias: ptr.To("minimal-org"),
					}, (*keycloakapi.Response)(nil), nil).Once()

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
			realmName: "test-realm",
			keycloakClient: func(t *testing.T) keycloakapi.OrganizationsClient {
				client := keycloakmocks.NewMockOrganizationsClient(t)

				// First call: GetOrganizationByAlias returns existing organization
				client.On("GetOrganizationByAlias", mock.Anything, "test-realm", "existing-org-with-id").
					Return(&keycloakapi.OrganizationRepresentation{
						Id:    ptr.To("existing-id-123"),
						Alias: ptr.To("existing-org-with-id"),
					}, (*keycloakapi.Response)(nil), nil).Once()

				// Second call: UpdateOrganization succeeds
				client.On("UpdateOrganization", mock.Anything, "test-realm", "existing-id-123", mock.MatchedBy(func(org keycloakapi.OrganizationRepresentation) bool {
					return ptr.Deref(org.Name, "") == "Existing Org" &&
						ptr.Deref(org.Alias, "") == "existing-org-with-id"
				})).Return((*keycloakapi.Response)(nil), nil).Once()

				return client
			},
			wantErr:       require.NoError,
			expectedOrgID: "existing-id-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgClient := tt.keycloakClient(t)
			kc := &keycloakapi.APIClient{}
			kc.Organizations = orgClient

			handler := NewCreateOrganization(kc)
			err := handler.ServeRequest(context.Background(), tt.organization, tt.realmName)

			tt.wantErr(t, err)

			if err == nil {
				require.Equal(t, tt.expectedOrgID, tt.organization.Status.OrganizationID)
			}
		})
	}
}

func TestSpecToOrganizationRepresentation(t *testing.T) {
	tests := []struct {
		name   string
		org    *keycloakApi.KeycloakOrganization
		verify func(t *testing.T, rep keycloakapi.OrganizationRepresentation)
	}{
		{
			name: "full spec with all fields",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:        "Test Organization",
					Alias:       "test-org",
					Description: "A description",
					RedirectURL: "https://example.com/redirect",
					Domains:     []string{"example.com", "test.com"},
					Attributes: map[string][]string{
						"dept": {"eng"},
						"loc":  {"us", "eu"},
					},
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()

				require.Equal(t, "Test Organization", ptr.Deref(rep.Name, ""))
				require.Equal(t, "test-org", ptr.Deref(rep.Alias, ""))
				require.Equal(t, "A description", ptr.Deref(rep.Description, ""))
				require.Equal(t, "https://example.com/redirect", ptr.Deref(rep.RedirectUrl, ""))

				require.NotNil(t, rep.Domains)
				require.Len(t, *rep.Domains, 2)

				domainNames := make([]string, len(*rep.Domains))
				for i, d := range *rep.Domains {
					domainNames[i] = ptr.Deref(d.Name, "")
				}

				require.ElementsMatch(t, []string{"example.com", "test.com"}, domainNames)

				require.NotNil(t, rep.Attributes)
				attrs := *rep.Attributes
				require.Equal(t, []string{"eng"}, attrs["dept"])
				require.ElementsMatch(t, []string{"us", "eu"}, attrs["loc"])
			},
		},
		{
			name: "minimal spec - optional fields absent",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "Minimal Org",
					Alias:   "minimal-org",
					Domains: []string{"minimal.com"},
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()

				require.Equal(t, "Minimal Org", ptr.Deref(rep.Name, ""))
				require.Equal(t, "minimal-org", ptr.Deref(rep.Alias, ""))
				require.Equal(t, "", ptr.Deref(rep.Description, ""))
				require.Equal(t, "", ptr.Deref(rep.RedirectUrl, ""))
				require.Nil(t, rep.Attributes)
				require.NotNil(t, rep.Domains)
				require.Len(t, *rep.Domains, 1)
				require.Equal(t, "minimal.com", ptr.Deref((*rep.Domains)[0].Name, ""))
			},
		},
		{
			name: "nil domains",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:    "No Domains Org",
					Alias:   "no-domains-org",
					Domains: nil,
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()
				require.Nil(t, rep.Domains)
			},
		},
		{
			name: "nil attributes",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:       "No Attrs Org",
					Alias:      "no-attrs-org",
					Domains:    []string{"no-attrs.com"},
					Attributes: nil,
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()
				require.Nil(t, rep.Attributes)
			},
		},
		{
			name: "empty attributes map",
			org: &keycloakApi.KeycloakOrganization{
				Spec: keycloakApi.KeycloakOrganizationSpec{
					Name:       "Empty Attrs Org",
					Alias:      "empty-attrs-org",
					Domains:    []string{"empty-attrs.com"},
					Attributes: map[string][]string{},
				},
			},
			verify: func(t *testing.T, rep keycloakapi.OrganizationRepresentation) {
				t.Helper()
				require.Nil(t, rep.Attributes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := specToOrganizationRepresentation(tt.org)
			tt.verify(t, rep)
		})
	}
}
