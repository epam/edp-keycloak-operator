package chain

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestProcessPermissions_Serve(t *testing.T) {
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
			name: "permissions created successfully",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      "resource-permission",
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
						return p.Name != nil && *p.Name == "resource-permission"
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
			name: "failed to delete permission",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "client",
					Authorization: &keycloakApi.Authorization{
						Permissions: []keycloakApi.Permission{
							{
								Name:      "resource-permission",
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
						return p.Name != nil && *p.Name == "resource-permission"
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
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
								Name:      "resource-permission",
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
						"resource-permission": {
							ID:   gocloak.StringP("resource-permission-id"),
							Name: gocloak.StringP("resource-permission"),
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
						return *p.Name == "resource-permission" && *p.ID == "resource-permission-id"
					})).
					Return(errors.New("failed to update permission")).Once()

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
								Name:      "resource-permission",
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
						return p.Name != nil && *p.Name == "resource-permission"
					})).
					Return(nil, errors.New("failed to create permission")).Once()

				return client
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
								Name:      "resource-permission",
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
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
								Name:      "resource-permission",
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
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
								Name:      "resource-permission",
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
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
								Name:      "resource-permission",
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
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get client id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := NewProcessPermissions(tt.keycloakApiClient(t))
			err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloakClient, "master")

			tt.wantErr(t, err)
		})
	}
}
