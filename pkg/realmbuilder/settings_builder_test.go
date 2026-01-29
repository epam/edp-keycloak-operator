package realmbuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestSettingsBuilder_SetRealmEventConfigFromV1(t *testing.T) {
	testSetRealmEventConfig(
		t,
		func(builder *SettingsBuilder, client keycloak.Client, realmName string, hasConfig bool) error {
			var config *keycloakApi.RealmEventConfig
			if hasConfig {
				config = &keycloakApi.RealmEventConfig{
					AdminEventsDetailsEnabled: true,
					AdminEventsEnabled:        true,
					EventsEnabled:             true,
					EventsExpiration:          100,
					EnabledEventTypes:         []string{"LOGIN", "LOGOUT"},
					EventsListeners:           []string{"jboss-logging"},
				}
			}

			return builder.SetRealmEventConfigFromV1(client, realmName, config)
		},
	)
}

func TestSettingsBuilder_SetRealmEventConfigFromV1Alpha1(t *testing.T) {
	testSetRealmEventConfig(
		t,
		func(builder *SettingsBuilder, client keycloak.Client, realmName string, hasConfig bool) error {
			var config *v1alpha1.RealmEventConfig
			if hasConfig {
				config = &v1alpha1.RealmEventConfig{
					AdminEventsDetailsEnabled: true,
					AdminEventsEnabled:        true,
					EventsEnabled:             true,
					EventsExpiration:          100,
					EnabledEventTypes:         []string{"LOGIN", "LOGOUT"},
					EventsListeners:           []string{"jboss-logging"},
				}
			}

			return builder.SetRealmEventConfigFromV1Alpha1(client, realmName, config)
		},
	)
}

