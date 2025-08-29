package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestServiceAccount_Serve(t *testing.T) {
	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		mockSetup      func(*keycloakmocks.MockClient)
		realmName      string
		expectedError  string
	}{
		{
			name: "success - service account disabled (nil)",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: nil,
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				// No mock calls expected
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - service account disabled (enabled=false)",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled: false,
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				// No mock calls expected
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "error - service account with public client",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					Public: true,
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled: true,
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				// No mock calls expected - should fail before calling API
			},
			realmName:     "test-realm",
			expectedError: "service account can not be configured with public client",
		},
		{
			name: "success - basic service account with full reconciliation",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyFull,
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1", "realm-role2"},
						ClientRoles: []keycloakApi.UserClientRole{
							{
								ClientID: "client1",
								Roles:    []string{"role1", "role2"},
							},
							{
								ClientID: "client2",
								Roles:    []string{"role3"},
							},
						},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedClientRoles := map[string][]string{
					"client1": {"role1", "role2"},
					"client2": {"role3"},
				}
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1", "realm-role2"}, expectedClientRoles, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - basic service account with add-only reconciliation",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
						ClientRoles: []keycloakApi.UserClientRole{
							{
								ClientID: "client1",
								Roles:    []string{"role1"},
							},
						},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedClientRoles := map[string][]string{
					"client1": {"role1"},
				}
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, expectedClientRoles, true).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - service account with groups",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
						Groups:     []string{"group1", "group2"},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
				m.On("SyncServiceAccountGroups", "test-realm", "client-123", []string{"group1", "group2"}, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - service account with attributes",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
						AttributesV2: map[string][]string{
							"attr1": {"value1", "value2"},
							"attr2": {"value3"},
						},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedAttributes := map[string][]string{
					"attr1": {"value1", "value2"},
					"attr2": {"value3"},
				}
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
				m.On("SetServiceAccountAttributes", "test-realm", "client-123", expectedAttributes, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - full service account with all features",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1", "realm-role2"},
						ClientRoles: []keycloakApi.UserClientRole{
							{
								ClientID: "client1",
								Roles:    []string{"role1", "role2"},
							},
						},
						Groups: []string{"group1", "group2"},
						AttributesV2: map[string][]string{
							"attr1": {"value1"},
							"attr2": {"value2", "value3"},
						},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedClientRoles := map[string][]string{
					"client1": {"role1", "role2"},
				}
				expectedAttributes := map[string][]string{
					"attr1": {"value1"},
					"attr2": {"value2", "value3"},
				}
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1", "realm-role2"}, expectedClientRoles, true).Return(nil)
				m.On("SyncServiceAccountGroups", "test-realm", "client-123", []string{"group1", "group2"}, true).Return(nil)
				m.On("SetServiceAccountAttributes", "test-realm", "client-123", expectedAttributes, true).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "error - SyncServiceAccountRoles fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(errors.New("roles sync failed"))
			},
			realmName:     "test-realm",
			expectedError: "unable to sync service account roles: roles sync failed",
		},
		{
			name: "error - SyncServiceAccountGroups fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
						Groups:     []string{"group1"},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
				m.On("SyncServiceAccountGroups", "test-realm", "client-123", []string{"group1"}, false).Return(errors.New("groups sync failed"))
			},
			realmName:     "test-realm",
			expectedError: "unable to sync service account groups: groups sync failed",
		},
		{
			name: "error - SetServiceAccountAttributes fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
						AttributesV2: map[string][]string{
							"attr1": {"value1"},
						},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedAttributes := map[string][]string{
					"attr1": {"value1"},
				}
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
				m.On("SetServiceAccountAttributes", "test-realm", "client-123", expectedAttributes, false).Return(errors.New("attributes set failed"))
			},
			realmName:     "test-realm",
			expectedError: "unable to set service account attributes: attributes set failed",
		},
		{
			name: "success - empty client roles",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:     true,
						RealmRoles:  []string{"realm-role1"},
						ClientRoles: []keycloakApi.UserClientRole{},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - empty realm roles",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{},
						ClientRoles: []keycloakApi.UserClientRole{
							{
								ClientID: "client1",
								Roles:    []string{"role1"},
							},
						},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				expectedClientRoles := map[string][]string{
					"client1": {"role1"},
				}
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{}, expectedClientRoles, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - default reconciliation strategy (full)",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					// ReconciliationStrategy not specified, should default to full
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.On("SyncServiceAccountRoles", "test-realm", "client-123", []string{"realm-role1"}, map[string][]string{}, false).Return(nil)
			},
			realmName:     "test-realm",
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := keycloakmocks.NewMockClient(t)
			tt.mockSetup(mockClient)

			// Create the service
			service := NewServiceAccount(mockClient)

			// Execute the method
			err := service.Serve(context.Background(), tt.keycloakClient, tt.realmName)

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
