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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapiMocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestPutRealmRole_Serve(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		mockSetup      func(*keycloakapiMocks.MockRolesClient)
		realmName      string
		expectedError  string
	}{
		{
			name: "error - composite realm role has no id",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
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
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, testRealmName, "test-role").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).Once()
				m.On("CreateRealmRole", mock.Anything, testRealmName, mock.Anything).
					Return((*keycloakapi.Response)(nil), nil)
				// Keycloak resolves a composite by id alone, so no AddRealmRoleComposites may follow.
				m.On("GetRealmRole", mock.Anything, testRealmName, "composite-role").
					Return(&keycloakapi.RoleRepresentation{Name: ptr.To("composite-role")}, (*keycloakapi.Response)(nil), nil)
			},
			realmName:     testRealmName,
			expectedError: `composite of realm role "test-role": realm role "composite-role" has no id`,
		},
		{
			name: "success - create single realm role with composite",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
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
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				// GetRealmRole returns not found → role doesn't exist
				m.On("GetRealmRole", mock.Anything, testRealmName, "test-role").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).Once()
				m.On("CreateRealmRole", mock.Anything, testRealmName, mock.Anything).
					Return((*keycloakapi.Response)(nil), nil)
				// Get composite role
				m.On("GetRealmRole", mock.Anything, testRealmName, "composite-role").
					Return(&keycloakapi.RoleRepresentation{Id: ptr.To("comp-id"), Name: ptr.To("composite-role")}, (*keycloakapi.Response)(nil), nil)
				m.On("AddRealmRoleComposites", mock.Anything, testRealmName, "test-role", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil)
			},
			realmName:     testRealmName,
			expectedError: "",
		},
		{
			name: "success - create multiple realm roles",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
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
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, testRealmName, "role1").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).Once()
				m.On("CreateRealmRole", mock.Anything, testRealmName, mock.MatchedBy(func(r keycloakapi.RoleRepresentation) bool {
					return r.Name != nil && *r.Name == "role1"
				})).Return((*keycloakapi.Response)(nil), nil)
				m.On("GetRealmRole", mock.Anything, testRealmName, "composite1").
					Return(&keycloakapi.RoleRepresentation{Id: ptr.To("comp1-id"), Name: ptr.To("composite1")}, (*keycloakapi.Response)(nil), nil)
				m.On("AddRealmRoleComposites", mock.Anything, testRealmName, "role1", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil)

				m.On("GetRealmRole", mock.Anything, testRealmName, "role2").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).Once()
				m.On("CreateRealmRole", mock.Anything, testRealmName, mock.MatchedBy(func(r keycloakapi.RoleRepresentation) bool {
					return r.Name != nil && *r.Name == "role2"
				})).Return((*keycloakapi.Response)(nil), nil)
				m.On("GetRealmRole", mock.Anything, testRealmName, "composite2").
					Return(&keycloakapi.RoleRepresentation{Id: ptr.To("comp2-id"), Name: ptr.To("composite2")}, (*keycloakapi.Response)(nil), nil)
				m.On("AddRealmRoleComposites", mock.Anything, testRealmName, "role2", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil)
			},
			realmName:     testRealmName,
			expectedError: "",
		},
		{
			name: "success - no realm roles (nil)",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: nil,
				},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
			},
			realmName:     testRealmName,
			expectedError: "",
		},
		{
			name: "success - no realm roles (empty slice)",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{},
				},
			},
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
			},
			realmName:     testRealmName,
			expectedError: "",
		},
		{
			name: "role already exists - returns early",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
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
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, testRealmName, "existing-role").
					Return(&keycloakapi.RoleRepresentation{Id: ptr.To("id"), Name: ptr.To("existing-role")}, (*keycloakapi.Response)(nil), nil)
			},
			realmName:     testRealmName,
			expectedError: "",
		},
		{
			name: "error - GetRealmRole fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
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
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, testRealmName, "test-role").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("keycloak api error"))
			},
			realmName:     testRealmName,
			expectedError: "error checking realm role test-role",
		},
		{
			name: "error - CreateRealmRole fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
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
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, testRealmName, "test-role").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound)
				m.On("CreateRealmRole", mock.Anything, testRealmName, mock.Anything).
					Return((*keycloakapi.Response)(nil), errors.New("creation failed"))
			},
			realmName:     testRealmName,
			expectedError: "error creating realm role test-role",
		},
		{
			name: "multiple roles - first exists, second is created",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
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
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				// First role exists, skip it
				m.On("GetRealmRole", mock.Anything, testRealmName, "existing-role").
					Return(&keycloakapi.RoleRepresentation{Id: ptr.To("id"), Name: ptr.To("existing-role")}, (*keycloakapi.Response)(nil), nil)
				// Second role doesn't exist, create it
				m.On("GetRealmRole", mock.Anything, testRealmName, "new-role").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound).Once()
				m.On("CreateRealmRole", mock.Anything, testRealmName, mock.MatchedBy(func(r keycloakapi.RoleRepresentation) bool {
					return r.Name != nil && *r.Name == "new-role"
				})).Return((*keycloakapi.Response)(nil), nil)
				m.On("GetRealmRole", mock.Anything, testRealmName, "composite2").
					Return(&keycloakapi.RoleRepresentation{Id: ptr.To("comp2-id"), Name: ptr.To("composite2")}, (*keycloakapi.Response)(nil), nil)
				m.On("AddRealmRoleComposites", mock.Anything, testRealmName, "new-role", mock.Anything).
					Return((*keycloakapi.Response)(nil), nil)
			},
			realmName:     testRealmName,
			expectedError: "",
		},
		{
			name: "empty composite name - no composite added",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testClientName,
					Namespace: testClientNamespace,
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
			mockSetup: func(m *keycloakapiMocks.MockRolesClient) {
				m.On("GetRealmRole", mock.Anything, testRealmName, "test-role").
					Return((*keycloakapi.RoleRepresentation)(nil), (*keycloakapi.Response)(nil), keycloakapi.ErrNotFound)
				m.On("CreateRealmRole", mock.Anything, testRealmName, mock.Anything).
					Return((*keycloakapi.Response)(nil), nil)
				// No AddRealmRoleComposites call since composite is empty
			},
			realmName:     testRealmName,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rolesMock := keycloakapiMocks.NewMockRolesClient(t)
			tt.mockSetup(rolesMock)

			kClient := &keycloakapi.KeycloakClient{
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
