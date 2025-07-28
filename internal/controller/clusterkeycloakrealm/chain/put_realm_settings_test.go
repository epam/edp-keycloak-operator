package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestPutRealmSettings_ServeRequest(t *testing.T) {
	tests := []struct {
		name          string
		realm         *v1alpha1.ClusterKeycloakRealm
		setupMocks    func(*mocks.MockClient)
		expectedError string
		expectedCalls []string
	}{
		{
			name: "successful realm settings update with minimal configuration",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
				},
			},
			setupMocks: func(mockClient *mocks.MockClient) {
				mockClient.On("UpdateRealmSettings", "test-realm", &adapter.RealmSettings{
					FrontendURL:     "",
					DisplayHTMLName: "",
					DisplayName:     "",
				}).Return(nil)
				mockClient.On("SetRealmOrganizationsEnabled", mock.Anything, "test-realm", false).Return(nil)
			},
			expectedCalls: []string{"UpdateRealmSettings", "SetRealmOrganizationsEnabled"},
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
					RealmEventConfig: &v1alpha1.RealmEventConfig{
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
						"X-Frame-Options":        "SAMEORIGIN",
						"X-Content-Type-Options": "nosniff",
					},
					PasswordPolicies: []v1alpha1.PasswordPolicy{
						{Type: "length", Value: "8"},
						{Type: "uppercase", Value: "1"},
					},
					TokenSettings: &common.TokenSettings{
						DefaultSignatureAlgorithm:           "RS256",
						RevokeRefreshToken:                  true,
						RefreshTokenMaxReuse:                0,
						AccessTokenLifespan:                 300,
						AccessTokenLifespanForImplicitFlow:  900,
						AccessCodeLifespan:                  60,
						ActionTokenGeneratedByUserLifespan:  300,
						ActionTokenGeneratedByAdminLifespan: 300,
					},
				},
			},
			setupMocks: func(mockClient *mocks.MockClient) {
				mockClient.On("SetRealmEventConfig", "test-realm", &adapter.RealmEventConfig{
					AdminEventsDetailsEnabled: true,
					AdminEventsEnabled:        true,
					EnabledEventTypes:         []string{"LOGIN", "LOGOUT"},
					EventsEnabled:             true,
					EventsExpiration:          7200,
					EventsListeners:           []string{"jboss-logging"},
				}).Return(nil)
				mockClient.On("UpdateRealmSettings", "test-realm", &adapter.RealmSettings{
					FrontendURL:     "https://example.com",
					DisplayHTMLName: "<div>Test</div>",
					DisplayName:     "Test Realm",
					Themes: &adapter.RealmThemes{
						LoginTheme:                  ptr.To("keycloak"),
						AccountTheme:                ptr.To("keycloak"),
						AdminConsoleTheme:           ptr.To("keycloak"),
						EmailTheme:                  ptr.To("keycloak"),
						InternationalizationEnabled: ptr.To(true),
					},
					BrowserSecurityHeaders: &map[string]string{
						"X-Frame-Options":        "SAMEORIGIN",
						"X-Content-Type-Options": "nosniff",
					},
					PasswordPolicies: []adapter.PasswordPolicy{
						{Type: "length", Value: "8"},
						{Type: "uppercase", Value: "1"},
					},
					TokenSettings: &adapter.TokenSettings{
						DefaultSignatureAlgorithm:           "RS256",
						RevokeRefreshToken:                  true,
						RefreshTokenMaxReuse:                0,
						AccessTokenLifespan:                 300,
						AccessTokenLifespanForImplicitFlow:  900,
						AccessCodeLifespan:                  60,
						ActionTokenGeneratedByUserLifespan:  300,
						ActionTokenGeneratedByAdminLifespan: 300,
					},
					AdminEventsExpiration: ptr.To(3600),
				}).Return(nil)
				mockClient.On("SetRealmOrganizationsEnabled", mock.Anything, "test-realm", true).Return(nil)
			},
			expectedCalls: []string{"SetRealmEventConfig", "UpdateRealmSettings", "SetRealmOrganizationsEnabled"},
		},
		{
			name: "successful realm settings update with themes only",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Themes: &v1alpha1.ClusterRealmThemes{
						LoginTheme: ptr.To("custom-theme"),
					},
				},
			},
			setupMocks: func(mockClient *mocks.MockClient) {
				mockClient.On("UpdateRealmSettings", "test-realm", &adapter.RealmSettings{
					FrontendURL:     "",
					DisplayHTMLName: "",
					DisplayName:     "",
					Themes: &adapter.RealmThemes{
						LoginTheme: ptr.To("custom-theme"),
					},
				}).Return(nil)
				mockClient.On("SetRealmOrganizationsEnabled", mock.Anything, "test-realm", false).Return(nil)
			},
			expectedCalls: []string{"UpdateRealmSettings", "SetRealmOrganizationsEnabled"},
		},
		{
			name: "successful realm settings update with password policies only",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					PasswordPolicies: []v1alpha1.PasswordPolicy{
						{Type: "length", Value: "10"},
					},
				},
			},
			setupMocks: func(mockClient *mocks.MockClient) {
				mockClient.On("UpdateRealmSettings", "test-realm", &adapter.RealmSettings{
					FrontendURL:     "",
					DisplayHTMLName: "",
					DisplayName:     "",
					PasswordPolicies: []adapter.PasswordPolicy{
						{Type: "length", Value: "10"},
					},
				}).Return(nil)
				mockClient.On("SetRealmOrganizationsEnabled", mock.Anything, "test-realm", false).Return(nil)
			},
			expectedCalls: []string{"UpdateRealmSettings", "SetRealmOrganizationsEnabled"},
		},
		{
			name: "error when SetRealmEventConfig fails",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					RealmEventConfig: &v1alpha1.RealmEventConfig{
						EventsEnabled: true,
					},
				},
			},
			setupMocks: func(mockClient *mocks.MockClient) {
				mockClient.On("SetRealmEventConfig", "test-realm", &adapter.RealmEventConfig{
					AdminEventsDetailsEnabled: false,
					AdminEventsEnabled:        false,
					EnabledEventTypes:         nil,
					EventsEnabled:             true,
					EventsExpiration:          0,
					EventsListeners:           nil,
				}).Return(errors.New("event config error"))
			},
			expectedError: "failed to set realm event config: event config error",
		},
		{
			name: "error when UpdateRealmSettings fails",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
				},
			},
			setupMocks: func(mockClient *mocks.MockClient) {
				mockClient.On("UpdateRealmSettings", "test-realm", &adapter.RealmSettings{
					FrontendURL:     "",
					DisplayHTMLName: "",
					DisplayName:     "",
				}).Return(errors.New("update settings error"))
			},
			expectedError: "unable to update realm settings: update settings error",
		},
		{
			name: "error when SetRealmOrganizationsEnabled fails",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
				},
			},
			setupMocks: func(mockClient *mocks.MockClient) {
				mockClient.On("UpdateRealmSettings", "test-realm", &adapter.RealmSettings{
					FrontendURL:     "",
					DisplayHTMLName: "",
					DisplayName:     "",
				}).Return(nil)
				mockClient.On("SetRealmOrganizationsEnabled", mock.Anything, "test-realm", false).Return(errors.New("organizations error"))
			},
			expectedError: "unable to set realm organizations enabled: organizations error",
		},
		{
			name: "successful realm settings update with localization only",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					Themes: &v1alpha1.ClusterRealmThemes{
						LoginTheme: ptr.To("keycloak"),
					},
					Localization: &v1alpha1.RealmLocalization{
						InternationalizationEnabled: ptr.To(true),
					},
				},
			},
			setupMocks: func(mockClient *mocks.MockClient) {
				mockClient.On("UpdateRealmSettings", "test-realm", &adapter.RealmSettings{
					FrontendURL:     "",
					DisplayHTMLName: "",
					DisplayName:     "",
					Themes: &adapter.RealmThemes{
						LoginTheme:                  ptr.To("keycloak"),
						InternationalizationEnabled: ptr.To(true),
					},
				}).Return(nil)
				mockClient.On("SetRealmOrganizationsEnabled", mock.Anything, "test-realm", false).Return(nil)
			},
			expectedCalls: []string{"UpdateRealmSettings", "SetRealmOrganizationsEnabled"},
		},
		{
			name: "successful realm settings update with admin events expiration",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
					RealmEventConfig: &v1alpha1.RealmEventConfig{
						AdminEventsEnabled:    true,
						AdminEventsExpiration: 1800,
					},
				},
			},
			setupMocks: func(mockClient *mocks.MockClient) {
				mockClient.On("SetRealmEventConfig", "test-realm", &adapter.RealmEventConfig{
					AdminEventsDetailsEnabled: false,
					AdminEventsEnabled:        true,
					EnabledEventTypes:         nil,
					EventsEnabled:             false,
					EventsExpiration:          0,
					EventsListeners:           nil,
				}).Return(nil)
				mockClient.On("UpdateRealmSettings", "test-realm", &adapter.RealmSettings{
					FrontendURL:           "",
					DisplayHTMLName:       "",
					DisplayName:           "",
					AdminEventsExpiration: ptr.To(1800),
				}).Return(nil)
				mockClient.On("SetRealmOrganizationsEnabled", mock.Anything, "test-realm", false).Return(nil)
			},
			expectedCalls: []string{"SetRealmEventConfig", "UpdateRealmSettings", "SetRealmOrganizationsEnabled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := mocks.NewMockClient(t)

			// Setup mocks
			if tt.setupMocks != nil {
				tt.setupMocks(mockClient)
			}

			// Create handler
			handler := NewPutRealmSettings()

			// Execute test
			ctx := context.Background()
			err := handler.ServeRequest(ctx, tt.realm, mockClient)

			// Assert results
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify all expected calls were made
			mockClient.AssertExpectations(t)
		})
	}
}
