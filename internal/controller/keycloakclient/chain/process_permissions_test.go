package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetPermissions", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.PermissionRepresentation{
						"resource-permission2": {
							ID:   gocloak.StringP("resource-permission2-id"),
							Name: gocloak.StringP("resource-permission2"),
						},
						"scope-permission": {
							ID:   gocloak.StringP("scope-permission-id"),
							Name: gocloak.StringP("scope permission"),
						},
					}, nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource": {
							ID:   gocloak.StringP("resource-id"),
							Name: gocloak.StringP("resource"),
						},
					}, nil).Once()
				client.On("GetPolicies", mock.Anything, "master", "clientID").
					Return(map[string]*gocloak.PolicyRepresentation{
						"policy": {
							ID:   gocloak.StringP("policy-id"),
							Name: gocloak.StringP("policy"),
						},
					}, nil).Twice()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ScopeRepresentation{
						"scope": {
							ID:   gocloak.StringP("scope-id"),
							Name: gocloak.StringP("scope"),
						},
					}, nil).Once()
				client.On(
					"UpdatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.PermissionRepresentation) bool {
						return *p.Name == "scope-permission" && *p.ID == "scope-permission-id"
					})).
					Return(nil).Once()
				client.On(
					"CreatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.PermissionRepresentation) bool {
						return p.Name != nil && *p.Name == permissionName
					})).
					Return(nil, nil).Once()
				client.On(
					"DeletePermission",
					mock.Anything,
					"master",
					"clientID",
					"resource-permission2-id").
					Return(nil).Once()

				return client
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client-2", "master").
					Return("clientID", nil).Once()
				client.On("GetPermissions", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.PermissionRepresentation{
						"resource-permission2": {
							ID:   gocloak.StringP("resource-permission2-id"),
							Name: gocloak.StringP("resource-permission2"),
						},
						"scope-permission": {
							ID:   gocloak.StringP("scope-permission-id"),
							Name: gocloak.StringP("scope permission"),
						},
						"token-exchange": {
							ID:   gocloak.StringP("token-exchange-id"),
							Name: gocloak.StringP("token exchange"),
						},
					}, nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource": {
							ID:   gocloak.StringP("resource-id"),
							Name: gocloak.StringP("resource"),
						},
					}, nil).Once()
				client.On("GetPolicies", mock.Anything, "master", "clientID").
					Return(map[string]*gocloak.PolicyRepresentation{
						"policy": {
							ID:   gocloak.StringP("policy-id"),
							Name: gocloak.StringP("policy"),
						},
					}, nil).Twice()
				client.On("GetScopes", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ScopeRepresentation{
						"scope": {
							ID:   gocloak.StringP("scope-id"),
							Name: gocloak.StringP("scope"),
						},
					}, nil).Once()
				client.On(
					"UpdatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.PermissionRepresentation) bool {
						return *p.Name == "scope-permission" && *p.ID == "scope-permission-id"
					})).
					Return(nil).Once()
				client.On(
					"CreatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.PermissionRepresentation) bool {
						return p.Name != nil && *p.Name == permissionName
					})).
					Return(nil, nil).Once()

				return client
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetPermissions", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.PermissionRepresentation{
						"resource-permission2": {
							ID:   gocloak.StringP("resource-permission2-id"),
							Name: gocloak.StringP("resource-permission2"),
						},
					}, nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource": {
							ID:   gocloak.StringP("resource-id"),
							Name: gocloak.StringP("resource"),
						},
					}, nil).Once()
				client.On("GetPolicies", mock.Anything, "master", "clientID").
					Return(map[string]*gocloak.PolicyRepresentation{
						"policy": {
							ID:   gocloak.StringP("policy-id"),
							Name: gocloak.StringP("policy"),
						},
					}, nil).Once()
				client.On(
					"CreatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.PermissionRepresentation) bool {
						return p.Name != nil && *p.Name == permissionName
					})).
					Return(nil, nil).Once()
				client.On(
					"DeletePermission",
					mock.Anything,
					"master",
					"clientID",
					"resource-permission2-id").
					Return(errors.New("failed to delete permission")).Once()

				return client
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetPermissions", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.PermissionRepresentation{
						permissionName: {
							ID:   gocloak.StringP(permissionName + "-id"),
							Name: gocloak.StringP(permissionName),
						},
					}, nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource": {
							ID:   gocloak.StringP("resource-id"),
							Name: gocloak.StringP("resource"),
						},
					}, nil).Once()
				client.On("GetPolicies", mock.Anything, "master", "clientID").
					Return(map[string]*gocloak.PolicyRepresentation{
						"policy": {
							ID:   gocloak.StringP("policy-id"),
							Name: gocloak.StringP("policy"),
						},
					}, nil).Once()
				client.On(
					"UpdatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.PermissionRepresentation) bool {
						return *p.Name == permissionName && *p.ID == permissionName+"-id"
					})).
					Return(errors.New("failed to update permission")).Once()

				return client
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetPermissions", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.PermissionRepresentation{}, nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource": {
							ID:   gocloak.StringP("resource-id"),
							Name: gocloak.StringP("resource"),
						},
					}, nil).Once()
				client.On("GetPolicies", mock.Anything, "master", "clientID").
					Return(map[string]*gocloak.PolicyRepresentation{
						"policy": {
							ID:   gocloak.StringP("policy-id"),
							Name: gocloak.StringP("policy"),
						},
					}, nil).Once()
				client.On(
					"CreatePermission",
					mock.Anything,
					"master",
					"clientID",
					mock.MatchedBy(func(p gocloak.PermissionRepresentation) bool {
						return p.Name != nil && *p.Name == permissionName
					})).
					Return(nil, errors.New("failed to create permission")).Once()

				return client
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetPermissions", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.PermissionRepresentation{}, nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource": {
							ID:   gocloak.StringP("resource-id"),
							Name: gocloak.StringP("resource"),
						},
					}, nil).Once()
				client.On("GetPolicies", mock.Anything, "master", "clientID").
					Return(map[string]*gocloak.PolicyRepresentation{
						"policy": {
							ID:   gocloak.StringP("policy-id"),
							Name: gocloak.StringP("policy"),
						},
					}, errors.New("failed to get policies"))

				return client
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetPermissions", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.PermissionRepresentation{}, nil).Once()
				client.On("GetResources", mock.Anything, "master", "clientID").
					Return(map[string]gocloak.ResourceRepresentation{
						"resource": {
							ID:   gocloak.StringP("resource-id"),
							Name: gocloak.StringP("resource"),
						},
					}, errors.New("failed to get resources"))

				return client
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("clientID", nil).Once()
				client.On("GetPermissions", mock.Anything, "master", "clientID").
					Return(nil, errors.New("failed to get permissions"))

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get permissions")
			},
		},
		{
			name: "failed to get client id",
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
			keycloakApiClient: func(t *testing.T) keycloak.Client {
				client := mocks.NewMockClient(t)
				client.On("GetClientID", "client", "master").
					Return("", errors.New("failed to get client id"))

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get client id")
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

			h := NewProcessPermissions(tt.keycloakApiClient(t), k8sClient)
			err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloakClient, "master")

			tt.wantErr(t, err)
		})
	}
}