func testSetRealmEventConfig(t *testing.T, setConfigFn func(*SettingsBuilder, keycloak.Client, string, bool) error) {
	t.Helper()

	builder := NewSettingsBuilder()

	tests := []struct {
		name      string
		hasConfig bool
		setupMock func(*mocks.MockClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "nil config should not call client",
			hasConfig: false,
			setupMock: func(m *mocks.MockClient) {},
			wantErr:   require.NoError,
		},
		{
			name:      "valid config should call client",
			hasConfig: true,
			setupMock: func(m *mocks.MockClient) {
				m.EXPECT().SetRealmEventConfig("test-realm", mock.Anything).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name:      "error when SetRealmEventConfig fails",
			hasConfig: true,
			setupMock: func(m *mocks.MockClient) {
				m.EXPECT().SetRealmEventConfig("test-realm", mock.Anything).Return(assert.AnError)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to set realm event config")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockClient(t)
			tt.setupMock(mockClient)

			err := setConfigFn(builder, mockClient, "test-realm", tt.hasConfig)

			tt.wantErr(t, err)
		})
	}
}

func TestSettingsBuilder_BuildFromV1(t *testing.T) {
	builder := NewSettingsBuilder()
	loginTheme := "custom-login"

	tests := []struct {
		name  string
		realm *keycloakApi.KeycloakRealm
		want  adapter.RealmSettings
	}{
		{
			name: "minimal configuration",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName:       "test-realm",
					DisplayName:     "Test Realm",
					DisplayHTMLName: "<b>Test</b>",
					FrontendURL:     "https://example.com",
				},
			},
			want: adapter.RealmSettings{
				DisplayName:     "Test Realm",
				DisplayHTMLName: "<b>Test</b>",
				FrontendURL:     "https://example.com",
			},
		},
		{
			name: "with themes",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					Themes: &keycloakApi.RealmThemes{
						LoginTheme:                  &loginTheme,
						InternationalizationEnabled: ptr.To(true),
					},
				},
			},
			want: adapter.RealmSettings{
				Themes: &adapter.RealmThemes{
					LoginTheme:                  &loginTheme,
					InternationalizationEnabled: ptr.To(true),
				},
			},
		},
		{
			name: "with password policies",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					PasswordPolicies: []keycloakApi.PasswordPolicy{
						{Type: "length", Value: "8"},
						{Type: "upperCase", Value: "1"},
					},
				},
			},
			want: adapter.RealmSettings{
				PasswordPolicies: []adapter.PasswordPolicy{
					{Type: "length", Value: "8"},
					{Type: "upperCase", Value: "1"},
				},
			},
		},
		{
			name: "with login settings",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					Login: &keycloakApi.RealmLogin{
						UserRegistration: true,
						ForgotPassword:   true,
						RememberMe:       true,
						EmailAsUsername:  false,
						LoginWithEmail:   true,
						DuplicateEmails:  false,
						VerifyEmail:      true,
						EditUsername:     false,
					},
				},
			},
			want: adapter.RealmSettings{
				Login: &adapter.RealmLogin{
					UserRegistration: true,
					ForgotPassword:   true,
					RememberMe:       true,
					EmailAsUsername:  false,
					LoginWithEmail:   true,
					DuplicateEmails:  false,
					VerifyEmail:      true,
					EditUsername:     false,
				},
			},
		},
		{
			name: "with admin events expiration",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					RealmEventConfig: &keycloakApi.RealmEventConfig{
						AdminEventsEnabled:    true,
						AdminEventsExpiration: 3600,
					},
				},
			},
			want: adapter.RealmSettings{
				AdminEventsExpiration: ptr.To(3600),
			},
		},
		{
			name: "with browser security headers",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					BrowserSecurityHeaders: &map[string]string{
						"X-Frame-Options":        "SAMEORIGIN",
						"X-Content-Type-Options": "nosniff",
					},
				},
			},
			want: adapter.RealmSettings{
				BrowserSecurityHeaders: &map[string]string{
					"X-Frame-Options":        "SAMEORIGIN",
					"X-Content-Type-Options": "nosniff",
				},
			},
		},
		{
			name: "with all session types",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					Login: &keycloakApi.RealmLogin{
						RememberMe: true,
					},
					Sessions: &common.RealmSessions{
						SSOSessionSettings: &common.RealmSSOSessionSettings{
							IdleTimeout:           1800,
							MaxLifespan:           36000,
							IdleTimeoutRememberMe: 3600,
							MaxLifespanRememberMe: 72000,
						},
						SSOOfflineSessionSettings: &common.RealmSSOOfflineSessionSettings{
							IdleTimeout:        2592000,
							MaxLifespanEnabled: true,
							MaxLifespan:        5184000,
						},
						SSOLoginSettings: &common.RealmSSOLoginSettings{
							AccessCodeLifespanLogin:      1800,
							AccessCodeLifespanUserAction: 300,
						},
					},
				},
			},
			want: adapter.RealmSettings{
				Login: &adapter.RealmLogin{
					RememberMe: true,
				},
				SSOSessionSettings: &adapter.SSOSessionSettings{
					IdleTimeout:           1800,
					MaxLifespan:           36000,
					IdleTimeoutRememberMe: 3600,
					MaxRememberMe:         72000,
				},
				SSOOfflineSessionSettings: &adapter.SSOOfflineSessionSettings{
					IdleTimeout:        2592000,
					MaxLifespanEnabled: true,
					MaxLifespan:        5184000,
				},
				SSOLoginSettings: &adapter.SSOLoginSettings{
					AccessCodeLifespanLogin:      1800,
					AccessCodeLifespanUserAction: 300,
				},
			},
		},
	}

	//nolint:dupl // Duplicate assertion pattern aids readability in testing BuildFromV1 and BuildFromV1Alpha1 separately
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildFromV1(tt.realm)

			assert.Equal(t, tt.want.DisplayName, got.DisplayName)
			assert.Equal(t, tt.want.DisplayHTMLName, got.DisplayHTMLName)
			assert.Equal(t, tt.want.FrontendURL, got.FrontendURL)
			assert.Equal(t, tt.want.Themes, got.Themes)
			assert.Equal(t, tt.want.PasswordPolicies, got.PasswordPolicies)
			assert.Equal(t, tt.want.Login, got.Login)
			assert.Equal(t, tt.want.AdminEventsExpiration, got.AdminEventsExpiration)
			assert.Equal(t, tt.want.BrowserSecurityHeaders, got.BrowserSecurityHeaders)
			assert.Equal(t, tt.want.SSOSessionSettings, got.SSOSessionSettings)
			assert.Equal(t, tt.want.SSOOfflineSessionSettings, got.SSOOfflineSessionSettings)
			assert.Equal(t, tt.want.SSOLoginSettings, got.SSOLoginSettings)
		})
	}
}

