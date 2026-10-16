package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestPutRealm_ServeRequest(t *testing.T) {
	t.Parallel()

	notFoundErr := &keycloakapi.ApiError{Code: http.StatusNotFound}

	tests := []struct {
		name         string
		realm        *keycloakApi.ClusterKeycloakRealm
		mockRealm    func(t *testing.T) *v2mocks.MockRealmClient
		wantErr      require.ErrorAssertionFunc
		wantStatusID string
	}{
		{
			name: "realm already exists, skip creation",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec: keycloakApi.ClusterKeycloakRealmSpec{
					RealmName: "existing-realm",
				},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "existing-realm").
					Return(&keycloakapi.RealmRepresentation{Id: ptr.To("existing-realm-id")}, nil, nil)

				return m
			},
			wantErr:      require.NoError,
			wantStatusID: "existing-realm-id",
		},
		{
			name: "realm exists with empty body, persisted status ID is kept",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec:   keycloakApi.ClusterKeycloakRealmSpec{RealmName: "existing-realm"},
				Status: keycloakApi.ClusterKeycloakRealmStatus{ID: "persisted-realm-id"},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "existing-realm").Return(nil, nil, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, keycloakapi.ErrNilRealmRepresentation)
			},
			wantStatusID: "persisted-realm-id",
		},
		{
			name: "realm exists but credential lacks view-realm, persisted status ID is kept",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec:   keycloakApi.ClusterKeycloakRealmSpec{RealmName: "existing-realm"},
				Status: keycloakApi.ClusterKeycloakRealmStatus{ID: "persisted-realm-id"},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "existing-realm").
					Return(&keycloakapi.RealmRepresentation{Realm: ptr.To("existing-realm")}, nil, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.ErrorIs(t, err, keycloakapi.ErrRealmIDMissing)
				require.Contains(t, err.Error(), "lacks view-realm")
			},
			wantStatusID: "persisted-realm-id",
		},
		{
			name: "realm missing in Keycloak and create fails, stale status ID is cleared",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec:   keycloakApi.ClusterKeycloakRealmSpec{RealmName: "gone-realm"},
				Status: keycloakApi.ClusterKeycloakRealmStatus{ID: "stale-realm-id"},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "gone-realm").Return(nil, nil, notFoundErr).Once()
				m.EXPECT().CreateRealm(mock.Anything, mock.Anything).
					Return(nil, &keycloakapi.ApiError{Code: http.StatusForbidden})

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create realm")
			},
			wantStatusID: "",
		},
		{
			name: "realm does not exist, create successfully",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec: keycloakApi.ClusterKeycloakRealmSpec{
					RealmName: "new-realm",
				},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "new-realm").
					Return(nil, nil, notFoundErr).Once()
				m.EXPECT().CreateRealm(mock.Anything, mock.MatchedBy(func(r keycloakapi.RealmRepresentation) bool {
					return r.Realm != nil && *r.Realm == "new-realm"
				})).Return(nil, nil)
				m.EXPECT().GetRealm(mock.Anything, "new-realm").
					Return(&keycloakapi.RealmRepresentation{Id: ptr.To("created-realm-id")}, nil, nil).Once()

				return m
			},
			wantErr:      require.NoError,
			wantStatusID: "created-realm-id",
		},
		{
			name: "error on GetRealm after create",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec: keycloakApi.ClusterKeycloakRealmSpec{
					RealmName: "new-realm",
				},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "new-realm").
					Return(nil, nil, notFoundErr).Once()
				m.EXPECT().CreateRealm(mock.Anything, mock.Anything).Return(nil, nil)
				m.EXPECT().GetRealm(mock.Anything, "new-realm").
					Return(nil, nil, errors.New("connection error")).Once()

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get created realm")
			},
		},
		{
			name: "error on GetRealm (non-404)",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec: keycloakApi.ClusterKeycloakRealmSpec{
					RealmName: "test-realm",
				},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "test-realm").
					Return(nil, nil, errors.New("connection error"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to check realm existence")
			},
		},
		{
			name: "error on CreateRealm",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec: keycloakApi.ClusterKeycloakRealmSpec{
					RealmName: "new-realm",
				},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().GetRealm(mock.Anything, "new-realm").
					Return(nil, nil, notFoundErr)
				m.EXPECT().CreateRealm(mock.Anything, mock.Anything).
					Return(nil, errors.New("create failed"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create realm")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kClient := &keycloakapi.KeycloakClient{Realms: tt.mockRealm(t)}
			h := NewPutRealm()

			err := h.ServeRequest(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.realm,
				kClient,
			)

			tt.wantErr(t, err)
			require.Equal(t, tt.wantStatusID, tt.realm.Status.ID)
		})
	}
}
