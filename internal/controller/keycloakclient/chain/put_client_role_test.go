package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2Mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestPutClientRole_Serve(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))

	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		kClient        func(t *testing.T) *keycloakv2.KeycloakClient
		realmName      string
		expectedError  string
	}{
		{
			name: "success - sync client roles with ClientRoles (v1 only, no ClientRolesV2) returns early",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId:    "test-client-id",
					ClientRoles: []string{"role1", "role2"},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				return &keycloakv2.KeycloakClient{
					Clients: keycloakv2Mocks.NewMockClientsClient(t),
				}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - sync client roles with ClientRolesV2",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name:        "admin-role",
							Description: "Administrator role",
						},
						{
							Name:        "user-role",
							Description: "User role",
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("CreateClientRole", mock.Anything, "test-realm", "client-uuid", mock.MatchedBy(func(r keycloakv2.RoleRepresentation) bool {
					return r.Name != nil && *r.Name == "admin-role"
				})).Return((*keycloakv2.Response)(nil), nil)
				clientsMock.On("CreateClientRole", mock.Anything, "test-realm", "client-uuid", mock.MatchedBy(func(r keycloakv2.RoleRepresentation) bool {
					return r.Name != nil && *r.Name == "user-role"
				})).Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - no client roles",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				return &keycloakv2.KeycloakClient{
					Clients: keycloakv2Mocks.NewMockClientsClient(t),
				}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - empty client roles arrays",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId:      "test-client-id",
					ClientRoles:   []string{},
					ClientRolesV2: []keycloakApi.ClientRole{},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				return &keycloakv2.KeycloakClient{
					Clients: keycloakv2Mocks.NewMockClientsClient(t),
				}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "error - GetClientRoles fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name: "role1",
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return(([]keycloakv2.RoleRepresentation)(nil), (*keycloakv2.Response)(nil), errors.New("keycloak API error"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "unable to get existing client roles",
		},
		{
			name: "error - CreateClientRole fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name: "new-role",
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("CreateClientRole", mock.Anything, "test-realm", "client-uuid", mock.Anything).
					Return((*keycloakv2.Response)(nil), errors.New("creation failed"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "unable to create client role new-role",
		},
		{
			name: "success - deletes extra roles when not addOnly",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name:        "keep-role",
							Description: "Role to keep",
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{
						{Name: ptr.To("keep-role"), Description: ptr.To("Role to keep")},
						{Name: ptr.To("extra-role")},
					}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("DeleteClientRole", mock.Anything, "test-realm", "client-uuid", "extra-role").
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - addOnly does not delete extra roles",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId:               "test-client-id",
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name:        "keep-role",
							Description: "Role to keep",
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{
						{Name: ptr.To("keep-role"), Description: ptr.To("Role to keep")},
						{Name: ptr.To("extra-role")},
					}, (*keycloakv2.Response)(nil), nil)
				// No DeleteClientRole call expected

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - client roles with associated roles (composites)",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name:                  "composite-role",
							Description:           "Composite role",
							AssociatedClientRoles: []string{"role1", "role2"},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{
						{Name: ptr.To("composite-role"), Description: ptr.To("Composite role")},
					}, (*keycloakv2.Response)(nil), nil)
				// Sync composites
				clientsMock.On("GetClientRoleComposites", mock.Anything, "test-realm", "client-uuid", "composite-role").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("GetClientRole", mock.Anything, "test-realm", "client-uuid", "role1").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("role1-id"), Name: ptr.To("role1")}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("GetClientRole", mock.Anything, "test-realm", "client-uuid", "role2").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("role2-id"), Name: ptr.To("role2")}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("AddClientRoleComposites", mock.Anything, "test-realm", "client-uuid", "composite-role", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - composites already exist, removes extra",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name:                  "composite-role",
							Description:           "Composite role",
							AssociatedClientRoles: []string{"role1"},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{
						{Name: ptr.To("composite-role"), Description: ptr.To("Composite role")},
					}, (*keycloakv2.Response)(nil), nil)
				// Existing composites include role1 (desired) and role2 (extra)
				clientsMock.On("GetClientRoleComposites", mock.Anything, "test-realm", "client-uuid", "composite-role").
					Return([]keycloakv2.RoleRepresentation{
						{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
						{Id: ptr.To("role2-id"), Name: ptr.To("role2")},
					}, (*keycloakv2.Response)(nil), nil)
				// role1 already exists, no add needed; role2 is extra and gets removed
				clientsMock.On("DeleteClientRoleComposites", mock.Anything, "test-realm", "client-uuid", "composite-role", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "error - GetClientRoleComposites fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name:                  "composite-role",
							AssociatedClientRoles: []string{"role1"},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{
						{Name: ptr.To("composite-role"), Description: ptr.To("")},
					}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("GetClientRoleComposites", mock.Anything, "test-realm", "client-uuid", "composite-role").
					Return(([]keycloakv2.RoleRepresentation)(nil), (*keycloakv2.Response)(nil), errors.New("composites API error"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "unable to get composites for role composite-role",
		},
		{
			name: "success - updates existing role when description changes",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name:        "existing-role",
							Description: "Updated description",
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{
						{Id: ptr.To("role-id"), Name: ptr.To("existing-role"), Description: ptr.To("Old description")},
					}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("UpdateClientRole", mock.Anything, "test-realm", "client-uuid", "existing-role", mock.MatchedBy(func(r keycloakv2.RoleRepresentation) bool {
					return r.Name != nil && *r.Name == "existing-role" && r.Description != nil && *r.Description == "Updated description"
				})).Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "error - UpdateClientRole fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name:        "existing-role",
							Description: "Updated description",
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{
						{Id: ptr.To("role-id"), Name: ptr.To("existing-role"), Description: ptr.To("Old description")},
					}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("UpdateClientRole", mock.Anything, "test-realm", "client-uuid", "existing-role", mock.Anything).
					Return((*keycloakv2.Response)(nil), errors.New("update failed"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "unable to update client role existing-role",
		},
		{
			name: "error - DeleteClientRole fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ClientRolesV2: []keycloakApi.ClientRole{
						{
							Name: "keep-role",
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetClientRoles", mock.Anything, "test-realm", "client-uuid", (*keycloakv2.GetClientRolesParams)(nil)).
					Return([]keycloakv2.RoleRepresentation{
						{Name: ptr.To("keep-role"), Description: ptr.To("")},
						{Name: ptr.To("extra-role")},
					}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("DeleteClientRole", mock.Anything, "test-realm", "client-uuid", "extra-role").
					Return((*keycloakv2.Response)(nil), errors.New("delete failed"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "unable to delete client role extra-role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.keycloakClient).
				WithStatusSubresource(tt.keycloakClient).
				Build()

			kClient := tt.kClient(t)

			service := NewPutClientRole(kClient, k8sClient)

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

			err := service.Serve(ctx, tt.keycloakClient, tt.realmName, &ClientContext{ClientUUID: "client-uuid"})

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
