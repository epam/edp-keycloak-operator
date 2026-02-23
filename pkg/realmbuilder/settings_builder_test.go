package realmbuilder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestBuildRealmRepresentationFromV1(t *testing.T) {
	loginTheme := "custom-login"
	accountTheme := "custom-account"

	tests := []struct {
		name  string
		realm *keycloakApi.KeycloakRealm
		check func(t *testing.T, got keycloakv2.RealmRepresentation)
	}{
		{
			name: "minimal configuration",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					DisplayName:     "Test Realm",
					DisplayHTMLName: "<b>Test</b>",
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, ptr.To("Test Realm"), got.DisplayName)
				assert.Equal(t, ptr.To("<b>Test</b>"), got.DisplayNameHtml)
				assert.Equal(t, ptr.To(false), got.OrganizationsEnabled)
				assert.Nil(t, got.LoginTheme)
				assert.Nil(t, got.Attributes)
			},
		},
		{
			name: "with themes",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					Themes: &keycloakApi.RealmThemes{
						LoginTheme:                  &loginTheme,
						AccountTheme:                &accountTheme,
						InternationalizationEnabled: ptr.To(true),
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, &loginTheme, got.LoginTheme)
				assert.Equal(t, &accountTheme, got.AccountTheme)
				assert.Equal(t, ptr.To(true), got.InternationalizationEnabled)
			},
		},
		{
			name: "with frontend URL stored in attributes",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					FrontendURL: "https://example.com",
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				require.NotNil(t, got.Attributes)
				assert.Equal(t, "https://example.com", (*got.Attributes)["frontendUrl"])
			},
		},
		{
			name: "with browser security headers",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					BrowserSecurityHeaders: &map[string]string{
						"X-Frame-Options": "SAMEORIGIN",
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				require.NotNil(t, got.BrowserSecurityHeaders)
				assert.Equal(t, "SAMEORIGIN", (*got.BrowserSecurityHeaders)["X-Frame-Options"])
			},
		},
		{
			name: "with password policies formatted as string",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					PasswordPolicies: []common.PasswordPolicy{
						{Type: "length", Value: "8"},
						{Type: "upperCase", Value: "1"},
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				require.NotNil(t, got.PasswordPolicy)
				assert.Equal(t, "length(8) and upperCase(1)", *got.PasswordPolicy)
			},
		},
		{
			name: "with token settings",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					TokenSettings: &common.TokenSettings{
						DefaultSignatureAlgorithm:           "RS256",
						RevokeRefreshToken:                  true,
						RefreshTokenMaxReuse:                3,
						AccessTokenLifespan:                 300,
						AccessTokenLifespanForImplicitFlow:  900,
						AccessCodeLifespan:                  60,
						ActionTokenGeneratedByUserLifespan:  300,
						ActionTokenGeneratedByAdminLifespan: 43200,
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, ptr.To("RS256"), got.DefaultSignatureAlgorithm)
				assert.Equal(t, ptr.To(true), got.RevokeRefreshToken)
				assert.Equal(t, ptr.To(int32(3)), got.RefreshTokenMaxReuse)
				assert.Equal(t, ptr.To(int32(300)), got.AccessTokenLifespan)
				assert.Equal(t, ptr.To(int32(900)), got.AccessTokenLifespanForImplicitFlow)
				assert.Equal(t, ptr.To(int32(60)), got.AccessCodeLifespan)
				assert.Equal(t, ptr.To(int32(300)), got.ActionTokenGeneratedByUserLifespan)
				assert.Equal(t, ptr.To(int32(43200)), got.ActionTokenGeneratedByAdminLifespan)
			},
		},
		{
			name: "with admin events expiration stored in attributes",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmEventConfig: &common.RealmEventConfig{
						AdminEventsEnabled:    true,
						AdminEventsExpiration: 3600,
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				require.NotNil(t, got.Attributes)
				assert.Equal(t, "3600", (*got.Attributes)["adminEventsExpiration"])
			},
		},
		{
			name: "admin events expiration not set when admin events disabled",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmEventConfig: &common.RealmEventConfig{
						AdminEventsEnabled:    false,
						AdminEventsExpiration: 3600,
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				if got.Attributes != nil {
					assert.NotContains(t, *got.Attributes, "adminEventsExpiration")
				}
			},
		},
		{
			name: "with login settings",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
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
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, ptr.To(true), got.RegistrationAllowed)
				assert.Equal(t, ptr.To(true), got.ResetPasswordAllowed)
				assert.Equal(t, ptr.To(true), got.RememberMe)
				assert.Equal(t, ptr.To(false), got.RegistrationEmailAsUsername)
				assert.Equal(t, ptr.To(true), got.LoginWithEmailAllowed)
				assert.Equal(t, ptr.To(false), got.DuplicateEmailsAllowed)
				assert.Equal(t, ptr.To(true), got.VerifyEmail)
				assert.Equal(t, ptr.To(false), got.EditUsernameAllowed)
			},
		},
		{
			name: "with session settings",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
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
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, ptr.To(int32(1800)), got.SsoSessionIdleTimeout)
				assert.Equal(t, ptr.To(int32(36000)), got.SsoSessionMaxLifespan)
				assert.Equal(t, ptr.To(int32(3600)), got.SsoSessionIdleTimeoutRememberMe)
				assert.Equal(t, ptr.To(int32(72000)), got.SsoSessionMaxLifespanRememberMe)
				assert.Equal(t, ptr.To(int32(2592000)), got.OfflineSessionIdleTimeout)
				assert.Equal(t, ptr.To(true), got.OfflineSessionMaxLifespanEnabled)
				assert.Equal(t, ptr.To(int32(5184000)), got.OfflineSessionMaxLifespan)
				assert.Equal(t, ptr.To(int32(1800)), got.AccessCodeLifespanLogin)
				assert.Equal(t, ptr.To(int32(300)), got.AccessCodeLifespanUserAction)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRealmRepresentationFromV1(tt.realm)
			tt.check(t, got)
		})
	}
}

