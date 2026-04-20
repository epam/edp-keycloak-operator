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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapiMocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestProcessPermissions_Serve(t *testing.T) {
	const (
		testClientName      = "test-client"
		testClientNamespace = "default"
		permissionName      = "resource-permission"
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
			name: "permissions created successfully",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      permissionName,
								Type:      keycloakApi.PermissionTypeResource,
								Policies:  []string{"policy"},
								Resources: []string{"resource"},
							},
							{
								Name:     "scope-permission",
								Type:     keycloakApi.PermissionTypeScope,
								Policies: []string{"policy"},
								Scopes:   []string{"scope"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPermissions", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("resource-permission2-id"),
							Name: ptr.To("resource-permission2"),
						},
						{
							Id:   ptr.To("scope-permission-id"),
							Name: ptr.To("scope-permission"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakapi.ResourceRepresentation{
						{
							UnderscoreId: ptr.To("resource-id"),
							Name:         ptr.To("resource"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetPolicies", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("policy-id"),
							Name: ptr.To("policy"),
						},
					}, (*keycloakapi.Response)(nil), nil).Twice()
				authzMock.On("GetScopes", mock.Anything, "master", "clientID").
					Return([]keycloakapi.ScopeRepresentation{
						{
							Id:   ptr.To("scope-id"),
							Name: ptr.To("scope"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On(
					"UpdatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(p keycloakapi.PolicyRepresentation) bool {
						return p.Name != nil && *p.Name == "scope-permission" && p.Id != nil && *p.Id == "scope-permission-id"
					})).
					Return((*keycloakapi.Response)(nil), nil).Once()
				authzMock.On(
					"CreatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.Anything,
					mock.MatchedBy(func(p keycloakapi.PolicyRepresentation) bool {
						return p.Name != nil && *p.Name == permissionName
					})).
					Return((*keycloakapi.PolicyRepresentation)(nil), (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On(
					"DeletePermission",
					mock.Anything,
					"master",
					"clientID",
					"resource-permission2-id").
					Return((*keycloakapi.Response)(nil), nil).Once()

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: require.NoError,
		},
		{
			name: "permissions addOnly successful",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
					ClientId:               "client-2",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      permissionName,
								Type:      keycloakApi.PermissionTypeResource,
								Policies:  []string{"policy"},
								Resources: []string{"resource"},
							},
							{
								Name:     "scope-permission",
								Type:     keycloakApi.PermissionTypeScope,
								Policies: []string{"policy"},
								Scopes:   []string{"scope"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPermissions", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("resource-permission2-id"),
							Name: ptr.To("resource-permission2"),
						},
						{
							Id:   ptr.To("scope-permission-id"),
							Name: ptr.To("scope-permission"),
						},
						{
							Id:   ptr.To("token-exchange-id"),
							Name: ptr.To("token-exchange"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakapi.ResourceRepresentation{
						{
							UnderscoreId: ptr.To("resource-id"),
							Name:         ptr.To("resource"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetPolicies", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("policy-id"),
							Name: ptr.To("policy"),
						},
					}, (*keycloakapi.Response)(nil), nil).Twice()
				authzMock.On("GetScopes", mock.Anything, "master", "clientID").
					Return([]keycloakapi.ScopeRepresentation{
						{
							Id:   ptr.To("scope-id"),
							Name: ptr.To("scope"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On(
					"UpdatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(p keycloakapi.PolicyRepresentation) bool {
						return p.Name != nil && *p.Name == "scope-permission" && p.Id != nil && *p.Id == "scope-permission-id"
					})).
					Return((*keycloakapi.Response)(nil), nil).Once()
				authzMock.On(
					"CreatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.Anything,
					mock.MatchedBy(func(p keycloakapi.PolicyRepresentation) bool {
						return p.Name != nil && *p.Name == permissionName
					})).
					Return((*keycloakapi.PolicyRepresentation)(nil), (*keycloakapi.Response)(nil), nil).Once()

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to delete permission",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      permissionName,
								Type:      keycloakApi.PermissionTypeResource,
								Policies:  []string{"policy"},
								Resources: []string{"resource"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPermissions", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("resource-permission2-id"),
							Name: ptr.To("resource-permission2"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakapi.ResourceRepresentation{
						{
							UnderscoreId: ptr.To("resource-id"),
							Name:         ptr.To("resource"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetPolicies", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("policy-id"),
							Name: ptr.To("policy"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On(
					"CreatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.Anything,
					mock.MatchedBy(func(p keycloakapi.PolicyRepresentation) bool {
						return p.Name != nil && *p.Name == permissionName
					})).
					Return((*keycloakapi.PolicyRepresentation)(nil), (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On(
					"DeletePermission",
					mock.Anything,
					"master",
					"clientID",
					"resource-permission2-id").
					Return((*keycloakapi.Response)(nil), errors.New("failed to delete permission")).Once()

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to delete permission")
			},
		},
		{
			name: "failed to update permission",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      permissionName,
								Type:      keycloakApi.PermissionTypeResource,
								Policies:  []string{"policy"},
								Resources: []string{"resource"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPermissions", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To(permissionName + "-id"),
							Name: ptr.To(permissionName),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakapi.ResourceRepresentation{
						{
							UnderscoreId: ptr.To("resource-id"),
							Name:         ptr.To("resource"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetPolicies", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("policy-id"),
							Name: ptr.To("policy"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On(
					"UpdatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.Anything,
					mock.Anything,
					mock.MatchedBy(func(p keycloakapi.PolicyRepresentation) bool {
						return p.Name != nil && *p.Name == permissionName && p.Id != nil && *p.Id == permissionName+"-id"
					})).
					Return((*keycloakapi.Response)(nil), errors.New("failed to update permission")).Once()

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update permission")
			},
		},
		{
			name: "failed to create permission",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      permissionName,
								Type:      keycloakApi.PermissionTypeResource,
								Policies:  []string{"policy"},
								Resources: []string{"resource"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPermissions", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakapi.ResourceRepresentation{
						{
							UnderscoreId: ptr.To("resource-id"),
							Name:         ptr.To("resource"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetPolicies", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{
							Id:   ptr.To("policy-id"),
							Name: ptr.To("policy"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On(
					"CreatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.Anything,
					mock.MatchedBy(func(p keycloakapi.PolicyRepresentation) bool {
						return p.Name != nil && *p.Name == permissionName
					})).
					Return((*keycloakapi.PolicyRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("failed to create permission")).Once()

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create permission")
			},
		},
		{
			name: "failed to get policies",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      permissionName,
								Type:      keycloakApi.PermissionTypeResource,
								Policies:  []string{"policy"},
								Resources: []string{"resource"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPermissions", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return([]keycloakapi.ResourceRepresentation{
						{
							UnderscoreId: ptr.To("resource-id"),
							Name:         ptr.To("resource"),
						},
					}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetPolicies", mock.Anything, "master", "clientID").
					Return(nil, (*keycloakapi.Response)(nil), errors.New("failed to get policies"))

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get policies")
			},
		},
		{
			name: "failed to get resources",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      permissionName,
								Type:      keycloakApi.PermissionTypeResource,
								Policies:  []string{"policy"},
								Resources: []string{"resource"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPermissions", mock.Anything, "master", "clientID").
					Return([]keycloakapi.AbstractPolicyRepresentation{}, (*keycloakapi.Response)(nil), nil).Once()
				authzMock.On("GetResources", mock.Anything, "master", "clientID").
					Return(nil, (*keycloakapi.Response)(nil), errors.New("failed to get resources"))

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get resources")
			},
		},
		{
			name: "failed to get permissions",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      permissionName,
								Type:      keycloakApi.PermissionTypeResource,
								Policies:  []string{"policy"},
								Resources: []string{"resource"},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.KeycloakClient {
				clientsMock := keycloakapiMocks.NewMockClientsClient(t)
				authzMock := keycloakapiMocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPermissions", mock.Anything, "master", "clientID").
					Return(nil, (*keycloakapi.Response)(nil), errors.New("failed to get permissions"))

				return &keycloakapi.KeycloakClient{Clients: clientsMock, Authorization: authzMock}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get permissions")
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

			h := NewProcessPermissions(tt.kClient(t), k8sClient)
			err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloakClient, "master", &ClientContext{ClientUUID: "clientID"})

			tt.wantErr(t, err)
		})
	}
}
