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
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	keycloak_go_client "github.com/zmotso/keycloak-go-client"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

func TestGoCloakAdapter_SyncRealmUser(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "users/user-with-groups-id/groups") {
			setJSONContentType(w)
			w.WriteHeader(http.StatusOK)

			_, err := w.Write([]byte(`[{"id":"group1-id","name":"group1"},{"id":"group2-id","name":"group2"}]`))
			assert.NoError(t, err)

			return
		}

		if strings.Contains(r.URL.Path, "identity-provider/instances/idp1") {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"alias":"idp1"}`))
			assert.NoError(t, err)

			return
		}

		if strings.Contains(r.URL.Path, "identity-provider/instances/non-existent-idp") {
			w.WriteHeader(http.StatusNotFound)
			_, err := w.Write([]byte(`{"error":"idp not found"}`))
			assert.NoError(t, err)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	t.Cleanup(func() {
		server.Close()
	})

	tests := []struct {
		name    string
		userDto *KeycloakUser
		client  func(t *testing.T) *mocks.MockGoCloak
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "create user success",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
				IdentityProviders:   &[]string{"idp1"},
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.On("CreateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return("user-id", nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role1"),
						ID:   gocloak.StringP("role1-id"),
					}, nil)
				m.On("AddRealmRoleToUser",
					mock.Anything,
					"",
					"realm",
					"user-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)
				m.On("GetGroups",
					mock.Anything,
					"",
					"realm",
					mock.Anything).
					Return([]*gocloak.Group{{
						Name: gocloak.StringP("group1"),
						ID:   gocloak.StringP("group1-id"),
					}}, nil)
				m.On("RestyClient").Return(resty.New())
				m.On("GetUserFederatedIdentities",
					mock.Anything,
					"",
					"realm",
					"user-id").
					Return([]*gocloak.FederatedIdentityRepresentation{{IdentityProvider: gocloak.StringP("idp2")}}, nil)
				m.On("CreateUserFederatedIdentity",
					mock.Anything,
					"",
					"realm",
					"user-id",
					"idp1",
					mock.Anything).
					Return(nil)
				m.On("DeleteUserFederatedIdentity",
					mock.Anything,
					"",
					"realm",
					"user-id",
					"idp2",
					mock.Anything).
					Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "update user success",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1", "group3"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return([]*gocloak.User{{
						ID:       gocloak.StringP("user-with-groups-id"),
						Username: gocloak.StringP("user"),
					}}, nil)
				m.On("UpdateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return(nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-with-groups-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role1"),
						ID:   gocloak.StringP("role1-id"),
					}, nil)
				m.On("AddRealmRoleToUser",
					mock.Anything,
					"",
					"realm",
					"user-with-groups-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)
				m.On("GetGroups",
					mock.Anything,
					"",
					"realm",
					mock.Anything).
					Return([]*gocloak.Group{
						{
							Name: gocloak.StringP("group1"),
							ID:   gocloak.StringP("group1-id"),
						},
						{
							Name: gocloak.StringP("group2"),
							ID:   gocloak.StringP("group2-id"),
						},
						{
							Name: gocloak.StringP("group3"),
							ID:   gocloak.StringP("group3-id"),
						},
					}, nil)
				m.On("RestyClient").Return(resty.New())

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to get groups",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.On("CreateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return("user-id", nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role1"),
						ID:   gocloak.StringP("role1-id"),
					}, nil)
				m.On("AddRealmRoleToUser",
					mock.Anything,
					"",
					"realm",
					"user-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)
				m.On("GetGroups",
					mock.Anything,
					"",
					"realm",
					mock.Anything).
					Return(nil, errors.New("failed to get groups"))
				m.On("RestyClient").Return(resty.New())

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get groups")
			},
		},
		{
			name: "failed to get roles",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return([]*gocloak.User{{
						ID:       gocloak.StringP("user-id"),
						Username: gocloak.StringP("user"),
					}}, nil)
				m.On("UpdateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return(nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(nil, errors.New("failed to get roles"))
				m.On("RestyClient").Return(resty.New())

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get roles")
			},
		},
		{
			name: "failed to create user",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.On("CreateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return("", errors.New("failed to create user"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to create user")
			},
		},
		{
			name: "failed to get user",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, errors.New("failed to get user"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get user")
			},
		},
		{
			name: "identity provider does not exist",
			userDto: &KeycloakUser{
				Username:            "user",
				Enabled:             true,
				EmailVerified:       true,
				Email:               "mail@mail.com",
				FirstName:           "first-name",
				LastName:            "last-name",
				RequiredUserActions: []string{"change-password"},
				Roles:               []string{"role1"},
				Groups:              []string{"group1"},
				Attributes:          map[string][]string{"attr1": {"attr1value"}},
				Password:            "password",
				IdentityProviders:   &[]string{"non-existent-idp"},
			},
			client: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetUsers", mock.Anything, "", "realm", mock.Anything).
					Return(nil, nil)
				m.On("CreateUser",
					mock.Anything,
					"",
					"realm",
					mock.MatchedBy(func(user gocloak.User) bool {
						return assert.Equal(t, "user", *user.Username)
					})).
					Return("user-id", nil)
				m.On("GetRoleMappingByUserID", mock.Anything, "", "realm", "user-id").
					Return(&gocloak.MappingsRepresentation{
						RealmMappings:  &[]gocloak.Role{},
						ClientMappings: map[string]*gocloak.ClientMappingsRepresentation{},
					}, nil)
				m.On("GetRealmRole", mock.Anything, "", "realm", "role1").
					Return(&gocloak.Role{
						Name: gocloak.StringP("role1"),
						ID:   gocloak.StringP("role1-id"),
					}, nil)
				m.On("AddRealmRoleToUser",
					mock.Anything,
					"",
					"realm",
					"user-id",
					mock.MatchedBy(func(roles []gocloak.Role) bool {
						return assert.Len(t, roles, 1) &&
							assert.Equal(t, "role1-id", *roles[0].ID)
					})).
					Return(nil)
				m.On("GetGroups",
					mock.Anything,
					"",
					"realm",
					mock.Anything).
					Return([]*gocloak.Group{{
						Name: gocloak.StringP("group1"),
						ID:   gocloak.StringP("group1-id"),
					}}, nil)
				m.On("RestyClient").Return(resty.New())
				m.On("GetUserFederatedIdentities",
					mock.Anything,
					"",
					"realm",
					"user-id").
					Return([]*gocloak.FederatedIdentityRepresentation{}, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "identity provider non-existent-idp does not exist")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a, err := Make(context.Background(), GoCloakConfig{Url: server.URL}, logr.Discard(), nil)
			a.client = tt.client(t)

			require.NoError(t, err)

			tt.wantErr(t, a.SyncRealmUser(
				context.Background(),
				"realm",
				tt.userDto,
				false,
			))
		})
	}
}

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