func TestBuildRealmRepresentationFromV1Alpha1(t *testing.T) {
	loginTheme := "custom-login"

	tests := []struct {
		name  string
		realm *v1alpha1.ClusterKeycloakRealm
		check func(t *testing.T, got keycloakv2.RealmRepresentation)
	}{
		{
			name: "minimal configuration",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					DisplayName:     "Test Realm",
					DisplayHTMLName: "<b>Test</b>",
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, ptr.To("Test Realm"), got.DisplayName)
				assert.Equal(t, ptr.To("<b>Test</b>"), got.DisplayNameHtml)
				assert.Equal(t, ptr.To(false), got.OrganizationsEnabled)
			},
		},
		{
			name: "with themes",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					Themes: &v1alpha1.ClusterRealmThemes{
						LoginTheme: &loginTheme,
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, &loginTheme, got.LoginTheme)
				assert.Nil(t, got.InternationalizationEnabled)
			},
		},
		{
			name: "with localization separate from themes",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					Localization: &v1alpha1.RealmLocalization{
						InternationalizationEnabled: ptr.To(true),
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, ptr.To(true), got.InternationalizationEnabled)
				assert.Nil(t, got.LoginTheme)
			},
		},
		{
			name: "with frontend URL stored in attributes",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					FrontendURL: "https://example.com",
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				require.NotNil(t, got.Attributes)
				assert.Equal(t, "https://example.com", (*got.Attributes)["frontendUrl"])
			},
		},
		{
			name: "with password policies formatted as string",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					PasswordPolicies: []common.PasswordPolicy{
						{Type: "length", Value: "8"},
						{Type: "digits", Value: "1"},
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				require.NotNil(t, got.PasswordPolicy)
				assert.Equal(t, "length(8) and digits(1)", *got.PasswordPolicy)
			},
		},
		{
			name: "with token settings",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					TokenSettings: &common.TokenSettings{
						DefaultSignatureAlgorithm: "ES256",
						AccessTokenLifespan:       600,
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, ptr.To("ES256"), got.DefaultSignatureAlgorithm)
				assert.Equal(t, ptr.To(int32(600)), got.AccessTokenLifespan)
			},
		},
		{
			name: "with admin events expiration stored in attributes",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmEventConfig: &common.RealmEventConfig{
						AdminEventsEnabled:    true,
						AdminEventsExpiration: 7200,
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				require.NotNil(t, got.Attributes)
				assert.Equal(t, "7200", (*got.Attributes)["adminEventsExpiration"])
			},
		},
		{
			name: "with login settings",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					Login: &keycloakApi.RealmLogin{
						UserRegistration: true,
						RememberMe:       true,
						EditUsername:     false,
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, ptr.To(true), got.RegistrationAllowed)
				assert.Equal(t, ptr.To(true), got.RememberMe)
				assert.Equal(t, ptr.To(false), got.EditUsernameAllowed)
			},
		},
		{
			name: "with session settings",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					Sessions: &common.RealmSessions{
						SSOSessionSettings: &common.RealmSSOSessionSettings{
							IdleTimeout: 900,
							MaxLifespan: 18000,
						},
					},
				},
			},
			check: func(t *testing.T, got keycloakv2.RealmRepresentation) {
				t.Helper()
				assert.Equal(t, ptr.To(int32(900)), got.SsoSessionIdleTimeout)
				assert.Equal(t, ptr.To(int32(18000)), got.SsoSessionMaxLifespan)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildRealmRepresentationFromV1Alpha1(tt.realm)
			tt.check(t, got)
		})
	}
}

