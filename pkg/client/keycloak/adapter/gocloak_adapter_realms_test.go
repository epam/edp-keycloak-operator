package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/common"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

func initRealmsAdapter(t *testing.T, server *httptest.Server) (*GoCloakAdapter, *mocks.MockGoCloak) {
	t.Helper()

	var mockClient *mocks.MockGoCloak

	if server != nil {
		mockClient = newMockClientWithResty(t, server.URL)
	} else {
		mockClient = mocks.NewMockGoCloak(t)
		restyClient := resty.New()
		mockClient.On("RestyClient").Return(restyClient).Maybe()
	}

	logger := ctrl.Log.WithName("test")

	return &GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
		log:      logger,
	}, mockClient
}

func TestGoCloakAdapter_UpdateRealmSettings(t *testing.T) {
	adapter, mockClient := initRealmsAdapter(t, nil)

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

func TestGoCloakAdapter_UpdateRealmSettings_WithLogin(t *testing.T) {
	adapter, mockClient := initRealmsAdapter(t, nil)

	settings := RealmSettings{
		FrontendURL:     "https://google.com",
		DisplayHTMLName: "Test Realm",
		DisplayName:     "Test",
		Login: &RealmLogin{
			UserRegistration: true,
			ForgotPassword:   true,
			RememberMe:       true,
			EmailAsUsername:  false,
			LoginWithEmail:   true,
			DuplicateEmails:  false,
			VerifyEmail:      true,
			EditUsername:     false,
		},
	}
	realmName := "realm-with-login"

	realm := gocloak.RealmRepresentation{}
	mockClient.EXPECT().GetRealm(mock.Anything, adapter.token.AccessToken, realmName).Return(&realm, nil)
	mockClient.EXPECT().UpdateRealm(mock.Anything, "token", mock.MatchedBy(func(realm gocloak.RealmRepresentation) bool {
		return assert.Equal(t, gocloak.BoolP(true), realm.RegistrationAllowed) &&
			assert.Equal(t, gocloak.BoolP(true), realm.ResetPasswordAllowed) &&
			assert.Equal(t, gocloak.BoolP(true), realm.RememberMe) &&
			assert.Equal(t, gocloak.BoolP(false), realm.RegistrationEmailAsUsername) &&
			assert.Equal(t, gocloak.BoolP(true), realm.LoginWithEmailAllowed) &&
			assert.Equal(t, gocloak.BoolP(false), realm.DuplicateEmailsAllowed) &&
			assert.Equal(t, gocloak.BoolP(true), realm.VerifyEmail) &&
			assert.Equal(t, gocloak.BoolP(false), realm.EditUsernameAllowed)
	})).Return(nil)

	err := adapter.UpdateRealmSettings(realmName, &settings)
	require.NoError(t, err)
}

func TestGoCloakAdapter_UpdateRealmSettings_WithSSOSessionSettings(t *testing.T) {
	adapter, mockClient := initRealmsAdapter(t, nil)

	settings := RealmSettings{
		SSOSessionSettings: &SSOSessionSettings{
			IdleTimeout:           1800,
			MaxLifespan:           36000,
			IdleTimeoutRememberMe: 3600,
			MaxRememberMe:         86400,
		},
	}
	realmName := "realm-with-sso-session"

	realm := gocloak.RealmRepresentation{}
	mockClient.EXPECT().GetRealm(mock.Anything, adapter.token.AccessToken, realmName).Return(&realm, nil)
	mockClient.EXPECT().UpdateRealm(mock.Anything, "token", mock.MatchedBy(func(realm gocloak.RealmRepresentation) bool {
		return assert.Equal(t, gocloak.IntP(1800), realm.SsoSessionIdleTimeout) &&
			assert.Equal(t, gocloak.IntP(36000), realm.SsoSessionMaxLifespan) &&
			assert.Equal(t, gocloak.IntP(3600), realm.SsoSessionIdleTimeoutRememberMe) &&
			assert.Equal(t, gocloak.IntP(86400), realm.SsoSessionMaxLifespanRememberMe)
	})).Return(nil)

	err := adapter.UpdateRealmSettings(realmName, &settings)
	require.NoError(t, err)
}

func TestGoCloakAdapter_UpdateRealmSettings_WithSSOOfflineSessionSettings(t *testing.T) {
	adapter, mockClient := initRealmsAdapter(t, nil)

	settings := RealmSettings{
		SSOOfflineSessionSettings: &SSOOfflineSessionSettings{
			IdleTimeout:        2592000,
			MaxLifespanEnabled: true,
			MaxLifespan:        5184000,
		},
	}
	realmName := "realm-with-sso-offline-session"

	realm := gocloak.RealmRepresentation{}
	mockClient.EXPECT().GetRealm(mock.Anything, adapter.token.AccessToken, realmName).Return(&realm, nil)
	mockClient.EXPECT().UpdateRealm(mock.Anything, "token", mock.MatchedBy(func(realm gocloak.RealmRepresentation) bool {
		return assert.Equal(t, gocloak.IntP(2592000), realm.OfflineSessionIdleTimeout) &&
			assert.Equal(t, gocloak.BoolP(true), realm.OfflineSessionMaxLifespanEnabled) &&
			assert.Equal(t, gocloak.IntP(5184000), realm.OfflineSessionMaxLifespan)
	})).Return(nil)

	err := adapter.UpdateRealmSettings(realmName, &settings)
	require.NoError(t, err)
}

func TestGoCloakAdapter_UpdateRealmSettings_WithSSOLoginSettings(t *testing.T) {
	adapter, mockClient := initRealmsAdapter(t, nil)

	settings := RealmSettings{
		SSOLoginSettings: &SSOLoginSettings{
			AccessCodeLifespanLogin:      1800,
			AccessCodeLifespanUserAction: 300,
		},
	}
	realmName := "realm-with-sso-login"

	realm := gocloak.RealmRepresentation{}
	mockClient.EXPECT().GetRealm(mock.Anything, adapter.token.AccessToken, realmName).Return(&realm, nil)
	mockClient.EXPECT().UpdateRealm(mock.Anything, "token", mock.MatchedBy(func(realm gocloak.RealmRepresentation) bool {
		return assert.Equal(t, gocloak.IntP(1800), realm.AccessCodeLifespanLogin) &&
			assert.Equal(t, gocloak.IntP(300), realm.AccessCodeLifespanUserAction)
	})).Return(nil)

	err := adapter.UpdateRealmSettings(realmName, &settings)
	require.NoError(t, err)
}

func TestGoCloakAdapter_UpdateRealmSettings_WithAllSSOSettings(t *testing.T) {
	adapter, mockClient := initRealmsAdapter(t, nil)

	settings := RealmSettings{
		SSOSessionSettings: &SSOSessionSettings{
			IdleTimeout:           1800,
			MaxLifespan:           36000,
			IdleTimeoutRememberMe: 3600,
			MaxRememberMe:         86400,
		},
		SSOOfflineSessionSettings: &SSOOfflineSessionSettings{
			IdleTimeout:        2592000,
			MaxLifespanEnabled: true,
			MaxLifespan:        5184000,
		},
		SSOLoginSettings: &SSOLoginSettings{
			AccessCodeLifespanLogin:      1800,
			AccessCodeLifespanUserAction: 300,
		},
	}
	realmName := "realm-with-all-sso-settings"

	realm := gocloak.RealmRepresentation{}
	mockClient.EXPECT().GetRealm(mock.Anything, adapter.token.AccessToken, realmName).Return(&realm, nil)
	mockClient.EXPECT().UpdateRealm(mock.Anything, "token", mock.MatchedBy(func(realm gocloak.RealmRepresentation) bool {
		return assert.Equal(t, gocloak.IntP(1800), realm.SsoSessionIdleTimeout) &&
			assert.Equal(t, gocloak.IntP(36000), realm.SsoSessionMaxLifespan) &&
			assert.Equal(t, gocloak.IntP(3600), realm.SsoSessionIdleTimeoutRememberMe) &&
			assert.Equal(t, gocloak.IntP(86400), realm.SsoSessionMaxLifespanRememberMe) &&
			assert.Equal(t, gocloak.IntP(2592000), realm.OfflineSessionIdleTimeout) &&
			assert.Equal(t, gocloak.BoolP(true), realm.OfflineSessionMaxLifespanEnabled) &&
			assert.Equal(t, gocloak.IntP(5184000), realm.OfflineSessionMaxLifespan) &&
			assert.Equal(t, gocloak.IntP(1800), realm.AccessCodeLifespanLogin) &&
			assert.Equal(t, gocloak.IntP(300), realm.AccessCodeLifespanUserAction)
	})).Return(nil)

	err := adapter.UpdateRealmSettings(realmName, &settings)
	require.NoError(t, err)
}

func TestGoCloakAdapter_SyncRealmIdentityProviderMappers(t *testing.T) {
	currentMapperID := "mp1id"
	realmName := "sso-realm-1"
	idpAlias := "alias-1"

	mappers := []interface{}{
		map[string]interface{}{
			keycloakApiParamId: currentMapperID,
			"name":             "mp1name",
		},
	}

	realm := gocloak.RealmRepresentation{
		Realm:                   gocloak.StringP(realmName),
		IdentityProviderMappers: &mappers,
	}

	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == strings.Replace(
			strings.Replace(idpMapperCreateList, "{realm}", realmName, 1), "{alias}", idpAlias, 1):
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("ok"))
		case r.Method == http.MethodPut && r.URL.Path == strings.Replace(
			strings.Replace(
				strings.Replace(idpMapperEntity, "{realm}", realmName, 1), "{alias}", idpAlias, 1), "{id}", currentMapperID, 1):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	adapter, mockClient := initRealmsAdapter(t, server)
	mockClient.On("GetRealm", mock.Anything, adapter.token.AccessToken, realmName).Return(&realm, nil)

	if err := adapter.SyncRealmIdentityProviderMappers(realmName,
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
	adapter, mockClient := initRealmsAdapter(t, nil)
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
	adapter, mockClient := initRealmsAdapter(t, nil)

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

			// Initialize adapter with test server URL
			mockClient := newMockClientWithResty(t, server.URL)

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

func TestSessionSettingsConversions(t *testing.T) {
	t.Run("ToRealmSSOSessionSettings", func(t *testing.T) {
		tests := []struct {
			name     string
			settings *common.RealmSSOSessionSettings
			want     *SSOSessionSettings
		}{
			{
				name:     "nil settings returns nil",
				settings: nil,
				want:     nil,
			},
			{
				name: "converts all fields correctly",
				settings: &common.RealmSSOSessionSettings{
					IdleTimeout:           1800,
					MaxLifespan:           36000,
					IdleTimeoutRememberMe: 3600,
					MaxLifespanRememberMe: 86400,
				},
				want: &SSOSessionSettings{
					IdleTimeout:           1800,
					MaxLifespan:           36000,
					IdleTimeoutRememberMe: 3600,
					MaxRememberMe:         86400,
				},
			},
			{
				name: "converts zero values correctly",
				settings: &common.RealmSSOSessionSettings{
					IdleTimeout:           0,
					MaxLifespan:           0,
					IdleTimeoutRememberMe: 0,
					MaxLifespanRememberMe: 0,
				},
				want: &SSOSessionSettings{
					IdleTimeout:           0,
					MaxLifespan:           0,
					IdleTimeoutRememberMe: 0,
					MaxRememberMe:         0,
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := ToRealmSSOSessionSettings(tt.settings)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("ToRealmSSOOfflineSessionSettings", func(t *testing.T) {
		tests := []struct {
			name     string
			settings *common.RealmSSOOfflineSessionSettings
			want     *SSOOfflineSessionSettings
		}{
			{
				name:     "nil settings returns nil",
				settings: nil,
				want:     nil,
			},
			{
				name: "converts all fields correctly",
				settings: &common.RealmSSOOfflineSessionSettings{
					IdleTimeout:        2592000,
					MaxLifespanEnabled: true,
					MaxLifespan:        5184000,
				},
				want: &SSOOfflineSessionSettings{
					IdleTimeout:        2592000,
					MaxLifespanEnabled: true,
					MaxLifespan:        5184000,
				},
			},
			{
				name: "converts with disabled max lifespan",
				settings: &common.RealmSSOOfflineSessionSettings{
					IdleTimeout:        2592000,
					MaxLifespanEnabled: false,
					MaxLifespan:        0,
				},
				want: &SSOOfflineSessionSettings{
					IdleTimeout:        2592000,
					MaxLifespanEnabled: false,
					MaxLifespan:        0,
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := ToRealmSSOOfflineSessionSettings(tt.settings)
				assert.Equal(t, tt.want, got)
			})
		}
	})

	t.Run("ToRealmSSOLoginSettings", func(t *testing.T) {
		tests := []struct {
			name     string
			settings *common.RealmSSOLoginSettings
			want     *SSOLoginSettings
		}{
			{
				name:     "nil settings returns nil",
				settings: nil,
				want:     nil,
			},
			{
				name: "converts all fields correctly",
				settings: &common.RealmSSOLoginSettings{
					AccessCodeLifespanLogin:      1800,
					AccessCodeLifespanUserAction: 300,
				},
				want: &SSOLoginSettings{
					AccessCodeLifespanLogin:      1800,
					AccessCodeLifespanUserAction: 300,
				},
			},
			{
				name: "converts zero values correctly",
				settings: &common.RealmSSOLoginSettings{
					AccessCodeLifespanLogin:      0,
					AccessCodeLifespanUserAction: 0,
				},
				want: &SSOLoginSettings{
					AccessCodeLifespanLogin:      0,
					AccessCodeLifespanUserAction: 0,
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := ToRealmSSOLoginSettings(tt.settings)
				assert.Equal(t, tt.want, got)
			})
		}
	})
}
