package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestPutRealmLocalizationTexts_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		realm     *v1alpha1.ClusterKeycloakRealm
		setupMock func(*v2mocks.MockRealmClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:      "no localization spec — no API calls",
			realm:     &v1alpha1.ClusterKeycloakRealm{},
			setupMock: func(_ *v2mocks.MockRealmClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "empty localizationTexts map — no API calls",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					Localization: &v1alpha1.RealmLocalization{
						LocalizationTexts: map[string]map[string]string{},
					},
				},
			},
			setupMock: func(_ *v2mocks.MockRealmClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "already in sync — PostRealmLocalization not called",
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &v1alpha1.RealmLocalization{
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
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &v1alpha1.RealmLocalization{
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
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &v1alpha1.RealmLocalization{
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
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &v1alpha1.RealmLocalization{
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
			realm: &v1alpha1.ClusterKeycloakRealm{
				Spec: v1alpha1.ClusterKeycloakRealmSpec{
					RealmName: "realm1",
					Localization: &v1alpha1.RealmLocalization{
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

			h := NewPutRealmLocalizationTexts()
			kClient := &keycloakapi.KeycloakClient{Realms: mockRealm}

			err := h.ServeRequest(context.Background(), tt.realm, kClient)
			tt.wantErr(t, err)
		})
	}
}
