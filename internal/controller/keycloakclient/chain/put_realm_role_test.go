package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2Mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestPutRealmRole_Serve(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		mockSetup      func(*keycloakv2Mocks.MockRolesClient)
		realmName      string
		expectedError  string
	}{
		{
			name: "success - create single realm role with composite",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{
						{
							Name:      "test-role",
							Composite: "composite-role",
						},
					},
				},
			},
			mockSetup: func(m *keycloakv2Mocks.MockRolesClient) {
				// GetRealmRole returns not found → role doesn't exist
				m.On("GetRealmRole", mock.Anything, "test-realm", "test-role").
					Return((*keycloakv2.RoleRepresentation)(nil), (*keycloakv2.Response)(nil), keycloakv2.ErrNotFound).Once()
				m.On("CreateRealmRole", mock.Anything, "test-realm", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)
				// Get composite role
				m.On("GetRealmRole", mock.Anything, "test-realm", "composite-role").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("comp-id"), Name: ptr.To("composite-role")}, (*keycloakv2.Response)(nil), nil)
				m.On("AddRealmRoleComposites", mock.Anything, "test-realm", "test-role", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - create multiple realm roles",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{
						{
							Name:      "role1",
							Composite: "composite1",
						},
						{
							Name:      "role2",
							Composite: "composite2",
						},
					},
				},
			},
			mockSetup: func(m *keycloakv2Mocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, "test-realm", "role1").
					Return((*keycloakv2.RoleRepresentation)(nil), (*keycloakv2.Response)(nil), keycloakv2.ErrNotFound).Once()
				m.On("CreateRealmRole", mock.Anything, "test-realm", mock.MatchedBy(func(r keycloakv2.RoleRepresentation) bool {
					return r.Name != nil && *r.Name == "role1"
				})).Return((*keycloakv2.Response)(nil), nil)
				m.On("GetRealmRole", mock.Anything, "test-realm", "composite1").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("comp1-id"), Name: ptr.To("composite1")}, (*keycloakv2.Response)(nil), nil)
				m.On("AddRealmRoleComposites", mock.Anything, "test-realm", "role1", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				m.On("GetRealmRole", mock.Anything, "test-realm", "role2").
					Return((*keycloakv2.RoleRepresentation)(nil), (*keycloakv2.Response)(nil), keycloakv2.ErrNotFound).Once()
				m.On("CreateRealmRole", mock.Anything, "test-realm", mock.MatchedBy(func(r keycloakv2.RoleRepresentation) bool {
					return r.Name != nil && *r.Name == "role2"
				})).Return((*keycloakv2.Response)(nil), nil)
				m.On("GetRealmRole", mock.Anything, "test-realm", "composite2").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("comp2-id"), Name: ptr.To("composite2")}, (*keycloakv2.Response)(nil), nil)
				m.On("AddRealmRoleComposites", mock.Anything, "test-realm", "role2", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - no realm roles (nil)",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: nil,
				},
			},
			mockSetup: func(m *keycloakv2Mocks.MockRolesClient) {
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - no realm roles (empty slice)",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{},
				},
			},
			mockSetup: func(m *keycloakv2Mocks.MockRolesClient) {
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "role already exists - returns early",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{
						{
							Name:      "existing-role",
							Composite: "composite",
						},
					},
				},
			},
			mockSetup: func(m *keycloakv2Mocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, "test-realm", "existing-role").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("id"), Name: ptr.To("existing-role")}, (*keycloakv2.Response)(nil), nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "error - GetRealmRole fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{
						{
							Name:      "test-role",
							Composite: "composite",
						},
					},
				},
			},
			mockSetup: func(m *keycloakv2Mocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, "test-realm", "test-role").
					Return((*keycloakv2.RoleRepresentation)(nil), (*keycloakv2.Response)(nil), errors.New("keycloak api error"))
			},
			realmName:     "test-realm",
			expectedError: "error checking realm role test-role",
		},
		{
			name: "error - CreateRealmRole fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{
						{
							Name:      "test-role",
							Composite: "composite",
						},
					},
				},
			},
			mockSetup: func(m *keycloakv2Mocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, "test-realm", "test-role").
					Return((*keycloakv2.RoleRepresentation)(nil), (*keycloakv2.Response)(nil), keycloakv2.ErrNotFound)
				m.On("CreateRealmRole", mock.Anything, "test-realm", mock.Anything).
					Return((*keycloakv2.Response)(nil), errors.New("creation failed"))
			},
			realmName:     "test-realm",
			expectedError: "error creating realm role test-role",
		},
		{
			name: "multiple roles - first exists, second is created",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{
						{
							Name:      "existing-role",
							Composite: "composite1",
						},
						{
							Name:      "new-role",
							Composite: "composite2",
						},
					},
				},
			},
			mockSetup: func(m *keycloakv2Mocks.MockRolesClient) {
				// First role exists, skip it
				m.On("GetRealmRole", mock.Anything, "test-realm", "existing-role").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("id"), Name: ptr.To("existing-role")}, (*keycloakv2.Response)(nil), nil)
				// Second role doesn't exist, create it
				m.On("GetRealmRole", mock.Anything, "test-realm", "new-role").
					Return((*keycloakv2.RoleRepresentation)(nil), (*keycloakv2.Response)(nil), keycloakv2.ErrNotFound).Once()
				m.On("CreateRealmRole", mock.Anything, "test-realm", mock.MatchedBy(func(r keycloakv2.RoleRepresentation) bool {
					return r.Name != nil && *r.Name == "new-role"
				})).Return((*keycloakv2.Response)(nil), nil)
				m.On("GetRealmRole", mock.Anything, "test-realm", "composite2").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("comp2-id"), Name: ptr.To("composite2")}, (*keycloakv2.Response)(nil), nil)
				m.On("AddRealmRoleComposites", mock.Anything, "test-realm", "new-role", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "empty composite name - no composite added",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{
						{
							Name:      "test-role",
							Composite: "",
						},
					},
				},
			},
			mockSetup: func(m *keycloakv2Mocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, "test-realm", "test-role").
					Return((*keycloakv2.RoleRepresentation)(nil), (*keycloakv2.Response)(nil), keycloakv2.ErrNotFound)
				m.On("CreateRealmRole", mock.Anything, "test-realm", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)
				// No AddRealmRoleComposites call since composite is empty
			},
			realmName:     "test-realm",
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rolesMock := keycloakv2Mocks.NewMockRolesClient(t)
			tt.mockSetup(rolesMock)

			kClient := &keycloakv2.KeycloakClient{
				Roles: rolesMock,
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.keycloakClient).
				WithStatusSubresource(tt.keycloakClient).
				Build()

			service := NewPutRealmRole(kClient, k8sClient)

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

			err := service.Serve(ctx, tt.keycloakClient, tt.realmName, &ClientContext{})

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
