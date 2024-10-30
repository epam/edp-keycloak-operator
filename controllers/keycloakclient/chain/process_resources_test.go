package chain

import (
	"context"
	"errors"
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

func TestProcessResources_Serve(t *testing.T) {
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
			name: "resources created/updated successfully",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Resources: []keycloakApi.Resource{
							{
								Name:               "resource-1",
								DisplayName:        "Resource 1",
								Type:               "resource",
								IconURI:            "https://icon.uri",
								OwnerManagedAccess: true,
								URIs:               []string{"https://example.com", "https://example2.com"},
								Attributes:         map[string][]string{"key": {"value1", "value2"}},
								Scopes:             []string{"scope1", "scope2"},
							},
							{
								Name: "resource-2",
								Type: "resource",
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource-2": {
							ID:   gocloak.StringP("resource-resource2-id"),
							Name: gocloak.StringP("resource-2"),
						},
						"resource-3": {
							ID:   gocloak.StringP("resource-resource3-id"),
							Name: gocloak.StringP("resource-3"),
						},
						"Default Resource": {
							ID:   gocloak.StringP("resource-default-id"),
							Name: gocloak.StringP("Default Resource"),
						},
					}, nil).Once()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ScopeRepresentation{
						"scope1": {
							ID:   gocloak.StringP("scope1-id"),
							Name: gocloak.StringP("scope1"),
						},
						"scope2": {
							ID:   gocloak.StringP("scope2-id"),
							Name: gocloak.StringP("scope2"),
						},
					}, nil).Once()
				client.On(
					"CreateResource",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.ResourceRepresentation) bool {
						return p.Name != nil && *p.Name == "resource-1"
					})).
					Return(nil, nil).Once()
				client.On(
					"UpdateResource",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.ResourceRepresentation) bool {
						return p.Name != nil && *p.Name == "resource-2"
					})).
					Return(nil, nil).Once()
				client.On(
					"DeleteResource",
					mock.Anything,
					"master",
					"clientID",
					"resource-resource3-id").
					Return(nil).Once()

				return client
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to delete resource",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Resources: []keycloakApi.Resource{
							{
								Name: "resource-1",
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource-2": {
							ID:   gocloak.StringP("resource-resource2-id"),
							Name: gocloak.StringP("resource-2"),
						},
					}, nil).Once()
				client.On(
					"CreateResource",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.ResourceRepresentation) bool {
						return p.Name != nil && *p.Name == "resource-1"
					})).
					Return(nil, nil).Once()
				client.On(
					"DeleteResource",
					mock.Anything,
					"master",
					"clientID",
					"resource-resource2-id").
					Return(errors.New("failed to delete resource")).Once()

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to delete resource")
			},
		},
		{
			name: "failed to update resource",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Resources: []keycloakApi.Resource{
							{
								Name:        "resource-1",
								DisplayName: "Resource 1",
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource-1": {
							ID:   gocloak.StringP("resource-1-id"),
							Name: gocloak.StringP("resource-1"),
						},
					}, nil).Once()
				client.On(
					"UpdateResource",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.ResourceRepresentation) bool {
						return *p.Name == "resource-1" && *p.ID == "resource-1-id"
					})).
					Return(errors.New("failed to update resource")).Once()

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update resource")
			},
		},
		{
			name: "failed to create resource",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Resources: []keycloakApi.Resource{
							{
								Name:        "resource-1",
								DisplayName: "Resource 1",
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{}, nil).Once()
				client.On(
					"CreateResource",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.ResourceRepresentation) bool {
						return p.Name != nil && *p.Name == "resource-1"
					})).
					Return(nil, errors.New("failed to create resource")).Once()

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create resource")
			},
		},
		{
			name: "failed to get scopes",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Resources: []keycloakApi.Resource{
							{
								Name:        "resource-1",
								DisplayName: "Resource 1",
								Scopes:      []string{"scope1"},
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{}, nil).Once()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(nil, errors.New("failed to get scopes"))

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get scopes")
			},
		},
		{
			name: "failed to get resources",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Resources: []keycloakApi.Resource{
							{
								Name: "resource-1",
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(nil, errors.New("failed to get resources")).Once()

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get resources")
			},
		},
		{
			name: "failed to get client id",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Resources: []keycloakApi.Resource{
							{
								Name: "resource-1",
							},
						},
					},
				},
			},
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("", errors.New("failed to get client id"))

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get client id")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewProcessResources(tt.keycloakApiClient(t))
			err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloakClient, "master")

			tt.wantErr(t, err)
		})
	}
}
