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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestAuthFlow_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		realm   *keycloakApi.ClusterKeycloakRealm
		kClient func(t *testing.T) keycloak.Client
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:  "realm browser flow is not provided",
			realm: &keycloakApi.ClusterKeycloakRealm{},
			kClient: func(t *testing.T) keycloak.Client {
				return mocks.NewMockClient(t)
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
			kClient: func(t *testing.T) keycloak.Client {
				kc := mocks.NewMockClient(t)
				kc.On("SetRealmBrowserFlow", mock.Anything, "realm1", "flow-alias-1").
					Return(nil)

				return kc
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
			kClient: func(t *testing.T) keycloak.Client {
				kc := mocks.NewMockClient(t)
				kc.On("SetRealmBrowserFlow", mock.Anything, "realm1", "flow-alias-1").
					Return(errors.New("failed to set realm browser flow"))
				return kc
			},

			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "setting realm browser flow")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := NewAuthFlow()
			err := a.ServeRequest(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				tt.realm,
				tt.kClient(t),
			)

			tt.wantErr(t, err)
		})
	}
}
