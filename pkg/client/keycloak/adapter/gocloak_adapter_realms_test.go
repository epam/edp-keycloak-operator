package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/common"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

func TestGoCloakAdapter_UpdateRealmSettings(t *testing.T) {
	adapter, mockClient, _ := initAdapter(t)

	settings := RealmSettings{
		Themes: &RealmThemes{
			LoginTheme: gocloak.StringP("keycloak"),
		},
		BrowserSecurityHeaders: &map[string]string{
			"foo": "bar",
		},
		PasswordPolicies: []PasswordPolicy{
			{Type: "foo", Value: "bar"},
			{Type: "bar", Value: "baz"},
		},
		FrontendURL: "https://google.com",
		TokenSettings: &TokenSettings{
			DefaultSignatureAlgorithm:           "RS256",
			RevokeRefreshToken:                  true,
			RefreshTokenMaxReuse:                230,
			AccessTokenLifespan:                 231,
			AccessTokenLifespanForImplicitFlow:  232,
			AccessCodeLifespan:                  233,
			ActionTokenGeneratedByUserLifespan:  234,
			ActionTokenGeneratedByAdminLifespan: 235,
		},
		AdminEventsExpiration: ptr.To(100),
	}
	realmName := "ream11"

	realm := gocloak.RealmRepresentation{
		BrowserSecurityHeaders: &map[string]string{
			"test": "dets",
		},
	}
	mockClient.On("GetRealm", mock.Anything, adapter.token.AccessToken, realmName).Return(&realm, nil)
	mockClient.On("UpdateRealm", mock.Anything, "token", mock.MatchedBy(func(realm gocloak.RealmRepresentation) bool {
		return assert.Equal(t, settings.Themes.LoginTheme, realm.LoginTheme) &&
			assert.Equal(t, &map[string]string{
				"test": "dets",
				"foo":  "bar",
			}, realm.BrowserSecurityHeaders) &&
			assert.Equal(t, gocloak.StringP("foo(bar) and bar(baz)"), realm.PasswordPolicy) &&
			assert.Equal(t, &map[string]string{
				"frontendUrl":           settings.FrontendURL,
				"adminEventsExpiration": "100",
			}, realm.Attributes) &&
			assert.Equal(
				t,
				settings.TokenSettings.DefaultSignatureAlgorithm,
				*realm.DefaultSignatureAlgorithm,
			) &&
			assert.Equal(
				t,
				settings.TokenSettings.RevokeRefreshToken,
				*realm.RevokeRefreshToken,
			) &&
			assert.Equal(
				t,
				settings.TokenSettings.RefreshTokenMaxReuse,
				*realm.RefreshTokenMaxReuse,
			) &&
			assert.Equal(
				t,
				settings.TokenSettings.AccessTokenLifespan,
				*realm.AccessTokenLifespan,
			) &&
			assert.Equal(
				t,
				settings.TokenSettings.AccessTokenLifespanForImplicitFlow,
				*realm.AccessTokenLifespanForImplicitFlow,
			) &&
			assert.Equal(
				t,
				settings.TokenSettings.AccessCodeLifespan,
				*realm.AccessCodeLifespan,
			) &&
			assert.Equal(
				t,
				settings.TokenSettings.ActionTokenGeneratedByUserLifespan,
				*realm.ActionTokenGeneratedByUserLifespan,
			) &&
			assert.Equal(
				t,
				settings.TokenSettings.ActionTokenGeneratedByAdminLifespan,
				*realm.ActionTokenGeneratedByAdminLifespan,
			)
	})).Return(nil)

	err := adapter.UpdateRealmSettings(realmName, &settings)
	require.NoError(t, err)
}

