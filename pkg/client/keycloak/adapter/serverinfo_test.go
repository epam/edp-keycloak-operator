package adapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

// setupTestServer creates a test server and GoCloakAdapter for testing
func setupTestServer(t *testing.T, serverResponse string, statusCode int) *GoCloakAdapter {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/admin/serverinfo") {
			setJSONContentType(w)
			w.WriteHeader(statusCode)
			_, err := w.Write([]byte(serverResponse))
			assert.NoError(t, err)

			return
		}

		// Return 200 OK for all other requests to avoid authentication issues
		w.WriteHeader(http.StatusOK)
	}))

	t.Cleanup(func() {
		server.Close()
	})

	// Create adapter directly to avoid authentication
	mockClient := newMockClientWithResty(t, server.URL)

	a := &GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "test-token"},
		log:      logr.Discard(),
		basePath: server.URL,
	}

	return a
}

func TestGoCloakAdapter_GetServerInfo_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverResponse string
		expected       dto.ServerInfo
	}{
		{
			name: "successful server info retrieval with features",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				},
				"features": [
					{
						"name": "ADMIN_FINE_GRAINED_AUTHZ",
						"enabled": true
					},
					{
						"name": "ADMIN_FINE_GRAINED_AUTHZ",
						"enabled": false
					},
					{
						"name": "CUSTOM_ATTRIBUTES",
						"enabled": true
					}
				]
			}`,
			expected: dto.ServerInfo{
				SystemInfo: dto.SystemInfo{
					Version: "22.0.5",
				},
				Features: []dto.ServerFeature{
					{
						Name:    "ADMIN_FINE_GRAINED_AUTHZ",
						Enabled: true,
					},
					{
						Name:    "ADMIN_FINE_GRAINED_AUTHZ",
						Enabled: false,
					},
					{
						Name:    "CUSTOM_ATTRIBUTES",
						Enabled: true,
					},
				},
			},
		},
		{
			name: "server info with empty features",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				},
				"features": []
			}`,
			expected: dto.ServerInfo{
				SystemInfo: dto.SystemInfo{
					Version: "22.0.5",
				},
				Features: []dto.ServerFeature{},
			},
		},
		{
			name: "server info with missing features field",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				}
			}`,
			expected: dto.ServerInfo{
				SystemInfo: dto.SystemInfo{
					Version: "22.0.5",
				},
				Features: nil, // Should be nil when not present in JSON
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := setupTestServer(t, tt.serverResponse, http.StatusOK)

			got, err := a.GetServerInfo(context.Background())
			require.NoError(t, err)

			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGoCloakAdapter_GetServerInfo_ErrorScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name:           "server returns 500 error",
			serverResponse: `{"error":"internal server error"}`,
			statusCode:     http.StatusInternalServerError,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get server info")
				assert.Contains(t, err.Error(), "500")
			},
		},
		{
			name:           "server returns 401 unauthorized",
			serverResponse: `{"error":"unauthorized"}`,
			statusCode:     http.StatusUnauthorized,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get server info")
				assert.Contains(t, err.Error(), "401")
			},
		},
		{
			name:           "server returns 403 forbidden",
			serverResponse: `{"error":"forbidden"}`,
			statusCode:     http.StatusForbidden,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get server info")
				assert.Contains(t, err.Error(), "403")
			},
		},
		{
			name:           "server returns invalid JSON",
			serverResponse: `{"invalid": json}`,
			statusCode:     http.StatusOK,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get server info")
			},
		},
		{
			name:           "server returns empty response",
			serverResponse: ``,
			statusCode:     http.StatusOK,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get server info")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := setupTestServer(t, tt.serverResponse, tt.statusCode)

			_, err := a.GetServerInfo(context.Background())
			tt.wantErr(t, err)
		})
	}
}

func TestGoCloakAdapter_FeatureFlagEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverResponse string
		featureFlag    string
		expected       bool
		expectedErr    require.ErrorAssertionFunc
	}{
		{
			name: "feature flag enabled",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				},
				"features": [
					{
						"name": "ADMIN_FINE_GRAINED_AUTHZ",
						"enabled": true
					},
					{
						"name": "CUSTOM_ATTRIBUTES",
						"enabled": false
					}
				]
			}`,
			featureFlag: "ADMIN_FINE_GRAINED_AUTHZ",
			expected:    true,
			expectedErr: require.NoError,
		},
		{
			name: "feature flag disabled",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				},
				"features": [
					{
						"name": "ADMIN_FINE_GRAINED_AUTHZ",
						"enabled": false
					},
					{
						"name": "CUSTOM_ATTRIBUTES",
						"enabled": true
					}
				]
			}`,
			featureFlag: "ADMIN_FINE_GRAINED_AUTHZ",
			expected:    false,
			expectedErr: require.NoError,
		},
		{
			name: "feature flag not found",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				},
				"features": [
					{
						"name": "CUSTOM_ATTRIBUTES",
						"enabled": true
					}
				]
			}`,
			featureFlag: "NON_EXISTENT_FEATURE",
			expected:    false,
			expectedErr: require.NoError,
		},
		{
			name: "empty features array",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				},
				"features": []
			}`,
			featureFlag: "ADMIN_FINE_GRAINED_AUTHZ",
			expected:    false,
			expectedErr: require.NoError,
		},
		{
			name: "missing features field",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				}
			}`,
			featureFlag: "ADMIN_FINE_GRAINED_AUTHZ",
			expected:    false,
			expectedErr: require.NoError,
		},
		{
			name: "case sensitive feature flag",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				},
				"features": [
					{
						"name": "admin_fine_grained_authz",
						"enabled": true
					}
				]
			}`,
			featureFlag: "ADMIN_FINE_GRAINED_AUTHZ",
			expected:    false, // Should not match due to case sensitivity
			expectedErr: require.NoError,
		},
		{
			name: "multiple features with same name",
			serverResponse: `{
				"systemInfo": {
					"version": "22.0.5"
				},
				"features": [
					{
						"name": "ADMIN_FINE_GRAINED_AUTHZ",
						"enabled": false
					},
					{
						"name": "ADMIN_FINE_GRAINED_AUTHZ",
						"enabled": true
					}
				]
			}`,
			featureFlag: "ADMIN_FINE_GRAINED_AUTHZ",
			expected:    false, // Should return first match (false)
			expectedErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := setupTestServer(t, tt.serverResponse, http.StatusOK)

			got, err := a.FeatureFlagEnabled(context.Background(), tt.featureFlag)
			tt.expectedErr(t, err)

			if err == nil {
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestGoCloakAdapter_FeatureFlagEnabled_ErrorScenarios(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		featureFlag    string
		expected       bool
		expectedErr    require.ErrorAssertionFunc
	}{
		{
			name:           "server returns 500 error",
			serverResponse: `{"error":"internal server error"}`,
			statusCode:     http.StatusInternalServerError,
			featureFlag:    "ADMIN_FINE_GRAINED_AUTHZ",
			expected:       false,
			expectedErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get server info")
			},
		},
		{
			name:           "server returns 401 unauthorized",
			serverResponse: `{"error":"unauthorized"}`,
			statusCode:     http.StatusUnauthorized,
			featureFlag:    "ADMIN_FINE_GRAINED_AUTHZ",
			expected:       false,
			expectedErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get server info")
			},
		},
		{
			name:           "server returns invalid JSON",
			serverResponse: `{"invalid": json}`,
			statusCode:     http.StatusOK,
			featureFlag:    "ADMIN_FINE_GRAINED_AUTHZ",
			expected:       false,
			expectedErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get server info")
			},
		},
		{
			name:           "server returns empty response",
			serverResponse: ``,
			statusCode:     http.StatusOK,
			featureFlag:    "ADMIN_FINE_GRAINED_AUTHZ",
			expected:       false,
			expectedErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get server info")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := setupTestServer(t, tt.serverResponse, tt.statusCode)

			got, err := a.FeatureFlagEnabled(context.Background(), tt.featureFlag)
			tt.expectedErr(t, err)

			if err == nil {
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
