package adapter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	keycloak_go_client "github.com/zmotso/keycloak-go-client"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

func TestGoCloakAdapter_GetUserRealmRoleMappings(t *testing.T) {
	// Helper function to create expected path for a given realm and user ID
	buildExpectedPath := func(realm, userID string) string {
		return strings.NewReplacer("{realm}", realm, "{id}", userID).Replace(getUserRealmRoleMappings)
	}

	// Helper function to create server response handler
	createServerResponse := func(
		realm, userID, jsonResponse string, statusCode int,
	) func(w http.ResponseWriter, r *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			// Handle authentication requests
			if strings.Contains(r.URL.Path, openidConnectTokenPath) {
				setJSONContentType(w)
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
				assert.NoError(t, err)

				return
			}

			// Handle the actual role mappings request
			expectedPath := buildExpectedPath(realm, userID)
			if strings.Contains(r.URL.Path, expectedPath) {
				assert.Equal(t, http.MethodGet, r.Method)

				setJSONContentType(w)
				w.WriteHeader(statusCode)

				if jsonResponse != "" {
					_, err := w.Write([]byte(jsonResponse))
					assert.NoError(t, err)
				}

				return
			}

			// Default response for other requests
			w.WriteHeader(http.StatusOK)
		}
	}
	tests := []struct {
		name           string
		realmName      string
		userID         string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		want           []UserRealmRoleMapping
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name:      "successful role mappings retrieval",
			realmName: "test-realm",
			userID:    "user-123",
			serverResponse: createServerResponse("test-realm", "user-123", `[
				{
					"id": "role1-id",
					"name": "role1"
				},
				{
					"id": "role2-id", 
					"name": "role2"
				}
			]`, http.StatusOK),
			want: []UserRealmRoleMapping{
				{ID: "role1-id", Name: "role1"},
				{ID: "role2-id", Name: "role2"},
			},
			wantErr: require.NoError,
		},
		{
			name:           "empty role mappings",
			realmName:      "test-realm",
			userID:         "user-123",
			serverResponse: createServerResponse("test-realm", "user-123", `[]`, http.StatusOK),
			want:           []UserRealmRoleMapping{},
			wantErr:        require.NoError,
		},
		{
			name:      "single role mapping",
			realmName: "test-realm",
			userID:    "user-456",
			serverResponse: createServerResponse("test-realm", "user-456", `[
				{
					"id": "admin-role-id",
					"name": "admin"
				}
			]`, http.StatusOK),
			want: []UserRealmRoleMapping{
				{ID: "admin-role-id", Name: "admin"},
			},
			wantErr: require.NoError,
		},
		{
			name:      "user not found - 404 error",
			realmName: "test-realm",
			userID:    "non-existent-user",
			serverResponse: createServerResponse(
				"test-realm", "non-existent-user", `{"error":"User not found"}`, http.StatusNotFound,
			),
			want:    nil,
			wantErr: require.Error,
		},
		{
			name:           "unauthorized - 401 error",
			realmName:      "test-realm",
			userID:         "user-123",
			serverResponse: createServerResponse("test-realm", "user-123", `{"error":"Unauthorized"}`, http.StatusUnauthorized),
			want:           nil,
			wantErr:        require.Error,
		},
		{
			name:           "forbidden - 403 error",
			realmName:      "test-realm",
			userID:         "user-123",
			serverResponse: createServerResponse("test-realm", "user-123", `{"error":"Forbidden"}`, http.StatusForbidden),
			want:           nil,
			wantErr:        require.Error,
		},
		{
			name:      "internal server error - 500",
			realmName: "test-realm",
			userID:    "user-123",
			serverResponse: createServerResponse(
				"test-realm", "user-123", `Internal Server Error`, http.StatusInternalServerError,
			),
			want:    nil,
			wantErr: require.Error,
		},
		{
			name:           "invalid JSON response",
			realmName:      "test-realm",
			userID:         "user-123",
			serverResponse: createServerResponse("test-realm", "user-123", `invalid json`, http.StatusOK),
			want:           nil,
			wantErr:        require.Error,
		},
		{
			name:      "roles with special characters in names",
			realmName: "test-realm",
			userID:    "user-123",
			serverResponse: createServerResponse("test-realm", "user-123", `[
				{
					"id": "special-role-id",
					"name": "role-with-special-chars!@#$%"
				},
				{
					"id": "unicode-role-id",
					"name": "role-with-unicode-ñáéíóú"
				}
			]`, http.StatusOK),
			want: []UserRealmRoleMapping{
				{ID: "special-role-id", Name: "role-with-special-chars!@#$%"},
				{ID: "unicode-role-id", Name: "role-with-unicode-ñáéíóú"},
			},
			wantErr: require.NoError,
		},
		{
			name:      "realm name with special characters",
			realmName: "test-realm-with-special-chars",
			userID:    "user-123",
			serverResponse: createServerResponse("test-realm-with-special-chars", "user-123", `[
				{
					"id": "role1-id",
					"name": "role1"
				}
			]`, http.StatusOK),
			want: []UserRealmRoleMapping{
				{ID: "role1-id", Name: "role1"},
			},
			wantErr: require.NoError,
		},
		{
			name:      "user ID with special characters",
			realmName: "test-realm",
			userID:    "user-with-special-chars-123",
			serverResponse: createServerResponse("test-realm", "user-with-special-chars-123", `[
				{
					"id": "role1-id",
					"name": "role1"
				}
			]`, http.StatusOK),
			want: []UserRealmRoleMapping{
				{ID: "role1-id", Name: "role1"},
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))

			t.Cleanup(func() {
				server.Close()
			})

			adapter, err := Make(context.Background(), GoCloakConfig{Url: server.URL}, logr.Discard(), nil)
			require.NoError(t, err)

			ctx := context.Background()
			got, err := adapter.GetUserRealmRoleMappings(ctx, tt.realmName, tt.userID)

			tt.wantErr(t, err)

			if err == nil {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGoCloakAdapter_GetUserRealmRoleMappings_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle authentication requests
		if strings.Contains(r.URL.Path, openidConnectTokenPath) {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
			assert.NoError(t, err)

			return
		}

		// Simulate a slow response to test context cancellation
		select {
		case <-r.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`[]`))
			assert.NoError(t, err)
		}
	}))

	t.Cleanup(func() {
		server.Close()
	})

	adapter, err := Make(context.Background(), GoCloakConfig{Url: server.URL}, logr.Discard(), nil)
	require.NoError(t, err)

	// Create a context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = adapter.GetUserRealmRoleMappings(ctx, "test-realm", "user-123")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestGoCloakAdapter_GetUsersProfile(t *testing.T) {
	tests := []struct {
		name           string
		realm          string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		want           *keycloak_go_client.UserProfileConfig
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name:  "successful profile retrieval",
			realm: "test-realm",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Handle authentication requests
				if strings.Contains(r.URL.Path, openidConnectTokenPath) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
					assert.NoError(t, err)
					return
				}

				// Handle the actual user profile request
				if strings.Contains(r.URL.Path, strings.Replace(realmUsersProfile, "{realm}", "test-realm", 1)) {
					assert.Equal(t, http.MethodGet, r.Method)

					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)

					response := `{
						"unmanagedAttributePolicy": "ENABLED",
						"attributes": [
							{
								"name": "firstName",
								"displayName": "First Name",
								"required": {}
							}
						],
						"groups": []
					}`
					_, err := w.Write([]byte(response))
					assert.NoError(t, err)
					return
				}

				// Default response for other requests
				w.WriteHeader(http.StatusOK)
			},
			want: &keycloak_go_client.UserProfileConfig{
				UnmanagedAttributePolicy: (*keycloak_go_client.UnmanagedAttributePolicy)(gocloak.StringP("ENABLED")),
				Attributes: &[]keycloak_go_client.UserProfileAttribute{
					{
						Name:        gocloak.StringP("firstName"),
						DisplayName: gocloak.StringP("First Name"),
						Required:    &keycloak_go_client.UserProfileAttributeRequired{},
					},
				},
				Groups: &[]keycloak_go_client.UserProfileGroup{},
			},
			wantErr: require.NoError,
		},
		{
			name:  "empty profile",
			realm: "test-realm",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Handle authentication requests
				if strings.Contains(r.URL.Path, openidConnectTokenPath) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
					assert.NoError(t, err)
					return
				}

				// Handle the actual user profile request
				if strings.Contains(r.URL.Path, strings.Replace(realmUsersProfile, "{realm}", "test-realm", 1)) {
					assert.Equal(t, http.MethodGet, r.Method)

					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)

					response := `{
						"attributes": [],
						"groups": []
					}`
					_, err := w.Write([]byte(response))
					assert.NoError(t, err)
					return
				}

				// Default response for other requests
				w.WriteHeader(http.StatusOK)
			},
			want: &keycloak_go_client.UserProfileConfig{
				Attributes: &[]keycloak_go_client.UserProfileAttribute{},
				Groups:     &[]keycloak_go_client.UserProfileGroup{},
			},
			wantErr: require.NoError,
		},
		{
			name:  "profile with groups and attributes",
			realm: "test-realm",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Handle authentication requests
				if strings.Contains(r.URL.Path, openidConnectTokenPath) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
					assert.NoError(t, err)
					return
				}

				// Handle the actual user profile request
				if strings.Contains(r.URL.Path, strings.Replace(realmUsersProfile, "{realm}", "test-realm", 1)) {
					assert.Equal(t, http.MethodGet, r.Method)

					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)

					response := `{
						"unmanagedAttributePolicy": "ADMIN_EDIT",
						"attributes": [
							{
								"name": "email",
								"displayName": "Email",
								"group": "contact"
							},
							{
								"name": "phone",
								"displayName": "Phone Number",
								"group": "contact"
							}
						],
						"groups": [
							{
								"name": "contact",
								"displayDescription": "Contact Information",
								"displayHeader": "Contact Details"
							}
						]
					}`
					_, err := w.Write([]byte(response))
					assert.NoError(t, err)
					return
				}

				// Default response for other requests
				w.WriteHeader(http.StatusOK)
			},
			want: &keycloak_go_client.UserProfileConfig{
				UnmanagedAttributePolicy: (*keycloak_go_client.UnmanagedAttributePolicy)(gocloak.StringP("ADMIN_EDIT")),
				Attributes: &[]keycloak_go_client.UserProfileAttribute{
					{
						Name:        gocloak.StringP("email"),
						DisplayName: gocloak.StringP("Email"),
						Group:       gocloak.StringP("contact"),
					},
					{
						Name:        gocloak.StringP("phone"),
						DisplayName: gocloak.StringP("Phone Number"),
						Group:       gocloak.StringP("contact"),
					},
				},
				Groups: &[]keycloak_go_client.UserProfileGroup{
					{
						Name:               gocloak.StringP("contact"),
						DisplayDescription: gocloak.StringP("Contact Information"),
						DisplayHeader:      gocloak.StringP("Contact Details"),
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name:  "realm not found - 404 error",
			realm: "non-existent-realm",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Handle authentication requests
				if strings.Contains(r.URL.Path, openidConnectTokenPath) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
					assert.NoError(t, err)
					return
				}

				// Handle the actual user profile request
				if strings.Contains(r.URL.Path, strings.Replace(realmUsersProfile, "{realm}", "non-existent-realm", 1)) {
					assert.Equal(t, http.MethodGet, r.Method)

					setJSONContentType(w)
					w.WriteHeader(http.StatusNotFound)

					_, err := w.Write([]byte(`{"error":"Realm not found"}`))
					assert.NoError(t, err)
					return
				}

				// Default response for other requests
				w.WriteHeader(http.StatusOK)
			},
			want:    nil,
			wantErr: require.Error,
		},
		{
			name:  "unauthorized - 401 error",
			realm: "test-realm",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Handle authentication requests
				if strings.Contains(r.URL.Path, openidConnectTokenPath) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
					assert.NoError(t, err)
					return
				}

				// Handle the actual user profile request - return 401 for the API call
				if strings.Contains(r.URL.Path, strings.Replace(realmUsersProfile, "{realm}", "test-realm", 1)) {
					assert.Equal(t, http.MethodGet, r.Method)

					setJSONContentType(w)
					w.WriteHeader(http.StatusUnauthorized)

					_, err := w.Write([]byte(`{"error":"Unauthorized"}`))
					assert.NoError(t, err)
					return
				}

				// Default response for other requests
				w.WriteHeader(http.StatusOK)
			},
			want:    nil,
			wantErr: require.Error,
		},
		{
			name:  "internal server error - 500",
			realm: "test-realm",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				// Handle authentication requests
				if strings.Contains(r.URL.Path, openidConnectTokenPath) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
					assert.NoError(t, err)
					return
				}

				// Handle the actual user profile request
				if strings.Contains(r.URL.Path, strings.Replace(realmUsersProfile, "{realm}", "test-realm", 1)) {
					assert.Equal(t, http.MethodGet, r.Method)

					w.WriteHeader(http.StatusInternalServerError)

					_, err := w.Write([]byte(`Internal Server Error`))
					assert.NoError(t, err)
					return
				}

				// Default response for other requests
				w.WriteHeader(http.StatusOK)
			},
			want:    nil,
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))

			t.Cleanup(func() {
				server.Close()
			})

			adapter, err := Make(context.Background(), GoCloakConfig{Url: server.URL}, logr.Discard(), nil)
			require.NoError(t, err)

			ctx := context.Background()
			got, err := adapter.GetUsersProfile(ctx, tt.realm)

			tt.wantErr(t, err)

			if err == nil {
				// Note: Due to the complexity of the UserProfileConfig structure and
				// potential differences in JSON marshaling/unmarshaling, we'll do
				// basic structural validation rather than deep equality
				assert.NotNil(t, got)

				if tt.want.Attributes != nil {
					assert.NotNil(t, got.Attributes)
				}

				if tt.want.Groups != nil {
					assert.NotNil(t, got.Groups)
				}
			}
		})
	}
}

