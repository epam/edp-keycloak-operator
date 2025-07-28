package adapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func createTestAdapter(t *testing.T, server *httptest.Server) *GoCloakAdapter {
	t.Helper()

	mockClient := mocks.NewMockGoCloak(t)
	restyClient := resty.New()
	restyClient.SetBaseURL(server.URL)
	mockClient.On("RestyClient").Return(restyClient).Maybe()

	return &GoCloakAdapter{
		client:   mockClient,
		basePath: server.URL,
		token:    &gocloak.JWT{AccessToken: "token"},
		log:      mock.NewLogr(),
	}
}

func TestGoCloakAdapter_CreateOrganization(t *testing.T) {
	org := &dto.Organization{
		Name:        "test-org",
		Alias:       "test-org",
		Description: "Test organization",
		RedirectURL: "http://test.com",
		Attributes: map[string][]string{
			"key1": {"value1", "value2"},
		},
		Domains: []dto.OrganizationDomain{
			{Name: "test.com"},
		},
	}

	tests := []struct {
		name           string
		realm          string
		organization   *dto.Organization
		setupServer    func() *httptest.Server
		expectedError  string
		expectedResult bool
	}{
		{
			name:         "successful creation",
			realm:        "realm-name",
			organization: org,
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "POST" && r.URL.Path == "/admin/realms/realm-name/organizations" {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedError:  "",
			expectedResult: true,
		},
		{
			name:         "server error",
			realm:        "realm-name-error",
			organization: org,
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte("fatal"))
					require.NoError(t, err)
				}))
			},
			expectedError:  "unable to create organization",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := createTestAdapter(t, server)

			err := adapter.CreateOrganization(context.Background(), tt.realm, tt.organization)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_GetOrganization(t *testing.T) {
	expectedOrg := &dto.Organization{
		ID:          "org-id",
		Name:        "test-org",
		Alias:       "test-org",
		Description: "Test organization",
		RedirectURL: "http://test.com",
		Attributes: map[string][]string{
			"key1": {"value1", "value2"},
		},
		Domains: []dto.OrganizationDomain{
			{Name: "test.com"},
		},
	}

	tests := []struct {
		name             string
		realm            string
		orgID            string
		setupServer      func() *httptest.Server
		expectedOrg      *dto.Organization
		expectedError    string
		expectedNotFound bool
	}{
		{
			name:  "successful retrieval",
			realm: "realm-name",
			orgID: "org-id",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "GET" && r.URL.Path == "/admin/realms/realm-name/organizations/org-id" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(expectedOrg))
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedOrg:      expectedOrg,
			expectedError:    "",
			expectedNotFound: false,
		},
		{
			name:  "not found",
			realm: "realm-name",
			orgID: "non-existent",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedOrg:      nil,
			expectedError:    "",
			expectedNotFound: true,
		},
		{
			name:  "server error",
			realm: "realm-name",
			orgID: "error-org",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte("fatal"))
					require.NoError(t, err)
				}))
			},
			expectedOrg:      nil,
			expectedError:    "unable to get organization",
			expectedNotFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := createTestAdapter(t, server)

			org, err := adapter.GetOrganization(context.Background(), tt.realm, tt.orgID)

			switch {
			case tt.expectedNotFound:
				require.Error(t, err)
				require.True(t, IsErrNotFound(err))
			case tt.expectedError != "":
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			default:
				require.NoError(t, err)
				require.Equal(t, tt.expectedOrg, org)
			}
		})
	}
}

func TestGoCloakAdapter_UpdateOrganization(t *testing.T) {
	org := &dto.Organization{
		ID:          "org-id",
		Name:        "updated-org",
		Alias:       "updated-org",
		Description: "Updated organization",
		RedirectURL: "http://updated.com",
		Attributes: map[string][]string{
			"key2": {"value3"},
		},
		Domains: []dto.OrganizationDomain{
			{Name: "updated.com"},
		},
	}

	orgWithoutID := &dto.Organization{
		Name:  "test-org",
		Alias: "test-org",
	}

	tests := []struct {
		name           string
		realm          string
		organization   *dto.Organization
		setupServer    func() *httptest.Server
		expectedError  string
		expectedResult bool
	}{
		{
			name:         "successful update",
			realm:        "realm-name",
			organization: org,
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "PUT" && r.URL.Path == "/admin/realms/realm-name/organizations/org-id" {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedError:  "",
			expectedResult: true,
		},
		{
			name:         "missing ID",
			realm:        "realm-name",
			organization: orgWithoutID,
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectedError:  "organization ID is required for update",
			expectedResult: false,
		},
		{
			name:         "server error",
			realm:        "realm-name",
			organization: &dto.Organization{ID: "error-org"},
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte("fatal"))
					require.NoError(t, err)
				}))
			},
			expectedError:  "unable to update organization",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := createTestAdapter(t, server)

			err := adapter.UpdateOrganization(context.Background(), tt.realm, tt.organization)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_DeleteOrganization(t *testing.T) {
	tests := []struct {
		name           string
		realm          string
		orgID          string
		setupServer    func() *httptest.Server
		expectedError  string
		expectedResult bool
	}{
		{
			name:  "successful deletion",
			realm: "realm-name",
			orgID: "org-id",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "DELETE" && r.URL.Path == "/admin/realms/realm-name/organizations/org-id" {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedError:  "",
			expectedResult: true,
		},
		{
			name:  "server error",
			realm: "realm-name",
			orgID: "error-org",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte("fatal"))
					require.NoError(t, err)
				}))
			},
			expectedError:  "unable to delete organization",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := createTestAdapter(t, server)

			err := adapter.DeleteOrganization(context.Background(), tt.realm, tt.orgID)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_GetOrganizations(t *testing.T) {
	expectedOrgs := []dto.Organization{
		{
			ID:          "org1",
			Name:        "org1",
			Alias:       "org1",
			Description: "First organization",
		},
		{
			ID:          "org2",
			Name:        "org2",
			Alias:       "org2",
			Description: "Second organization",
		},
	}

	tests := []struct {
		name           string
		realm          string
		setupServer    func() *httptest.Server
		expectedOrgs   []dto.Organization
		expectedError  string
		expectedResult bool
	}{
		{
			name:  "successful retrieval",
			realm: "realm-name",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "GET" && r.URL.Path == "/admin/realms/realm-name/organizations" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(expectedOrgs))
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedOrgs:   expectedOrgs,
			expectedError:  "",
			expectedResult: true,
		},
		{
			name:  "server error",
			realm: "realm-name-error",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte("fatal"))
					require.NoError(t, err)
				}))
			},
			expectedOrgs:   nil,
			expectedError:  "unable to get organizations",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := createTestAdapter(t, server)

			orgs, err := adapter.GetOrganizations(context.Background(), tt.realm, nil)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedOrgs, orgs)
			}
		})
	}
}

