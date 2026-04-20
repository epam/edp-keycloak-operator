package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapiMocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestProcessScope_Serve(t *testing.T) {
	const (
		testClientName      = "test-client"
		testClientNamespace = "default"
	)

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		kClient        func(t *testing.T) *keycloakapi.KeycloakClient
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name:           "client has no authorization settings",
			keycloakClient: &keycloakApi.KeycloakClient{},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				return &keycloakapi.KeycloakClient{}
			},
			wantErr: require.NoError,
		},
		{
			name: "scope created and old scope deleted",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Scopes: []string{"scopeID1"},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetScopes", mock.Anything, "master", "clientUUID").
					Return([]keycloakapi.ScopeRepresentation{
						{Id: ptr.To("oldScopeID"), Name: ptr.To("oldScope")},
					}, (*keycloakapi.Response)(nil), nil)
				authzMock.On("CreateScope", mock.Anything, "master", "clientUUID", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil)
				authzMock.On("DeleteScope", mock.Anything, "master", "clientUUID", "oldScopeID").
					Return((*keycloakapi.Response)(nil), nil)

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: require.NoError,
		},
		{
			name: "scope already exists",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Scopes: []string{"token-exchange"},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetScopes", mock.Anything, "master", "clientUUID").
					Return([]keycloakapi.ScopeRepresentation{
						{Id: ptr.To("scopeID"), Name: ptr.To("token-exchange")},
					}, (*keycloakapi.Response)(nil), nil)

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: require.NoError,
		},
		{
			name: "scope deleted successfully",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Scopes: []string{},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetScopes", mock.Anything, "master", "clientUUID").
					Return([]keycloakapi.ScopeRepresentation{
						{Id: ptr.To("scopeID"), Name: ptr.To("token-exchange")},
					}, (*keycloakapi.Response)(nil), nil)
				authzMock.On("DeleteScope", mock.Anything, "master", "clientUUID", "scopeID").
					Return((*keycloakapi.Response)(nil), nil)

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.keycloakClient.Name == "" {
				tt.keycloakClient.Name = testClientName
			}

			if tt.keycloakClient.Namespace == "" {
				tt.keycloakClient.Namespace = testClientNamespace
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.keycloakClient).
				WithStatusSubresource(tt.keycloakClient).
				Build()

			h := NewProcessScope(tt.kClient(t), k8sClient)
			err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloakClient, "master", &ClientContext{ClientUUID: "clientUUID"})

			tt.wantErr(t, err)
		})
	}
}