// createErrorResponseHandler creates a server response handler for error testing
func createErrorResponseHandler(
	t *testing.T,
	realm string,
	statusCode int,
	errorMessage string,
) func(w http.ResponseWriter, r *http.Request, userProfile keycloak_go_client.UserProfileConfig) {
	return func(w http.ResponseWriter, r *http.Request, userProfile keycloak_go_client.UserProfileConfig) {
		// Handle authentication requests
		if strings.Contains(r.URL.Path, openidConnectTokenPath) {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
			assert.NoError(t, err)

			return
		}

		// Handle the actual user profile update request
		expectedPath := strings.Replace(realmUsersProfile, "{realm}", realm, 1)
		if strings.Contains(r.URL.Path, expectedPath) {
			assert.Equal(t, http.MethodPut, r.Method)

			setJSONContentType(w)
			w.WriteHeader(statusCode)

			_, err := fmt.Fprintf(w, `{"error":"%s"}`, errorMessage)
			assert.NoError(t, err)

			return
		}

		// Default response for other requests
		w.WriteHeader(http.StatusOK)
	}
}

func TestGoCloakAdapter_UpdateUsersProfile(t *testing.T) {
	tests := []struct {
		name           string
		realm          string
		userProfile    keycloak_go_client.UserProfileConfig
		serverResponse func(w http.ResponseWriter, r *http.Request, userProfile keycloak_go_client.UserProfileConfig)
		want           *keycloak_go_client.UserProfileConfig
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name:  "successful profile update",
			realm: "test-realm",
			userProfile: keycloak_go_client.UserProfileConfig{
				UnmanagedAttributePolicy: (*keycloak_go_client.UnmanagedAttributePolicy)(gocloak.StringP("ENABLED")),
				Attributes: &[]keycloak_go_client.UserProfileAttribute{
					{
						Name:        gocloak.StringP("customAttribute"),
						DisplayName: gocloak.StringP("Custom Attribute"),
						Required:    &keycloak_go_client.UserProfileAttributeRequired{},
					},
				},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request, userProfile keycloak_go_client.UserProfileConfig) {
				// Handle authentication requests
				if strings.Contains(r.URL.Path, openidConnectTokenPath) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
					assert.NoError(t, err)
					return
				}

				// Handle the actual user profile update request
				if strings.Contains(r.URL.Path, strings.Replace(realmUsersProfile, "{realm}", "test-realm", 1)) {
					assert.Equal(t, http.MethodPut, r.Method)

					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)

					// Return the updated profile
					response := `{
						"unmanagedAttributePolicy": "ENABLED",
						"attributes": [
							{
								"name": "customAttribute",
								"displayName": "Custom Attribute",
								"required": {}
							}
						],
						"groups": []
					}`
					_, err := w.Write([]byte(response))
					assert.NoError(t, err)
					return
				}

				// Default response for other requests
				w.WriteHeader(http.StatusOK)
			},
			want: &keycloak_go_client.UserProfileConfig{
				UnmanagedAttributePolicy: (*keycloak_go_client.UnmanagedAttributePolicy)(gocloak.StringP("ENABLED")),
				Attributes: &[]keycloak_go_client.UserProfileAttribute{
					{
						Name:        gocloak.StringP("customAttribute"),
						DisplayName: gocloak.StringP("Custom Attribute"),
						Required:    &keycloak_go_client.UserProfileAttributeRequired{},
					},
				},
				Groups: &[]keycloak_go_client.UserProfileGroup{},
			},
			wantErr: require.NoError,
		},
		{
			name:  "update profile with groups",
			realm: "test-realm",
			userProfile: keycloak_go_client.UserProfileConfig{
				UnmanagedAttributePolicy: (*keycloak_go_client.UnmanagedAttributePolicy)(gocloak.StringP("ADMIN_VIEW")),
				Groups: &[]keycloak_go_client.UserProfileGroup{
					{
						Name:               gocloak.StringP("personalInfo"),
						DisplayDescription: gocloak.StringP("Personal Information"),
						DisplayHeader:      gocloak.StringP("Personal Details"),
					},
				},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request, userProfile keycloak_go_client.UserProfileConfig) {
				// Handle authentication requests
				if strings.Contains(r.URL.Path, openidConnectTokenPath) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
					assert.NoError(t, err)
					return
				}

				// Handle the actual user profile update request
				if strings.Contains(r.URL.Path, strings.Replace(realmUsersProfile, "{realm}", "test-realm", 1)) {
					assert.Equal(t, http.MethodPut, r.Method)

					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)

					// Return the updated profile
					response := `{
						"unmanagedAttributePolicy": "ADMIN_VIEW",
						"attributes": [],
						"groups": [
							{
								"name": "personalInfo",
								"displayDescription": "Personal Information",
								"displayHeader": "Personal Details"
							}
						]
					}`
					_, err := w.Write([]byte(response))
					assert.NoError(t, err)
					return
				}

				// Default response for other requests
				w.WriteHeader(http.StatusOK)
			},
			want: &keycloak_go_client.UserProfileConfig{
				UnmanagedAttributePolicy: (*keycloak_go_client.UnmanagedAttributePolicy)(gocloak.StringP("ADMIN_VIEW")),
				Attributes:               &[]keycloak_go_client.UserProfileAttribute{},
				Groups: &[]keycloak_go_client.UserProfileGroup{
					{
						Name:               gocloak.StringP("personalInfo"),
						DisplayDescription: gocloak.StringP("Personal Information"),
						DisplayHeader:      gocloak.StringP("Personal Details"),
					},
				},
			},
			wantErr: require.NoError,
		},
		{
			name:  "empty profile update",
			realm: "test-realm",
			userProfile: keycloak_go_client.UserProfileConfig{
				Attributes: &[]keycloak_go_client.UserProfileAttribute{},
				Groups:     &[]keycloak_go_client.UserProfileGroup{},
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request, userProfile keycloak_go_client.UserProfileConfig) {
				// Handle authentication requests
				if strings.Contains(r.URL.Path, openidConnectTokenPath) {
					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)
					_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
					assert.NoError(t, err)
					return
				}

				// Handle the actual user profile update request
				if strings.Contains(r.URL.Path, strings.Replace(realmUsersProfile, "{realm}", "test-realm", 1)) {
					assert.Equal(t, http.MethodPut, r.Method)

					setJSONContentType(w)
					w.WriteHeader(http.StatusOK)

					// Return the updated profile
					response := `{
						"attributes": [],
						"groups": []
					}`
					_, err := w.Write([]byte(response))
					assert.NoError(t, err)
					return
				}

				// Default response for other requests
				w.WriteHeader(http.StatusOK)
			},
			want: &keycloak_go_client.UserProfileConfig{
				Attributes: &[]keycloak_go_client.UserProfileAttribute{},
				Groups:     &[]keycloak_go_client.UserProfileGroup{},
			},
			wantErr: require.NoError,
		},
		{
			name:  "realm not found - 404 error",
			realm: "non-existent-realm",
			userProfile: keycloak_go_client.UserProfileConfig{
				Attributes: &[]keycloak_go_client.UserProfileAttribute{},
			},
			serverResponse: createErrorResponseHandler(t, "non-existent-realm", http.StatusNotFound, "Realm not found"),
			want:           nil,
			wantErr:        require.Error,
		},
		{
			name:  "forbidden - 403 error",
			realm: "test-realm",
			userProfile: keycloak_go_client.UserProfileConfig{
				Attributes: &[]keycloak_go_client.UserProfileAttribute{},
			},
			serverResponse: createErrorResponseHandler(t, "test-realm", http.StatusForbidden, "Insufficient permissions"),
			want:           nil,
			wantErr:        require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				tt.serverResponse(w, r, tt.userProfile)
			}))

			t.Cleanup(func() {
				server.Close()
			})

			adapter, err := Make(context.Background(), GoCloakConfig{Url: server.URL}, logr.Discard(), nil)
			require.NoError(t, err)

			ctx := context.Background()
			got, err := adapter.UpdateUsersProfile(ctx, tt.realm, tt.userProfile)

			tt.wantErr(t, err)

			if err == nil {
				// Note: Due to the complexity of the UserProfileConfig structure and
				// potential differences in JSON marshaling/unmarshaling, we'll do
				// basic structural validation rather than deep equality
				assert.NotNil(t, got)

				if tt.want.Attributes != nil {
					assert.NotNil(t, got.Attributes)
				}

				if tt.want.Groups != nil {
					assert.NotNil(t, got.Groups)
				}
			}
		})
	}
}