func TestGoCloakAdapter_SyncRealmIdentityProviderMappers(t *testing.T) {
	adapter, mockClient, restyClient := initAdapter(t)
	httpmock.ActivateNonDefault(restyClient.GetClient())

	currentMapperID := "mp1id"

	mappers := []interface{}{
		map[string]interface{}{
			keycloakApiParamId: currentMapperID,
			"name":             "mp1name",
		},
	}

	realm := gocloak.RealmRepresentation{
		Realm:                   gocloak.StringP("sso-realm-1"),
		IdentityProviderMappers: &mappers,
	}

	idpAlias := "alias-1"

	mockClient.On("GetRealm", mock.Anything, adapter.token.AccessToken, *realm.Realm).Return(&realm, nil)

	httpmock.RegisterResponder(
		"POST",
		fmt.Sprintf("/admin/realms/%s/identity-provider/instances/%s/mappers", *realm.Realm, idpAlias),
		httpmock.NewStringResponder(http.StatusCreated, "ok"))

	httpmock.RegisterResponder(
		"PUT",
		fmt.Sprintf("/admin/realms/%s/identity-provider/instances/%s/mappers/%s", *realm.Realm, idpAlias,
			currentMapperID),
		httpmock.NewStringResponder(http.StatusOK, "ok"))

	if err := adapter.SyncRealmIdentityProviderMappers(*realm.Realm,
		[]dto.IdentityProviderMapper{
			{
				Name:                   "tname1",
				Config:                 map[string]string{"foo": "bar"},
				IdentityProviderMapper: "mapper-1",
				IdentityProviderAlias:  idpAlias,
			},
			{
				Name:                   "mp1name",
				Config:                 map[string]string{"foo": "bar"},
				IdentityProviderMapper: "mapper-2",
				IdentityProviderAlias:  idpAlias,
			},
		}); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestGoCloakAdapter_CreateRealmWithDefaultConfig(t *testing.T) {
	adapter, mockClient, _ := initAdapter(t)
	r := dto.Realm{}

	mockClient.On("CreateRealm", mock.Anything, "token", getDefaultRealm(&r)).Return("id1", nil).Once()
	err := adapter.CreateRealmWithDefaultConfig(&r)
	require.NoError(t, err)

	mockClient.On("CreateRealm", mock.Anything, "token", getDefaultRealm(&r)).Return("",
		errors.New("create realm fatal")).Once()

	err = adapter.CreateRealmWithDefaultConfig(&r)
	require.Error(t, err)

	if err.Error() != "unable to create realm: create realm fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_DeleteRealm(t *testing.T) {
	adapter, mockClient, _ := initAdapter(t)

	mockClient.On("DeleteRealm", mock.Anything, "token", "test-realm1").Return(nil).Once()

	err := adapter.DeleteRealm(context.Background(), "test-realm1")
	require.NoError(t, err)

	mockClient.On("DeleteRealm", mock.Anything, "token", "test-realm2").Return(errors.New("delete fatal")).Once()

	err = adapter.DeleteRealm(context.Background(), "test-realm2")
	require.Error(t, err)

	if err.Error() != "unable to delete realm: delete fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_GetRealm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		client  func(t *testing.T) GoCloak
		want    *gocloak.RealmRepresentation
		wantErr require.ErrorAssertionFunc
	}{
		{
			name: "realm exists",
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetRealm", mock.Anything, "token", mock.Anything, mock.Anything).
					Return(&gocloak.RealmRepresentation{
						ID: gocloak.StringP("realmId"),
					}, nil)

				return m
			},
			want: &gocloak.RealmRepresentation{
				ID: gocloak.StringP("realmId"),
			},
			wantErr: require.NoError,
		},
		{
			name: "realm does not exist",
			client: func(t *testing.T) GoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetRealm", mock.Anything, "token", mock.Anything, mock.Anything).
					Return(nil, errors.New("realm not found"))

				return m
			},
			want: nil,
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "realm not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := GoCloakAdapter{
				client: tt.client(t),
				token:  &gocloak.JWT{AccessToken: "token"},
				log:    logr.Discard(),
			}
			got, err := a.GetRealm(ctrl.LoggerInto(context.Background(), logr.Discard()), "realmName")
			tt.wantErr(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToRealmTokenSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tokenSettings *common.TokenSettings
		want          *TokenSettings
	}{
		{
			name:          "nil",
			tokenSettings: nil,
			want:          nil,
		},
		{
			name: "full settings",
			tokenSettings: &common.TokenSettings{
				DefaultSignatureAlgorithm:           "RS256",
				RevokeRefreshToken:                  true,
				RefreshTokenMaxReuse:                230,
				AccessTokenLifespan:                 231,
				AccessTokenLifespanForImplicitFlow:  232,
				AccessCodeLifespan:                  233,
				ActionTokenGeneratedByUserLifespan:  234,
				ActionTokenGeneratedByAdminLifespan: 235,
			},
			want: &TokenSettings{
				DefaultSignatureAlgorithm:           "RS256",
				RevokeRefreshToken:                  true,
				RefreshTokenMaxReuse:                230,
				AccessTokenLifespan:                 231,
				AccessTokenLifespanForImplicitFlow:  232,
				AccessCodeLifespan:                  233,
				ActionTokenGeneratedByUserLifespan:  234,
				ActionTokenGeneratedByAdminLifespan: 235,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, ToRealmTokenSettings(tt.tokenSettings))
		})
	}
}

