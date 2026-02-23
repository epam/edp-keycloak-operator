package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	v1 "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestPutRealmSettings_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		realm         *v1alpha1.ClusterKeycloakRealm
		setupMocks    func(*v2mocks.MockRealmClient)
		expectedError string
	}{
		{
			name: "successful realm settings update with minimal configuration",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
				},
			},
			setupMocks: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "test-realm").
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "test-realm", mock.Anything).
					Return(nil, nil)
			},
		},
		{
			name: "successful realm settings update with full configuration",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName:            "test-realm",
					FrontendURL:          "https://example.com",
					DisplayHTMLName:      "<div>Test</div>",
					DisplayName:          "Test Realm",
					OrganizationsEnabled: true,
					RealmEventConfig: &common.RealmEventConfig{
						AdminEventsDetailsEnabled: true,
						AdminEventsEnabled:        true,
						AdminEventsExpiration:     3600,
						EnabledEventTypes:         []string{"LOGIN", "LOGOUT"},
						EventsEnabled:             true,
						EventsExpiration:          7200,
						EventsListeners:           []string{"jboss-logging"},
					},
					Themes: &v1alpha1.ClusterRealmThemes{
						LoginTheme:        ptr.To("keycloak"),
						AccountTheme:      ptr.To("keycloak"),
						AdminConsoleTheme: ptr.To("keycloak"),
						EmailTheme:        ptr.To("keycloak"),
					},
					Localization: &v1alpha1.RealmLocalization{
						InternationalizationEnabled: ptr.To(true),
					},
					BrowserSecurityHeaders: &map[string]string{
						"X-Frame-Options": "SAMEORIGIN",
					},
					PasswordPolicies: []common.PasswordPolicy{
						{Type: "length", Value: "8"},
					},
					TokenSettings: &common.TokenSettings{
						DefaultSignatureAlgorithm: "RS256",
						RevokeRefreshToken:        true,
					},
				},
			},
			setupMocks: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().SetRealmEventConfig(mock.Anything, "test-realm", mock.Anything).
					Return(nil, nil)
				m.EXPECT().GetRealm(mock.Anything, "test-realm").
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "test-realm", mock.Anything).
					Return(nil, nil)
			},
		},
		{
			name: "error when SetRealmEventConfig fails",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					RealmEventConfig: &common.RealmEventConfig{
						EventsEnabled: true,
					},
				},
			},
			setupMocks: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().SetRealmEventConfig(mock.Anything, "test-realm", mock.Anything).
					Return(nil, assert.AnError)
			},
			expectedError: "unable to set realm event config",
		},
		{
			name: "error when GetRealm fails",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
				},
			},
			setupMocks: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "test-realm").
					Return(nil, nil, assert.AnError)
			},
			expectedError: "unable to get realm",
		},
		{
			name: "error when UpdateRealm fails",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
				},
			},
			setupMocks: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "test-realm").
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "test-realm", mock.Anything).
					Return(nil, assert.AnError)
			},
			expectedError: "unable to update realm settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRealm := v2mocks.NewMockRealmClient(t)
			tt.setupMocks(mockRealm)

			handler := NewPutRealmSettings()
			kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}

			err := handler.ServeRequest(context.Background(), tt.realm, kClientV2)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPutRealmSettings_ServeRequest_WithLogin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		realm         *v1alpha1.ClusterKeycloakRealm
		expectedError require.ErrorAssertionFunc
	}{
		{
			name: "successful realm settings update with login configuration",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Login: &v1.RealmLogin{
						UserRegistration: true,
						ForgotPassword:   true,
						RememberMe:       true,
						LoginWithEmail:   true,
						VerifyEmail:      true,
					},
				},
			},
			expectedError: require.NoError,
		},
		{
			name: "successful realm settings update with partial login configuration",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Login: &v1.RealmLogin{
						UserRegistration: true,
						RememberMe:       true,
					},
				},
			},
			expectedError: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRealm := v2mocks.NewMockRealmClient(t)
			mockRealm.EXPECT().GetRealm(mock.Anything, "test-realm").
				Return(&keycloakv2.RealmRepresentation{}, nil, nil)
			mockRealm.EXPECT().UpdateRealm(mock.Anything, "test-realm", mock.Anything).
				Return(nil, nil)

			handler := NewPutRealmSettings()
			kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}

			err := handler.ServeRequest(context.Background(), tt.realm, kClientV2)
			tt.expectedError(t, err)
		})
	}
}

func TestPutRealmSettings_ServeRequest_WithSSOSessionSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		realm         *v1alpha1.ClusterKeycloakRealm
		expectedError require.ErrorAssertionFunc
	}{
		{
			name: "successful realm settings update with all session types",
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
			expectedError: require.NoError,
		},
		{
			name: "successful realm settings update with all session types and login",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Login: &v1.RealmLogin{
						RememberMe: true,
					},
					Sessions: &common.RealmSessions{
						SSOSessionSettings: &common.RealmSSOSessionSettings{
							IdleTimeout: 1800,
							MaxLifespan: 36000,
						},
					},
				},
			},
			expectedError: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRealm := v2mocks.NewMockRealmClient(t)
			mockRealm.EXPECT().GetRealm(mock.Anything, "test-realm").
				Return(&keycloakv2.RealmRepresentation{}, nil, nil)
			mockRealm.EXPECT().UpdateRealm(mock.Anything, "test-realm", mock.Anything).
				Return(nil, nil)

			handler := NewPutRealmSettings()
			kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}

			err := handler.ServeRequest(context.Background(), tt.realm, kClientV2)
			tt.expectedError(t, err)
		})
	}
}
