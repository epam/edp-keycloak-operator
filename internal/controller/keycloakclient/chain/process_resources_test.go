package chain

import (
	"context"
	"errors"
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
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2Mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestProcessResources_Serve(t *testing.T) {
	const (
		testClientName      = "test-client"
		testClientNamespace = "default"
		resourceName        = "resource-1"
	)

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		kClient        func(t *testing.T) *keycloakv2.KeycloakClient
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name:           "client has no authorization settings",
			keycloakClient: &keycloakApi.KeycloakClient{},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				return &keycloakv2.KeycloakClient{}
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakv2.ResourceRepresentation{
						{UnderscoreId: ptr.To("resource-resource2-id"), Name: ptr.To("resource-2")},
						{UnderscoreId: ptr.To("resource-resource3-id"), Name: ptr.To("resource-3")},
						{UnderscoreId: ptr.To("resource-default-id"), Name: ptr.To("Default Resource")},
					}, (*keycloakv2.Response)(nil), nil)
				authzMock.On("GetScopes", mock.Anything, "master", "clientID").
					Return([]keycloakv2.ScopeRepresentation{
						{Id: ptr.To("scope1-id"), Name: ptr.To("scope1")},
						{Id: ptr.To("scope2-id"), Name: ptr.To("scope2")},
					}, (*keycloakv2.Response)(nil), nil)
				authzMock.On("CreateResource", mock.Anything, "master", "clientID",
					mock.MatchedBy(func(r keycloakv2.ResourceRepresentation) bool {
						return r.Name != nil && *r.Name == resourceName
					})).
					Return((*keycloakv2.ResourceRepresentation)(nil), (*keycloakv2.Response)(nil), nil)
				authzMock.On("UpdateResource", mock.Anything, "master", "clientID", "resource-resource2-id",
					mock.MatchedBy(func(r keycloakv2.ResourceRepresentation) bool {
						return r.Name != nil && *r.Name == "resource-2"
					})).
					Return((*keycloakv2.Response)(nil), nil)
				authzMock.On("DeleteResource", mock.Anything, "master", "clientID", "resource-resource3-id").
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: require.NoError,
		},
		{
			name: "resources addOnly successful",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId:               "client-1",
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakv2.ResourceRepresentation{
						{UnderscoreId: ptr.To("resource-resource2-id"), Name: ptr.To("resource-2")},
						{UnderscoreId: ptr.To("resource-resource3-id"), Name: ptr.To("resource-3")},
						{UnderscoreId: ptr.To("resource-default-id"), Name: ptr.To("Default Resource")},
						{UnderscoreId: ptr.To("resource.idp-id"), Name: ptr.To("resource.idp")},
					}, (*keycloakv2.Response)(nil), nil)
				authzMock.On("GetScopes", mock.Anything, "master", "clientID").
					Return([]keycloakv2.ScopeRepresentation{
						{Id: ptr.To("scope1-id"), Name: ptr.To("scope1")},
						{Id: ptr.To("scope2-id"), Name: ptr.To("scope2")},
					}, (*keycloakv2.Response)(nil), nil)
				authzMock.On("CreateResource", mock.Anything, "master", "clientID",
					mock.MatchedBy(func(r keycloakv2.ResourceRepresentation) bool {
						return r.Name != nil && *r.Name == resourceName
					})).
					Return((*keycloakv2.ResourceRepresentation)(nil), (*keycloakv2.Response)(nil), nil)
				authzMock.On("UpdateResource", mock.Anything, "master", "clientID", "resource-resource2-id",
					mock.MatchedBy(func(r keycloakv2.ResourceRepresentation) bool {
						return r.Name != nil && *r.Name == "resource-2"
					})).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
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
								Name: resourceName,
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakv2.ResourceRepresentation{
						{UnderscoreId: ptr.To("resource-resource2-id"), Name: ptr.To("resource-2")},
					}, (*keycloakv2.Response)(nil), nil)
				authzMock.On("CreateResource", mock.Anything, "master", "clientID",
					mock.MatchedBy(func(r keycloakv2.ResourceRepresentation) bool {
						return r.Name != nil && *r.Name == resourceName
					})).
					Return((*keycloakv2.ResourceRepresentation)(nil), (*keycloakv2.Response)(nil), nil)
				authzMock.On("DeleteResource", mock.Anything, "master", "clientID", "resource-resource2-id").
					Return((*keycloakv2.Response)(nil), errors.New("failed to delete resource"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
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
								Name:        resourceName,
								DisplayName: "Resource 1",
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakv2.ResourceRepresentation{
						{UnderscoreId: ptr.To(resourceName + "-id"), Name: ptr.To(resourceName)},
					}, (*keycloakv2.Response)(nil), nil)
				authzMock.On("UpdateResource", mock.Anything, "master", "clientID", resourceName+"-id",
					mock.MatchedBy(func(r keycloakv2.ResourceRepresentation) bool {
						return r.Name != nil && *r.Name == resourceName
					})).
					Return((*keycloakv2.Response)(nil), errors.New("failed to update resource"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
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
								Name:        resourceName,
								DisplayName: "Resource 1",
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakv2.ResourceRepresentation{}, (*keycloakv2.Response)(nil), nil)
				authzMock.On("CreateResource", mock.Anything, "master", "clientID",
					mock.MatchedBy(func(r keycloakv2.ResourceRepresentation) bool {
						return r.Name != nil && *r.Name == resourceName
					})).
					Return((*keycloakv2.ResourceRepresentation)(nil), (*keycloakv2.Response)(nil), errors.New("failed to create resource"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
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
								Name:        resourceName,
								DisplayName: "Resource 1",
								Scopes:      []string{"scope1"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakv2.ResourceRepresentation{}, (*keycloakv2.Response)(nil), nil)
				authzMock.On("GetScopes", mock.Anything, "master", "clientID").
					Return(nil, (*keycloakv2.Response)(nil), errors.New("failed to get scopes"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get scopes")
			},
		},
		{
			name: "existing resource has no ID",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Resources: []keycloakApi.Resource{
							{Name: resourceName},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakv2.ResourceRepresentation{
						{UnderscoreId: nil, Name: ptr.To(resourceName)},
					}, (*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "has no ID")
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
								Name: resourceName,
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return(nil, (*keycloakv2.Response)(nil), errors.New("failed to get resources"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get resources")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Ensure test client has proper metadata
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

			h := NewProcessResources(tt.kClient(t), k8sClient)
			err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloakClient, "master", &ClientContext{ClientUUID: "clientID"})

			tt.wantErr(t, err)
		})
	}
}