// setupOrganizationToggleServer creates a test server that handles GET and PUT requests
// for toggling organization settings. It returns the current organizationsEnabled state
// as the opposite of the target enabled state (simulating a state change).
func setupOrganizationToggleServer(realmName string, currentOrgEnabled bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			expectedPath := strings.NewReplacer(
				"{realm}", realmName,
			).Replace(realmEntity)
			if r.URL.Path == expectedPath {
				response := map[string]interface{}{
					"realm":                realmName,
					"organizationsEnabled": currentOrgEnabled,
				}

				setJSONContentType(w)
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(response)

				return
			}
		case http.MethodPut:
			expectedPath := strings.NewReplacer(
				"{realm}", realmName,
			).Replace(realmEntity)
			if r.URL.Path == expectedPath {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		http.NotFound(w, r)
	}))
}

func TestGoCloakAdapter_SetRealmOrganizationsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		realmName     string
		enabled       bool
		setupServer   func(t *testing.T) *httptest.Server
		expectedError string
	}{
		{
			name:      "enable organizations successfully",
			realmName: "test-realm",
			enabled:   true,
			setupServer: func(t *testing.T) *httptest.Server {
				return setupOrganizationToggleServer("test-realm", false)
			},
			expectedError: "",
		},
		{
			name:      "disable organizations successfully",
			realmName: "test-realm",
			enabled:   false,
			setupServer: func(t *testing.T) *httptest.Server {
				return setupOrganizationToggleServer("test-realm", true)
			},
			expectedError: "",
		},
		{
			name:      "no change needed - already enabled",
			realmName: "test-realm",
			enabled:   true,
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPath := strings.NewReplacer(
						"{realm}", "test-realm",
					).Replace(realmEntity)
					if r.Method == http.MethodGet && r.URL.Path == expectedPath {
						response := map[string]interface{}{
							"realm":                "test-realm",
							"organizationsEnabled": true,
						}
						setJSONContentType(w)
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(response)
						return
					}
					http.NotFound(w, r)
				}))
			},
			expectedError: "",
		},
		{
			name:      "no change needed - already disabled",
			realmName: "test-realm",
			enabled:   false,
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPath := strings.NewReplacer(
						"{realm}", "test-realm",
					).Replace(realmEntity)
					if r.Method == http.MethodGet && r.URL.Path == expectedPath {
						response := map[string]interface{}{
							"realm":                "test-realm",
							"organizationsEnabled": false,
						}
						setJSONContentType(w)
						w.WriteHeader(http.StatusOK)
						_ = json.NewEncoder(w).Encode(response)
						return
					}
					http.NotFound(w, r)
				}))
			},
			expectedError: "",
		},
		{
			name:      "get realm fails",
			realmName: "test-realm",
			enabled:   true,
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPath := strings.NewReplacer(
						"{realm}", "test-realm",
					).Replace(realmEntity)
					if r.Method == http.MethodGet && r.URL.Path == expectedPath {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte("realm not found"))
						return
					}
					http.NotFound(w, r)
				}))
			},
			expectedError: "unable to get realm",
		},
		{
			name:      "update realm fails",
			realmName: "test-realm",
			enabled:   true,
			setupServer: func(t *testing.T) *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case http.MethodGet:
						expectedPath := strings.NewReplacer(
							"{realm}", "test-realm",
						).Replace(realmEntity)
						if r.URL.Path == expectedPath {
							response := map[string]interface{}{
								"realm":                "test-realm",
								"organizationsEnabled": false,
							}
							setJSONContentType(w)
							w.WriteHeader(http.StatusOK)
							_ = json.NewEncoder(w).Encode(response)
							return
						}
					case http.MethodPut:
						expectedPath := strings.NewReplacer(
							"{realm}", "test-realm",
						).Replace(realmEntity)
						if r.URL.Path == expectedPath {
							w.WriteHeader(http.StatusInternalServerError)
							_, _ = w.Write([]byte("internal server error"))
							return
						}
					}
					http.NotFound(w, r)
				}))
			},
			expectedError: "unable to set realm organizations enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup test server
			server := tt.setupServer(t)
			defer server.Close()

			// Initialize adapter with test server URL (without httpmock)
			mockClient := mocks.NewMockGoCloak(t)
			restyClient := resty.New()
			restyClient.SetBaseURL(server.URL)
			mockClient.On("RestyClient").Return(restyClient).Maybe()

			adapter := &GoCloakAdapter{
				client:   mockClient,
				basePath: "",
				token:    &gocloak.JWT{AccessToken: "token"},
				log:      logr.Discard(),
			}

			// Execute the method
			err := adapter.SetRealmOrganizationsEnabled(context.Background(), tt.realmName, tt.enabled)

			// Assert results
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
