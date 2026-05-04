package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestRealmLocalizationTexts_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		realm     *keycloakApi.KeycloakRealm
		setupMock func(*v2mocks.MockRealmClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "no localization spec — no API calls",
			realm:     &keycloakApi.KeycloakRealm{},
			setupMock: func(_ *v2mocks.MockRealmClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "empty localizationTexts map — no API calls",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					Localization: &keycloakApi.RealmLocalization{
						LocalizationTexts: map[string]map[string]string{},
					},
				},
			},
			setupMock: func(_ *v2mocks.MockRealmClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "already in sync — PostRealmLocalization not called",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &keycloakApi.RealmLocalization{
						LocalizationTexts: map[string]map[string]string{
							"en": {"hello": "Hello"},
						},
					},
				},
			},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealmLocalization(mock.Anything, "realm1", "en").
					Return(map[string]string{"hello": "Hello", "extra": "kept"}, nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "out of sync — PostRealmLocalization called",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &keycloakApi.RealmLocalization{
						LocalizationTexts: map[string]map[string]string{
							"en": {"hello": "Hello"},
							"fr": {"hello": "Bonjour"},
						},
					},
				},
			},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealmLocalization(mock.Anything, "realm1", "en").
					Return(map[string]string{"hello": "old"}, nil, nil)
				m.EXPECT().PostRealmLocalization(mock.Anything, "realm1", "en", map[string]string{"hello": "Hello"}).
					Return(nil, nil)
				m.EXPECT().GetRealmLocalization(mock.Anything, "realm1", "fr").
					Return(map[string]string{}, nil, nil)
				m.EXPECT().PostRealmLocalization(mock.Anything, "realm1", "fr", map[string]string{"hello": "Bonjour"}).
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "GetRealmLocalization error — PostRealmLocalization still called",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &keycloakApi.RealmLocalization{
						LocalizationTexts: map[string]map[string]string{
							"en": {"k": "v"},
						},
					},
				},
			},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealmLocalization(mock.Anything, "realm1", "en").
					Return(nil, nil, assert.AnError)
				m.EXPECT().PostRealmLocalization(mock.Anything, "realm1", "en", map[string]string{"k": "v"}).
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "PostRealmLocalization error — error propagated",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &keycloakApi.RealmLocalization{
						LocalizationTexts: map[string]map[string]string{
							"en": {"k": "v"},
						},
					},
				},
			},
			setupMock: func(m *v2mocks.MockRealmClient) {
				m.EXPECT().GetRealmLocalization(mock.Anything, "realm1", "en").
					Return(nil, nil, assert.AnError)
				m.EXPECT().PostRealmLocalization(mock.Anything, "realm1", "en", mock.Anything).
					Return(nil, assert.AnError)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to set realm localization for locale")
			},
		},
		{
			name: "empty locale kv map — skipped",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &keycloakApi.RealmLocalization{
						LocalizationTexts: map[string]map[string]string{
							"en": {},
						},
					},
				},
			},
			setupMock: func(_ *v2mocks.MockRealmClient) {},
			wantErr:   require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockRealm := v2mocks.NewMockRealmClient(t)
			tt.setupMock(mockRealm)

			h := RealmLocalizationTexts{}
			kClient := &keycloakapi.KeycloakClient{Realms: mockRealm}

			err := h.ServeRequest(context.Background(), tt.realm, kClient)
			tt.wantErr(t, err)
		})
	}
}

func TestLocalesAlreadyInSync(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		current map[string]string
		desired map[string]string
		want    bool
	}{
		{
			name:    "all desired keys present with same values",
			current: map[string]string{"a": "1", "b": "2", "c": "3"},
			desired: map[string]string{"a": "1", "b": "2"},
			want:    true,
		},
		{
			name:    "one value differs",
			current: map[string]string{"a": "1", "b": "old"},
			desired: map[string]string{"a": "1", "b": "2"},
			want:    false,
		},
		{
			name:    "key missing from current",
			current: map[string]string{"a": "1"},
			desired: map[string]string{"a": "1", "b": "2"},
			want:    false,
		},
		{
			name:    "both empty",
			current: map[string]string{},
			desired: map[string]string{},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, localesAlreadyInSync(tt.current, tt.desired))
		})
	}
}
