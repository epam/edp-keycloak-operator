package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestPutRealm_ServeRequest(t *testing.T) {
	t.Parallel()

	notFoundErr := &keycloakapi.ApiError{Code: http.StatusNotFound}

	tests := []struct {
		name      string
		realm     *keycloakApi.ClusterKeycloakRealm
		mockRealm func(t *testing.T) *v2mocks.MockRealmClient
		wantErr   require.ErrorAssertionFunc
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
					Return(&keycloakapi.RealmRepresentation{}, nil, nil)

				return m
			},
			wantErr: require.NoError,
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
					Return(nil, nil, notFoundErr)
				m.EXPECT().CreateRealm(mock.Anything, mock.MatchedBy(func(r keycloakapi.RealmRepresentation) bool {
					return r.Realm != nil && *r.Realm == "new-realm"
				})).Return(nil, nil)

				return m
			},
			wantErr: require.NoError,
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

			keycloakAPIClient := &keycloakapi.APIClient{Realms: tt.mockRealm(t)}
			h := NewPutRealm(fake.NewClientBuilder().Build())

			err := h.ServeRequest(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.realm,
				keycloakAPIClient,
			)

			tt.wantErr(t, err)
		})
	}
}
