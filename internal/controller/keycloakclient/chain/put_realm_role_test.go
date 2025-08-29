package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	keycloakmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestPutRealmRole_Serve(t *testing.T) {
	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		mockSetup      func(*keycloakmocks.MockClient)
		realmName      string
		expectedError  string
		expectedCalls  int
	}{
		{
			name: "success - create single realm role",
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
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedRole := &dto.IncludedRealmRole{
					Name:      "test-role",
					Composite: "composite-role",
				}
				m.On("ExistRealmRole", "test-realm", "test-role").Return(false, nil)
				m.On("CreateIncludedRealmRole", "test-realm", expectedRole).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
			expectedCalls: 2,
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
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedRole1 := &dto.IncludedRealmRole{
					Name:      "role1",
					Composite: "composite1",
				}
				expectedRole2 := &dto.IncludedRealmRole{
					Name:      "role2",
					Composite: "composite2",
				}
				m.On("ExistRealmRole", "test-realm", "role1").Return(false, nil)
				m.On("CreateIncludedRealmRole", "test-realm", expectedRole1).Return(nil)
				m.On("ExistRealmRole", "test-realm", "role2").Return(false, nil)
				m.On("CreateIncludedRealmRole", "test-realm", expectedRole2).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
			expectedCalls: 4,
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
			mockSetup: func(m *keycloakmocks.MockClient) {
				// No mock calls expected
			},
			realmName:     "test-realm",
			expectedError: "",
			expectedCalls: 0,
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
			mockSetup: func(m *keycloakmocks.MockClient) {
				// No mock calls expected
			},
			realmName:     "test-realm",
			expectedError: "",
			expectedCalls: 0,
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
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("ExistRealmRole", "test-realm", "existing-role").Return(true, nil)
				// CreateIncludedRealmRole should not be called
			},
			realmName:     "test-realm",
			expectedError: "",
			expectedCalls: 1,
		},
		{
			name: "error - ExistRealmRole fails",
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
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("ExistRealmRole", "test-realm", "test-role").Return(false, errors.New("keycloak api error"))
			},
			realmName:     "test-realm",
			expectedError: "unable to put realm roles: error during ExistRealmRole: keycloak api error",
			expectedCalls: 1,
		},
		{
			name: "error - CreateIncludedRealmRole fails",
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
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedRole := &dto.IncludedRealmRole{
					Name:      "test-role",
					Composite: "composite",
				}
				m.On("ExistRealmRole", "test-realm", "test-role").Return(false, nil)
				m.On("CreateIncludedRealmRole", "test-realm", expectedRole).Return(errors.New("creation failed"))
			},
			realmName:     "test-realm",
			expectedError: "unable to put realm roles: error during CreateRealmRole: creation failed",
			expectedCalls: 2,
		},
		{
			name: "multiple roles - first exists, second fails on existence check",
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
							Name:      "failing-role",
							Composite: "composite2",
						},
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				// First role exists, so method returns early and second role is never checked
				m.On("ExistRealmRole", "test-realm", "existing-role").Return(true, nil)
			},
			realmName:     "test-realm",
			expectedError: "",
			expectedCalls: 1,
		},
		{
			name: "empty role name",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					RealmRoles: &[]keycloakApi.RealmRole{
						{
							Name:      "",
							Composite: "composite",
						},
					},
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedRole := &dto.IncludedRealmRole{
					Name:      "",
					Composite: "composite",
				}
				m.On("ExistRealmRole", "test-realm", "").Return(false, nil)
				m.On("CreateIncludedRealmRole", "test-realm", expectedRole).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
			expectedCalls: 2,
		},
		{
			name: "empty composite name",
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
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedRole := &dto.IncludedRealmRole{
					Name:      "test-role",
					Composite: "",
				}
				m.On("ExistRealmRole", "test-realm", "test-role").Return(false, nil)
				m.On("CreateIncludedRealmRole", "test-realm", expectedRole).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
			expectedCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := keycloakmocks.NewMockClient(t)
			tt.mockSetup(mockClient)

			// Create the service
			service := NewPutRealmRole(mockClient)

			// Set up context with logger
			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

			// Execute the method
			err := service.Serve(ctx, tt.keycloakClient, tt.realmName)

			// Assert the result
			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
