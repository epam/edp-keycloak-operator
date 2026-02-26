package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestAuthFlow_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		realm     *keycloakApi.ClusterKeycloakRealm
		mockRealm func(t *testing.T) *v2mocks.MockRealmClient
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name:  "realm browser flow is not provided",
			realm: &keycloakApi.ClusterKeycloakRealm{},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				return v2mocks.NewMockRealmClient(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "realm browser flow updated successfully",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec: keycloakApi.ClusterKeycloakRealmSpec{
					RealmName: "realm1",
					AuthenticationFlow: &keycloakApi.AuthenticationFlow{
						BrowserFlow: "flow-alias-1",
					},
				},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().SetRealmBrowserFlow(mock.Anything, "realm1", "flow-alias-1").
					Return(nil, nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "error on setting realm browser flow",
			realm: &keycloakApi.ClusterKeycloakRealm{
				Spec: keycloakApi.ClusterKeycloakRealmSpec{
					RealmName: "realm1",
					AuthenticationFlow: &keycloakApi.AuthenticationFlow{
						BrowserFlow: "flow-alias-1",
					},
				},
			},
			mockRealm: func(t *testing.T) *v2mocks.MockRealmClient {
				m := v2mocks.NewMockRealmClient(t)
				m.EXPECT().SetRealmBrowserFlow(mock.Anything, "realm1", "flow-alias-1").
					Return(nil, errors.New("failed to set realm browser flow"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "setting realm browser flow")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			kClientV2 := &keycloakv2.KeycloakClient{Realms: tt.mockRealm(t)}

			a := NewAuthFlow()
			err := a.ServeRequest(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.realm,
				kClientV2,
			)

			tt.wantErr(t, err)
		})
	}
}
