package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestPutClientScope_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		client            func(t *testing.T) client.Client
		keycloakClient    client.ObjectKey
		keycloakApiClient func(t *testing.T) *mocks.MockClient
		wantErr           require.ErrorAssertionFunc
	}{
		{
			name: "with default scopes",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&keycloakApi.KeycloakClient{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-client",
							Namespace: "default",
						},
						Spec: keycloakApi.KeycloakClientSpec{
							ClientId:            "test-client-id",
							DefaultClientScopes: []string{"default-scope"},
						},
					}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddDefaultScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
		{
			name: "with optional scopes",
			client: func(t *testing.T) client.Client {
				s := runtime.NewScheme()
				require.NoError(t, keycloakApi.AddToScheme(s))
				require.NoError(t, corev1.AddToScheme(s))

				return fake.NewClientBuilder().WithScheme(s).WithObjects(
					&keycloakApi.KeycloakClient{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-client",
							Namespace: "default",
						},
						Spec: keycloakApi.KeycloakClientSpec{
							ClientId:             "test-client-id",
							OptionalClientScopes: []string{"optional-scope"},
						},
					}).Build()
			},
			keycloakClient: client.ObjectKey{
				Name:      "test-client",
				Namespace: "default",
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientScopesByNames", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
				m.On("AddOptionalScopeToClient", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				return m
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cl := &keycloakApi.KeycloakClient{}
			require.NoError(t, tt.client(t).Get(context.Background(), tt.keycloakClient, cl))

			el := NewPutClientScope(tt.keycloakApiClient(t))
			err := el.Serve(
				ctrl.LoggerInto(context.Background(), logr.Discard()),
				cl,
				"realm",
			)
			tt.wantErr(t, err)
		})
	}
}
