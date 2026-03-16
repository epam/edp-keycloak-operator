package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	keycloakv2Mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestServiceAccount_Serve(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		kClient        func(t *testing.T) *keycloakv2.KeycloakClient
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				return &keycloakv2.KeycloakClient{}
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				return &keycloakv2.KeycloakClient{}
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				return &keycloakv2.KeycloakClient{}
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
					ClientId:               "test-client-id",
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)

				// GetServiceAccountUser
				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				// syncRealmRoles: get current mappings (empty), get each role, add
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role1"), Id: ptr.To("rr1-id")}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role2").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role2"), Id: ptr.To("rr2-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserRealmRoles", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				// syncClientRoles for client1
				clientsMock.On("GetClientByClientID", mock.Anything, "test-realm", "client1").
					Return(&keycloakv2.ClientRepresentation{Id: ptr.To("client1-uuid")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("GetUserClientRoleMappings", mock.Anything, "test-realm", "sa-user-id", "client1-uuid").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("GetClientRole", mock.Anything, "test-realm", "client1-uuid", "role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("role1"), Id: ptr.To("cr1-id")}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("GetClientRole", mock.Anything, "test-realm", "client1-uuid", "role2").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("role2"), Id: ptr.To("cr2-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserClientRoles", mock.Anything, "test-realm", "sa-user-id", "client1-uuid", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				// syncClientRoles for client2
				clientsMock.On("GetClientByClientID", mock.Anything, "test-realm", "client2").
					Return(&keycloakv2.ClientRepresentation{Id: ptr.To("client2-uuid")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("GetUserClientRoleMappings", mock.Anything, "test-realm", "sa-user-id", "client2-uuid").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("GetClientRole", mock.Anything, "test-realm", "client2-uuid", "role3").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("role3"), Id: ptr.To("cr3-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserClientRoles", mock.Anything, "test-realm", "sa-user-id", "client2-uuid", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
					Roles:   rolesMock,
				}
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
					ClientId:               "test-client-id",
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				// syncRealmRoles: role1 already exists
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{
						{Name: ptr.To("realm-role1"), Id: ptr.To("rr1-id")},
					}, (*keycloakv2.Response)(nil), nil)
				// No AddUserRealmRoles or DeleteUserRealmRoles expected (addOnly=true, role already exists)

				// syncClientRoles
				clientsMock.On("GetClientByClientID", mock.Anything, "test-realm", "client1").
					Return(&keycloakv2.ClientRepresentation{Id: ptr.To("client1-uuid")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("GetUserClientRoleMappings", mock.Anything, "test-realm", "sa-user-id", "client1-uuid").
					Return([]keycloakv2.RoleRepresentation{
						{Name: ptr.To("role1"), Id: ptr.To("cr1-id")},
					}, (*keycloakv2.Response)(nil), nil)
				// No AddUserClientRoles expected (role already exists)

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
					Roles:   rolesMock,
				}
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
					ClientId: "test-client-id",
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)
				groupsMock := keycloakv2Mocks.NewMockGroupsClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				// syncRealmRoles
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role1"), Id: ptr.To("rr1-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserRealmRoles", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				// syncGroups
				usersMock.On("GetUserGroups", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.GroupRepresentation{}, (*keycloakv2.Response)(nil), nil)
				groupsMock.On("GetGroups", mock.Anything, "test-realm", (*keycloakv2.GetGroupsParams)(nil)).
					Return([]keycloakv2.GroupRepresentation{
						{Name: ptr.To("group1"), Id: ptr.To("g1-id")},
						{Name: ptr.To("group2"), Id: ptr.To("g2-id")},
					}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserToGroup", mock.Anything, "test-realm", "sa-user-id", "g1-id").
					Return((*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserToGroup", mock.Anything, "test-realm", "sa-user-id", "g2-id").
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
					Roles:   rolesMock,
					Groups:  groupsMock,
				}
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
					ClientId: "test-client-id",
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				// syncRealmRoles
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role1"), Id: ptr.To("rr1-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserRealmRoles", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				// setAttributes
				usersMock.On("UpdateUser", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
					Roles:   rolesMock,
				}
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
					ClientId:               "test-client-id",
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)
				groupsMock := keycloakv2Mocks.NewMockGroupsClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				// syncRealmRoles - addOnly, roles don't exist yet
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role1"), Id: ptr.To("rr1-id")}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role2").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role2"), Id: ptr.To("rr2-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserRealmRoles", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				// syncClientRoles
				clientsMock.On("GetClientByClientID", mock.Anything, "test-realm", "client1").
					Return(&keycloakv2.ClientRepresentation{Id: ptr.To("client1-uuid")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("GetUserClientRoleMappings", mock.Anything, "test-realm", "sa-user-id", "client1-uuid").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("GetClientRole", mock.Anything, "test-realm", "client1-uuid", "role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("role1"), Id: ptr.To("cr1-id")}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("GetClientRole", mock.Anything, "test-realm", "client1-uuid", "role2").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("role2"), Id: ptr.To("cr2-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserClientRoles", mock.Anything, "test-realm", "sa-user-id", "client1-uuid", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				// syncGroups - addOnly
				usersMock.On("GetUserGroups", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.GroupRepresentation{}, (*keycloakv2.Response)(nil), nil)
				groupsMock.On("GetGroups", mock.Anything, "test-realm", (*keycloakv2.GetGroupsParams)(nil)).
					Return([]keycloakv2.GroupRepresentation{
						{Name: ptr.To("group1"), Id: ptr.To("g1-id")},
						{Name: ptr.To("group2"), Id: ptr.To("g2-id")},
					}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserToGroup", mock.Anything, "test-realm", "sa-user-id", "g1-id").
					Return((*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserToGroup", mock.Anything, "test-realm", "sa-user-id", "g2-id").
					Return((*keycloakv2.Response)(nil), nil)

				// setAttributes
				usersMock.On("UpdateUser", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
					Roles:   rolesMock,
					Groups:  groupsMock,
				}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "error - GetServiceAccountUser fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return((*keycloakv2.UserRepresentation)(nil), (*keycloakv2.Response)(nil), errors.New("sa user lookup failed"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock}
			},
			realmName:     "test-realm",
			expectedError: "unable to get service account user",
		},
		{
			name: "error - syncRealmRoles fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return(nil, (*keycloakv2.Response)(nil), errors.New("roles sync failed"))

				return &keycloakv2.KeycloakClient{Clients: clientsMock, Users: usersMock}
			},
			realmName:     "test-realm",
			expectedError: "unable to sync service account realm roles",
		},
		{
			name: "error - syncGroups fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				// syncRealmRoles succeeds
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role1"), Id: ptr.To("rr1-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserRealmRoles", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				// syncGroups fails
				usersMock.On("GetUserGroups", mock.Anything, "test-realm", "sa-user-id").
					Return(nil, (*keycloakv2.Response)(nil), errors.New("groups sync failed"))

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
					Roles:   rolesMock,
				}
			},
			realmName:     "test-realm",
			expectedError: "unable to sync service account groups",
		},
		{
			name: "error - setAttributes fails",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				// syncRealmRoles succeeds
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role1"), Id: ptr.To("rr1-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserRealmRoles", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				// setAttributes: UpdateUser fails
				usersMock.On("UpdateUser", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), errors.New("attributes set failed"))

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
					Roles:   rolesMock,
				}
			},
			realmName:     "test-realm",
			expectedError: "unable to set service account attributes",
		},
		{
			name: "success - empty client roles",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client-id",
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role1"), Id: ptr.To("rr1-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserRealmRoles", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
					Roles:   rolesMock,
				}
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
					ClientId: "test-client-id",
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
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				// syncRealmRoles: empty desired, empty current => no-op
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)

				// syncClientRoles for client1
				clientsMock.On("GetClientByClientID", mock.Anything, "test-realm", "client1").
					Return(&keycloakv2.ClientRepresentation{Id: ptr.To("client1-uuid")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("GetUserClientRoleMappings", mock.Anything, "test-realm", "sa-user-id", "client1-uuid").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				clientsMock.On("GetClientRole", mock.Anything, "test-realm", "client1-uuid", "role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("role1"), Id: ptr.To("cr1-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserClientRoles", mock.Anything, "test-realm", "sa-user-id", "client1-uuid", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
				}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
		{
			name: "success - addOnly preserves existing attributes",
			keycloakClient: &keycloakApi.KeycloakClient{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-client",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId:               "test-client-id",
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled: true,
						AttributesV2: map[string][]string{
							"new-attr": {"new-value"},
						},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)

				existingAttrs := map[string][]string{"existing-attr": {"existing-value"}}
				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{
						Id:         ptr.To("sa-user-id"),
						Attributes: &existingAttrs,
					}, (*keycloakv2.Response)(nil), nil)

				// syncRealmRoles: empty desired, empty current => no-op
				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)

				// setAttributes: must contain both existing and new attributes
				usersMock.On("UpdateUser", mock.Anything, "test-realm", "sa-user-id",
					mock.MatchedBy(func(u keycloakv2.UserRepresentation) bool {
						if u.Attributes == nil {
							return false
						}

						attrs := *u.Attributes

						_, hasExisting := attrs["existing-attr"]
						_, hasNew := attrs["new-attr"]

						return hasExisting && hasNew
					})).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
				}
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
					ClientId: "test-client-id",
					ServiceAccount: &keycloakApi.ServiceAccount{
						Enabled:    true,
						RealmRoles: []string{"realm-role1"},
					},
				},
				Status: keycloakApi.KeycloakClientStatus{
					ClientID: "client-123",
				},
			},
			kClient: func(t *testing.T) *keycloakv2.KeycloakClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)

				clientsMock.On("GetServiceAccountUser", mock.Anything, "test-realm", "client-uuid").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To("sa-user-id")}, (*keycloakv2.Response)(nil), nil)

				usersMock.On("GetUserRealmRoleMappings", mock.Anything, "test-realm", "sa-user-id").
					Return([]keycloakv2.RoleRepresentation{}, (*keycloakv2.Response)(nil), nil)
				rolesMock.On("GetRealmRole", mock.Anything, "test-realm", "realm-role1").
					Return(&keycloakv2.RoleRepresentation{Name: ptr.To("realm-role1"), Id: ptr.To("rr1-id")}, (*keycloakv2.Response)(nil), nil)
				usersMock.On("AddUserRealmRoles", mock.Anything, "test-realm", "sa-user-id", mock.Anything).
					Return((*keycloakv2.Response)(nil), nil)

				return &keycloakv2.KeycloakClient{
					Clients: clientsMock,
					Users:   usersMock,
					Roles:   rolesMock,
				}
			},
			realmName:     "test-realm",
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kClient := tt.kClient(t)

			if tt.keycloakClient.Name == "" {
				tt.keycloakClient.Name = "test-client"
			}

			if tt.keycloakClient.Namespace == "" {
				tt.keycloakClient.Namespace = "default"
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(tt.keycloakClient).
				WithStatusSubresource(tt.keycloakClient).
				Build()

			service := NewServiceAccount(kClient, k8sClient)

			err := service.Serve(context.Background(), tt.keycloakClient, tt.realmName, &ClientContext{ClientUUID: "client-uuid"})

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
