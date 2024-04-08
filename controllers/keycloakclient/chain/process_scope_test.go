package chain

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestProcessScope_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		keycloakClient    *keycloakApi.KeycloakClient
		keycloakApiClient func(t *testing.T) keycloak.Client
		wantErr           require.ErrorAssertionFunc
	}{
		{
			name:           "client has no authorization settings",
			keycloakClient: &keycloakApi.KeycloakClient{},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				return mocks.NewMockClient(t)
			},
			wantErr: require.NoError,
		},
		{
			name: "get scopes",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Scopes: []string{"scopeID1"},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ScopeRepresentation{
						"scopeID": {
							ID:   gocloak.StringP("scopeID"),
							Name: gocloak.StringP("scopeID1"),
						},
					}, nil).Once()
				client.On("CreateScope", mock.Anything, "master", "clientID", "scopeID1").Return(nil, nil).Once()
				client.On("DeleteScope", mock.Anything, "master", "clientID", "scopeID").Return(nil).Once()
				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "scope created successfully",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Scopes: []string{"token-exchange"},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ScopeRepresentation{
						"scopeID": {
							ID:   gocloak.StringP("scopeID"),
							Name: gocloak.StringP("token-exchange"),
						},
					}, nil).Once()
				client.On(
					"CreateScope",
					mock.Anything,
					"master",
					"clientID",
					"token-exchange").
					Return(&gocloak.ScopeRepresentation{Name: gocloak.StringP("token-exchange")}, nil)
				client.On("DeleteScope", mock.Anything, "master", "clientID", "scopeID").Return(nil)
				return client
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ScopeRepresentation{
						"scopeID": {
							ID:   gocloak.StringP("scopeID"),
							Name: gocloak.StringP("token-exchange"),
						},
					}, nil).Once()
				client.On(
					"DeleteScope",
					mock.Anything,
					"master",
					"clientID",
					"scopeID").
					Return(nil).Once()
				return client
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := NewProcessScope(tt.keycloakApiClient(t))
			err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloakClient, "master")

			tt.wantErr(t, err)
		})
	}
}