func TestSettingsBuilder_BuildFromV1Alpha1(t *testing.T) {
	builder := NewSettingsBuilder()
	loginTheme := "custom-login"

	tests := []struct {
		name  string
		realm *v1alpha1.ClusterKeycloakRealm
		want  adapter.RealmSettings
	}{
		{
			name: "minimal configuration",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName:       "test-realm",
					DisplayName:     "Test Realm",
					DisplayHTMLName: "<b>Test</b>",
					FrontendURL:     "https://example.com",
				},
			},
			want: adapter.RealmSettings{
				DisplayName:     "Test Realm",
				DisplayHTMLName: "<b>Test</b>",
				FrontendURL:     "https://example.com",
			},
		},
		{
			name: "with themes",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Themes: &v1alpha1.ClusterRealmThemes{
						LoginTheme: &loginTheme,
					},
				},
			},
			want: adapter.RealmSettings{
				Themes: &adapter.RealmThemes{
					LoginTheme: &loginTheme,
				},
			},
		},
		{
			name: "with localization",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Localization: &v1alpha1.RealmLocalization{
						InternationalizationEnabled: ptr.To(true),
					},
				},
			},
			want: adapter.RealmSettings{
				Themes: &adapter.RealmThemes{
					InternationalizationEnabled: ptr.To(true),
				},
			},
		},
		{
			name: "with themes and localization",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Themes: &v1alpha1.ClusterRealmThemes{
						LoginTheme: &loginTheme,
					},
					Localization: &v1alpha1.RealmLocalization{
						InternationalizationEnabled: ptr.To(true),
					},
				},
			},
			want: adapter.RealmSettings{
				Themes: &adapter.RealmThemes{
					LoginTheme:                  &loginTheme,
					InternationalizationEnabled: ptr.To(true),
				},
			},
		},
		{
			name: "with password policies",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					PasswordPolicies: []v1alpha1.PasswordPolicy{
						{Type: "length", Value: "8"},
						{Type: "upperCase", Value: "1"},
					},
				},
			},
			want: adapter.RealmSettings{
				PasswordPolicies: []adapter.PasswordPolicy{
					{Type: "length", Value: "8"},
					{Type: "upperCase", Value: "1"},
				},
			},
		},
		{
			name: "with admin events expiration",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					RealmEventConfig: &v1alpha1.RealmEventConfig{
						AdminEventsEnabled:    true,
						AdminEventsExpiration: 7200,
					},
				},
			},
			want: adapter.RealmSettings{
				AdminEventsExpiration: ptr.To(7200),
			},
		},
		{
			name: "with browser security headers",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					BrowserSecurityHeaders: &map[string]string{
						"X-Frame-Options":        "DENY",
						"X-Content-Type-Options": "nosniff",
					},
				},
			},
			want: adapter.RealmSettings{
				BrowserSecurityHeaders: &map[string]string{
					"X-Frame-Options":        "DENY",
					"X-Content-Type-Options": "nosniff",
				},
			},
		},
		{
			name: "with login settings",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Login: &keycloakApi.RealmLogin{
						UserRegistration: true,
						ForgotPassword:   false,
						RememberMe:       true,
						EmailAsUsername:  true,
						LoginWithEmail:   false,
						DuplicateEmails:  true,
						VerifyEmail:      false,
						EditUsername:     true,
					},
				},
			},
			want: adapter.RealmSettings{
				Login: &adapter.RealmLogin{
					UserRegistration: true,
					ForgotPassword:   false,
					RememberMe:       true,
					EmailAsUsername:  true,
					LoginWithEmail:   false,
					DuplicateEmails:  true,
					VerifyEmail:      false,
					EditUsername:     true,
				},
			},
		},
		{
			name: "with all session types",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Sessions: &common.RealmSessions{
						SSOSessionSettings: &common.RealmSSOSessionSettings{
							IdleTimeout:           1800,
							MaxLifespan:           36000,
							IdleTimeoutRememberMe: 3600,
							MaxLifespanRememberMe: 72000,
						},
						SSOOfflineSessionSettings: &common.RealmSSOOfflineSessionSettings{
							IdleTimeout:        2592000,
							MaxLifespanEnabled: true,
							MaxLifespan:        5184000,
						},
						SSOLoginSettings: &common.RealmSSOLoginSettings{
							AccessCodeLifespanLogin:      1800,
							AccessCodeLifespanUserAction: 300,
						},
					},
				},
			},
			want: adapter.RealmSettings{
				SSOSessionSettings: &adapter.SSOSessionSettings{
					IdleTimeout:           1800,
					MaxLifespan:           36000,
					IdleTimeoutRememberMe: 3600,
					MaxRememberMe:         72000,
				},
				SSOOfflineSessionSettings: &adapter.SSOOfflineSessionSettings{
					IdleTimeout:        2592000,
					MaxLifespanEnabled: true,
					MaxLifespan:        5184000,
				},
				SSOLoginSettings: &adapter.SSOLoginSettings{
					AccessCodeLifespanLogin:      1800,
					AccessCodeLifespanUserAction: 300,
				},
			},
		},
	}

	//nolint:dupl // Duplicate assertion pattern aids readability in testing BuildFromV1 and BuildFromV1Alpha1 separately
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := builder.BuildFromV1Alpha1(tt.realm)

			assert.Equal(t, tt.want.DisplayName, got.DisplayName)
			assert.Equal(t, tt.want.DisplayHTMLName, got.DisplayHTMLName)
			assert.Equal(t, tt.want.FrontendURL, got.FrontendURL)
			assert.Equal(t, tt.want.Themes, got.Themes)
			assert.Equal(t, tt.want.PasswordPolicies, got.PasswordPolicies)
			assert.Equal(t, tt.want.AdminEventsExpiration, got.AdminEventsExpiration)
			assert.Equal(t, tt.want.BrowserSecurityHeaders, got.BrowserSecurityHeaders)
			assert.Equal(t, tt.want.Login, got.Login)
			assert.Equal(t, tt.want.SSOSessionSettings, got.SSOSessionSettings)
			assert.Equal(t, tt.want.SSOOfflineSessionSettings, got.SSOOfflineSessionSettings)
			assert.Equal(t, tt.want.SSOLoginSettings, got.SSOLoginSettings)
		})
	}
}

func TestMakePasswordPoliciesFromV1(t *testing.T) {
	policies := makePasswordPoliciesFromV1([]keycloakApi.PasswordPolicy{
		{Type: "length", Value: "8"},
		{Type: "upperCase", Value: "1"},
	})

	require.Len(t, policies, 2)
	assert.Equal(t, adapter.PasswordPolicy{Type: "length", Value: "8"}, policies[0])
	assert.Equal(t, adapter.PasswordPolicy{Type: "upperCase", Value: "1"}, policies[1])
}

func TestMakePasswordPoliciesFromV1Alpha1(t *testing.T) {
	policies := makePasswordPoliciesFromV1Alpha1([]v1alpha1.PasswordPolicy{
		{Type: "length", Value: "8"},
		{Type: "upperCase", Value: "1"},
	})

	require.Len(t, policies, 2)
	assert.Equal(t, adapter.PasswordPolicy{Type: "length", Value: "8"}, policies[0])
	assert.Equal(t, adapter.PasswordPolicy{Type: "upperCase", Value: "1"}, policies[1])
}
