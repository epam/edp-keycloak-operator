package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestRealmSettings_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		realm     *keycloakApi.KeycloakRealm
		setupMock func(*v2mocks.MockRealmClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:  "minimal realm — no event config",
			realm: &keycloakApi.KeycloakRealm{},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "").
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "", mock.Anything).
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "with themes, security headers and password policies",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm1",
					Themes: &keycloakApi.RealmThemes{
						LoginTheme: ptr.To("LoginTheme test"),
					},
					BrowserSecurityHeaders: &map[string]string{
						"foo": "bar",
					},
					PasswordPolicies: []common.PasswordPolicy{
						{Type: "foo", Value: "bar"},
					},
					DisplayHTMLName: "<div class=\"kc-logo-text\"><span>Example</span></div>",
					FrontendURL:     "http://example.com",
				},
			},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "realm1").
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "realm1", mock.Anything).
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "with event config — SetRealmEventConfig called",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm1",
					RealmEventConfig: &common.RealmEventConfig{
						EventsListeners:       []string{"foo", "bar"},
						AdminEventsEnabled:    true,
						AdminEventsExpiration: 100,
					},
				},
			},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().SetRealmEventConfig(mock.Anything, "realm1", mock.Anything).
					Return(nil, nil)
				m.EXPECT().GetRealm(mock.Anything, "realm1").
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "realm1", mock.Anything).
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "SetRealmEventConfig fails",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm1",
					RealmEventConfig: &common.RealmEventConfig{
						EventsListeners:    []string{"foo", "bar"},
						AdminEventsEnabled: true,
					},
				},
			},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().SetRealmEventConfig(mock.Anything, "realm1", mock.Anything).
					Return(nil, assert.AnError)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to set realm event config")
			},
		},
		{
			name:  "GetRealm fails",
			realm: &keycloakApi.KeycloakRealm{},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "").
					Return(nil, nil, assert.AnError)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get realm")
			},
		},
		{
			name:  "UpdateRealm fails",
			realm: &keycloakApi.KeycloakRealm{},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealm(mock.Anything, "").
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				m.EXPECT().UpdateRealm(mock.Anything, "", mock.Anything).
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

			mockRealm := v2mocks.NewMockRealmClient(t)
			tt.setupMock(mockRealm)

			rs := RealmSettings{}
			kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}

			err := rs.ServeRequest(context.Background(), tt.realm, kClientV2)
			tt.wantErr(t, err)
		})
	}
}

func TestRealmSettings_ServeRequest_WithLogin(t *testing.T) {
	t.Parallel()

	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm-with-login",
			Login: &keycloakApi.RealmLogin{
				UserRegistration: true,
				ForgotPassword:   true,
				RememberMe:       true,
				LoginWithEmail:   true,
				VerifyEmail:      true,
			},
		},
	}

	mockRealm := v2mocks.NewMockRealmClient(t)
	mockRealm.EXPECT().GetRealm(mock.Anything, "realm-with-login").
		Return(&keycloakv2.RealmRepresentation{}, nil, nil)
	mockRealm.EXPECT().UpdateRealm(mock.Anything, "realm-with-login", mock.Anything).
		Return(nil, nil)

	rs := RealmSettings{}
	kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}

	err := rs.ServeRequest(context.Background(), &realm, kClientV2)
	require.NoError(t, err)
}

func TestRealmSettings_ServeRequest_WithSSOSessionSettings(t *testing.T) {
	t.Parallel()

	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm-with-sso-session",
			Login: &keycloakApi.RealmLogin{
				RememberMe: true,
			},
			Sessions: &common.RealmSessions{
				SSOSessionSettings: &common.RealmSSOSessionSettings{
					IdleTimeout:           1801,
					MaxLifespan:           36002,
					IdleTimeoutRememberMe: 3603,
					MaxLifespanRememberMe: 72004,
				},
				SSOOfflineSessionSettings: &common.RealmSSOOfflineSessionSettings{
					IdleTimeout:        2592007,
					MaxLifespanEnabled: true,
					MaxLifespan:        5184008,
				},
				SSOLoginSettings: &common.RealmSSOLoginSettings{
					AccessCodeLifespanLogin:      1809,
					AccessCodeLifespanUserAction: 310,
				},
			},
		},
	}

	mockRealm := v2mocks.NewMockRealmClient(t)
	mockRealm.EXPECT().GetRealm(mock.Anything, "realm-with-sso-session").
		Return(&keycloakv2.RealmRepresentation{}, nil, nil)
	mockRealm.EXPECT().UpdateRealm(mock.Anything, "realm-with-sso-session", mock.Anything).
		Return(nil, nil)

	rs := RealmSettings{}
	kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}

	err := rs.ServeRequest(context.Background(), &realm, kClientV2)
	require.NoError(t, err)
}