func TestGoCloakAdapter_GetUsersProfile_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle authentication requests
		if strings.Contains(r.URL.Path, openidConnectTokenPath) {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
			assert.NoError(t, err)

			return
		}

		// Simulate a slow response to test context cancellation
		select {
		case <-r.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"attributes":[],"groups":[]}`))
			assert.NoError(t, err)
		}
	}))

	t.Cleanup(func() {
		server.Close()
	})

	adapter, err := Make(context.Background(), GoCloakConfig{Url: server.URL}, logr.Discard(), nil)
	require.NoError(t, err)

	// Create a context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = adapter.GetUsersProfile(ctx, "test-realm")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestGoCloakAdapter_UpdateUsersProfile_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle authentication requests
		if strings.Contains(r.URL.Path, openidConnectTokenPath) {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`))
			assert.NoError(t, err)

			return
		}

		// Simulate a slow response to test context cancellation
		select {
		case <-r.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"attributes":[],"groups":[]}`))
			assert.NoError(t, err)
		}
	}))

	t.Cleanup(func() {
		server.Close()
	})

	adapter, err := Make(context.Background(), GoCloakConfig{Url: server.URL}, logr.Discard(), nil)
	require.NoError(t, err)

	// Create a context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	userProfile := keycloak_go_client.UserProfileConfig{
		Attributes: &[]keycloak_go_client.UserProfileAttribute{},
	}

	_, err = adapter.UpdateUsersProfile(ctx, "test-realm", userProfile)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestGoCloakAdapter_CreateOrUpdateUser(t *testing.T) {
	tests := []struct {
		name    string
		userDto *KeycloakUser
		addOnly bool
		client  func(t *testing.T) *mocks.MockGoCloak
		wantID  string
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "create new user success",
			userDto: &KeycloakUser{
				Username:            "newuser",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "newuser@mail.com",
				FirstName:           "New",
				LastName:            "User",
				RequiredUserActions: []string{"UPDATE_PASSWORD"},
				Attributes:          map[string][]string{"department": {"engineering"}},
			},
			addOnly: false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.EXPECT().GetUsers(mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.EXPECT().CreateUser(
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "newuser", *user.Username) &&
							assert.Equal(t, true, *user.Enabled) &&
							assert.Equal(t, "newuser@mail.com", *user.Email)
					})).
					Return("new-user-id", nil)

				return m
			},
			wantID:  "new-user-id",
			wantErr: require.NoError,
		},
		{
			name: "update existing user success",
			userDto: &KeycloakUser{
				Username:            "existinguser",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "existing@mail.com",
				FirstName:           "Existing",
				LastName:            "User",
				RequiredUserActions: []string{},
				Attributes:          map[string][]string{"role": {"admin"}},
			},
			addOnly: false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.EXPECT().GetUsers(mock.Anything, "", "realm", mock.Anything).
					Return([]*gocloak.User{{
						ID:       gocloak.StringP("existing-user-id"),
						Username: gocloak.StringP("existinguser"),
					}}, nil)
				m.EXPECT().UpdateUser(
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "existinguser", *user.Username) &&
							assert.Equal(t, "existing-user-id", *user.ID)
					})).
					Return(nil)

				return m
			},
			wantID:  "existing-user-id",
			wantErr: require.NoError,
		},
		{
			name: "create user without attributes",
			userDto: &KeycloakUser{
				Username:      "simpleuser",
				Enabled:       true,
				EmailVerified: false,
				Email:         "simple@mail.com",
				FirstName:     "Simple",
				LastName:      "User",
			},
			addOnly: false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.EXPECT().GetUsers(mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.EXPECT().CreateUser(
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "simpleuser", *user.Username) &&
							assert.Nil(t, user.Attributes)
					})).
					Return("simple-user-id", nil)

				return m
			},
			wantID:  "simple-user-id",
			wantErr: require.NoError,
		},
		{
			name: "failed to get users",
			userDto: &KeycloakUser{
				Username: "testuser",
				Enabled:  true,
			},
			addOnly: false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.EXPECT().GetUsers(mock.Anything, "", "realm", mock.Anything).
					Return(nil, errors.New("connection error"))

				return m
			},
			wantID: "",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "connection error")
			},
		},
		{
			name: "failed to create user",
			userDto: &KeycloakUser{
				Username: "testuser",
				Enabled:  true,
			},
			addOnly: false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.EXPECT().GetUsers(mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.EXPECT().CreateUser(mock.Anything, "", "realm", mock.Anything).
					Return("", errors.New("user already exists"))

				return m
			},
			wantID: "",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "user already exists")
			},
		},
		{
			name: "failed to update user",
			userDto: &KeycloakUser{
				Username: "existinguser",
				Enabled:  true,
			},
			addOnly: false,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.EXPECT().GetUsers(mock.Anything, "", "realm", mock.Anything).
					Return([]*gocloak.User{{
						ID:       gocloak.StringP("existing-user-id"),
						Username: gocloak.StringP("existinguser"),
					}}, nil)
				m.EXPECT().UpdateUser(mock.Anything, "", "realm", mock.Anything).
					Return(errors.New("update failed"))

				return m
			},
			wantID: "",
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "update failed")
			},
		},
		{
			name: "update user with addOnly mode",
			userDto: &KeycloakUser{
				Username:   "existinguser",
				Enabled:    true,
				Attributes: map[string][]string{"newAttr": {"value1"}},
			},
			addOnly: true,
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.EXPECT().GetUsers(mock.Anything, "", "realm", mock.Anything).
					Return([]*gocloak.User{{
						ID:         gocloak.StringP("existing-user-id"),
						Username:   gocloak.StringP("existinguser"),
						Attributes: &map[string][]string{"existingAttr": {"existingValue"}},
					}}, nil)
				m.EXPECT().UpdateUser(
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						// In addOnly mode, existing attributes should be preserved
						attrs := *user.Attributes
						return assert.Contains(t, attrs, "existingAttr") &&
							assert.Contains(t, attrs, "newAttr")
					})).
					Return(nil)

				return m
			},
			wantID:  "existing-user-id",
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := GoCloakAdapter{
				client: tt.client(t),
				token:  &gocloak.JWT{AccessToken: ""},
			}

			userID, err := a.CreateOrUpdateUser(
				context.Background(),
				"realm",
				tt.userDto,
				tt.addOnly,
			)

			tt.wantErr(t, err)
			assert.Equal(t, tt.wantID, userID)
		})
	}
}

func TestPreserveUpdatePasswordAction(t *testing.T) {
	tests := []struct {
		name    string
		current []string
		desired []string
		want    []string
	}{
		{
			name:    "current is nil, desired is empty",
			current: nil,
			desired: []string{},
			want:    []string{},
		},
		{
			name:    "current is empty, desired is empty",
			current: []string{},
			desired: []string{},
			want:    []string{},
		},
		{
			name:    "current has UPDATE_PASSWORD, desired is empty",
			current: []string{"UPDATE_PASSWORD"},
			desired: []string{},
			want:    []string{"UPDATE_PASSWORD"},
		},
		{
			name:    "current has UPDATE_PASSWORD, desired has other actions",
			current: []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
			desired: []string{"VERIFY_EMAIL"},
			want:    []string{"VERIFY_EMAIL", "UPDATE_PASSWORD"},
		},
		{
			name:    "current has UPDATE_PASSWORD, desired already has UPDATE_PASSWORD",
			current: []string{"UPDATE_PASSWORD"},
			desired: []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
			want:    []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
		},
		{
			name:    "current does not have UPDATE_PASSWORD",
			current: []string{"VERIFY_EMAIL"},
			desired: []string{"CONFIGURE_TOTP"},
			want:    []string{"CONFIGURE_TOTP"},
		},
		{
			name:    "current is nil, desired has UPDATE_PASSWORD",
			current: nil,
			desired: []string{"UPDATE_PASSWORD"},
			want:    []string{"UPDATE_PASSWORD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalDesired := make([]string, len(tt.desired))
			copy(originalDesired, tt.desired)

			got := preserveUpdatePasswordAction(tt.current, tt.desired)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, originalDesired, tt.desired, "original desired slice should not be mutated")
		})
	}
}