func TestMergeRealmRepresentation(t *testing.T) {
	t.Run("nil overlay pointer fields do not overwrite base", func(t *testing.T) {
		base := keycloakv2.RealmRepresentation{
			DisplayName:          ptr.To("Original"),
			OrganizationsEnabled: ptr.To(true),
		}
		overlay := keycloakv2.RealmRepresentation{
			DisplayName: nil,
		}

		MergeRealmRepresentation(&base, &overlay)

		assert.Equal(t, ptr.To("Original"), base.DisplayName)
		assert.Equal(t, ptr.To(true), base.OrganizationsEnabled)
	})

	t.Run("non-nil overlay pointer fields overwrite base", func(t *testing.T) {
		base := keycloakv2.RealmRepresentation{
			DisplayName: ptr.To("Original"),
			LoginTheme:  ptr.To("old-theme"),
		}
		overlay := keycloakv2.RealmRepresentation{
			DisplayName: ptr.To("Updated"),
			LoginTheme:  ptr.To("new-theme"),
		}

		MergeRealmRepresentation(&base, &overlay)

		assert.Equal(t, ptr.To("Updated"), base.DisplayName)
		assert.Equal(t, ptr.To("new-theme"), base.LoginTheme)
	})

	t.Run("BrowserSecurityHeaders keys merged preserving base-only keys", func(t *testing.T) {
		baseHeaders := map[string]string{
			"X-Frame-Options":        "SAMEORIGIN",
			"X-Content-Type-Options": "nosniff",
		}
		overlayHeaders := map[string]string{
			"X-Frame-Options":           "DENY",
			"Strict-Transport-Security": "max-age=31536000",
		}
		base := keycloakv2.RealmRepresentation{
			BrowserSecurityHeaders: &baseHeaders,
		}
		overlay := keycloakv2.RealmRepresentation{
			BrowserSecurityHeaders: &overlayHeaders,
		}

		MergeRealmRepresentation(&base, &overlay)

		require.NotNil(t, base.BrowserSecurityHeaders)
		// base-only key preserved
		assert.Equal(t, "nosniff", (*base.BrowserSecurityHeaders)["X-Content-Type-Options"])
		// shared key overwritten by overlay
		assert.Equal(t, "DENY", (*base.BrowserSecurityHeaders)["X-Frame-Options"])
		// overlay-only key added
		assert.Equal(t, "max-age=31536000", (*base.BrowserSecurityHeaders)["Strict-Transport-Security"])
	})

	t.Run("nil base BrowserSecurityHeaders initialised from overlay", func(t *testing.T) {
		overlayHeaders := map[string]string{"X-Frame-Options": "DENY"}
		base := keycloakv2.RealmRepresentation{}
		overlay := keycloakv2.RealmRepresentation{
			BrowserSecurityHeaders: &overlayHeaders,
		}

		MergeRealmRepresentation(&base, &overlay)

		require.NotNil(t, base.BrowserSecurityHeaders)
		assert.Equal(t, "DENY", (*base.BrowserSecurityHeaders)["X-Frame-Options"])
	})

	t.Run("Attributes keys merged preserving base-only keys", func(t *testing.T) {
		baseAttrs := map[string]string{
			"frontendUrl":           "https://existing.com",
			"adminEventsExpiration": "3600",
		}
		overlayAttrs := map[string]string{
			"frontendUrl": "https://new.com",
		}
		base := keycloakv2.RealmRepresentation{Attributes: &baseAttrs}
		overlay := keycloakv2.RealmRepresentation{Attributes: &overlayAttrs}

		MergeRealmRepresentation(&base, &overlay)

		require.NotNil(t, base.Attributes)
		assert.Equal(t, "https://new.com", (*base.Attributes)["frontendUrl"])
		assert.Equal(t, "3600", (*base.Attributes)["adminEventsExpiration"])
	})

	t.Run("PasswordPolicy string replaced not merged", func(t *testing.T) {
		base := keycloakv2.RealmRepresentation{
			PasswordPolicy: ptr.To("length(6)"),
		}
		overlay := keycloakv2.RealmRepresentation{
			PasswordPolicy: ptr.To("length(8) and upperCase(1)"),
		}

		MergeRealmRepresentation(&base, &overlay)

		assert.Equal(t, ptr.To("length(8) and upperCase(1)"), base.PasswordPolicy)
	})

	t.Run("token settings merged correctly", func(t *testing.T) {
		base := keycloakv2.RealmRepresentation{
			AccessTokenLifespan: ptr.To(int32(300)),
			RevokeRefreshToken:  ptr.To(false),
		}
		overlay := keycloakv2.RealmRepresentation{
			AccessTokenLifespan: ptr.To(int32(600)),
			RevokeRefreshToken:  ptr.To(true),
		}

		MergeRealmRepresentation(&base, &overlay)

		assert.Equal(t, ptr.To(int32(600)), base.AccessTokenLifespan)
		assert.Equal(t, ptr.To(true), base.RevokeRefreshToken)
	})
}

func TestApplyRealmEventConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cfg       *common.RealmEventConfig
		setupMock func(*v2mocks.MockRealmClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "nil config — no-op",
			cfg:       nil,
			setupMock: func(_ *v2mocks.MockRealmClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "full config — SetRealmEventConfig called",
			cfg: &common.RealmEventConfig{
				AdminEventsDetailsEnabled: true,
				AdminEventsEnabled:        true,
				EventsEnabled:             true,
				EventsExpiration:          3600,
				EnabledEventTypes:         []string{"LOGIN", "LOGOUT"},
				EventsListeners:           []string{"jboss-logging"},
			},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().SetRealmEventConfig(mock.Anything, "test-realm",
					mock.MatchedBy(func(rep keycloakv2.RealmEventsConfigRepresentation) bool {
						return rep.AdminEventsDetailsEnabled != nil && *rep.AdminEventsDetailsEnabled &&
							rep.AdminEventsEnabled != nil && *rep.AdminEventsEnabled &&
							rep.EventsEnabled != nil && *rep.EventsEnabled &&
							rep.EventsExpiration != nil && *rep.EventsExpiration == 3600 &&
							rep.EnabledEventTypes != nil && len(*rep.EnabledEventTypes) == 2 &&
							rep.EventsListeners != nil && len(*rep.EventsListeners) == 1
					})).Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "config without optional slices — SetRealmEventConfig called without slice fields",
			cfg: &common.RealmEventConfig{
				EventsEnabled:    true,
				EventsExpiration: 600,
			},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().SetRealmEventConfig(mock.Anything, "test-realm",
					mock.MatchedBy(func(rep keycloakv2.RealmEventsConfigRepresentation) bool {
						return rep.EnabledEventTypes == nil && rep.EventsListeners == nil
					})).Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "SetRealmEventConfig fails — error returned",
			cfg:  &common.RealmEventConfig{EventsEnabled: true},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().SetRealmEventConfig(mock.Anything, "test-realm", mock.Anything).
					Return(nil, assert.AnError)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to set realm event config")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := v2mocks.NewMockRealmClient(t)
			tt.setupMock(m)

			err := ApplyRealmEventConfig(context.Background(), "test-realm", tt.cfg, m)
			tt.wantErr(t, err)
		})
	}
}

func TestApplyRealmSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		overlay   keycloakv2.RealmRepresentation
		setupMock func(*v2mocks.MockRealmClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:    "successful — GetRealm, merge, UpdateRealm",
			overlay: keycloakv2.RealmRepresentation{DisplayName: ptr.To("My Realm")},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "test-realm").
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "test-realm", mock.MatchedBy(func(rep keycloakv2.RealmRepresentation) bool {
					return rep.DisplayName != nil && *rep.DisplayName == "My Realm"
				})).Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name:    "GetRealm fails — error returned",
			overlay: keycloakv2.RealmRepresentation{},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "test-realm").
					Return(nil, nil, assert.AnError)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get realm")
			},
		},
		{
			name:    "UpdateRealm fails — error returned",
			overlay: keycloakv2.RealmRepresentation{},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "test-realm").
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "test-realm", mock.Anything).
					Return(nil, assert.AnError)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to update realm settings")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := v2mocks.NewMockRealmClient(t)
			tt.setupMock(m)

			err := ApplyRealmSettings(context.Background(), "test-realm", tt.overlay, m)
			tt.wantErr(t, err)
		})
	}
}