func TestGoCloakAdapter_LinkIdentityProviderToOrganization(t *testing.T) {
	tests := []struct {
		name           string
		realm          string
		orgID          string
		idpAlias       string
		setupServer    func() *httptest.Server
		expectedError  string
		expectedResult bool
	}{
		{
			name:     "successful linking",
			realm:    "realm-name",
			orgID:    "org-id",
			idpAlias: "github-idp",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "POST" && r.URL.Path == "/admin/realms/realm-name/organizations/org-id/identity-providers" {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedError:  "",
			expectedResult: true,
		},
		{
			name:     "server error",
			realm:    "realm-name",
			orgID:    "error-org",
			idpAlias: "github-idp",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte("fatal"))
					require.NoError(t, err)
				}))
			},
			expectedError:  "unable to link identity provider to organization",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := createTestAdapter(t, server)

			err := adapter.LinkIdentityProviderToOrganization(context.Background(), tt.realm, tt.orgID, tt.idpAlias)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_UnlinkIdentityProviderFromOrganization(t *testing.T) {
	tests := []struct {
		name           string
		realm          string
		orgID          string
		idpAlias       string
		setupServer    func() *httptest.Server
		expectedError  string
		expectedResult bool
	}{
		{
			name:     "successful unlinking",
			realm:    "realm-name",
			orgID:    "org-id",
			idpAlias: "github-idp",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "DELETE" && r.URL.Path == "/admin/realms/realm-name/organizations/org-id/identity-providers/github-idp" {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedError:  "",
			expectedResult: true,
		},
		{
			name:     "server error",
			realm:    "realm-name",
			orgID:    "error-org",
			idpAlias: "github-idp",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte("fatal"))
					require.NoError(t, err)
				}))
			},
			expectedError:  "unable to unlink identity provider from organization",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := createTestAdapter(t, server)

			err := adapter.UnlinkIdentityProviderFromOrganization(context.Background(), tt.realm, tt.orgID, tt.idpAlias)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGoCloakAdapter_GetOrganizationIdentityProviders(t *testing.T) {
	expectedIdPs := []dto.OrganizationIdentityProvider{
		{Alias: "github-idp"},
		{Alias: "google-idp"},
	}

	tests := []struct {
		name           string
		realm          string
		orgID          string
		setupServer    func() *httptest.Server
		expectedIdPs   []dto.OrganizationIdentityProvider
		expectedError  string
		expectedResult bool
	}{
		{
			name:  "successful retrieval",
			realm: "realm-name",
			orgID: "org-id",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "GET" && r.URL.Path == "/admin/realms/realm-name/organizations/org-id/identity-providers" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(expectedIdPs))
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedIdPs:   expectedIdPs,
			expectedError:  "",
			expectedResult: true,
		},
		{
			name:  "server error",
			realm: "realm-name",
			orgID: "error-org",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte("fatal"))
					require.NoError(t, err)
				}))
			},
			expectedIdPs:   nil,
			expectedError:  "unable to get organization identity providers",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := createTestAdapter(t, server)

			idPs, err := adapter.GetOrganizationIdentityProviders(context.Background(), tt.realm, tt.orgID)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectedIdPs, idPs)
			}
		})
	}
}

func TestGoCloakAdapter_OrganizationExists(t *testing.T) {
	tests := []struct {
		name           string
		realm          string
		orgID          string
		setupServer    func() *httptest.Server
		expectedExists bool
		expectedError  string
	}{
		{
			name:  "organization exists",
			realm: "realm-name",
			orgID: "existing-org",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "GET" && r.URL.Path == "/admin/realms/realm-name/organizations/existing-org" {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusOK)
						require.NoError(t, json.NewEncoder(w).Encode(&dto.Organization{ID: "existing-org"}))
						return
					}
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedExists: true,
			expectedError:  "",
		},
		{
			name:  "organization doesn't exist",
			realm: "realm-name",
			orgID: "non-existent-org",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			expectedExists: false,
			expectedError:  "",
		},
		{
			name:  "server error",
			realm: "realm-name",
			orgID: "error-org",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, err := w.Write([]byte("fatal"))
					require.NoError(t, err)
				}))
			},
			expectedExists: false,
			expectedError:  "unable to get organization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			adapter := createTestAdapter(t, server)

			exists, err := adapter.OrganizationExists(context.Background(), tt.realm, tt.orgID)

			if tt.expectedError != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.expectedExists, exists)
		})
	}
}
