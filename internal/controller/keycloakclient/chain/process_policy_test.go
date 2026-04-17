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
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakv2Mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestProcessPolicy_Serve(t *testing.T) {
	const (
		testClientName      = "test-client"
		testClientNamespace = "default"
	)

	s := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(s))
	require.NoError(t, corev1.AddToScheme(s))

	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		kClient        func(t *testing.T) *keycloakapi.APIClient
		wantErr        require.ErrorAssertionFunc
	}{
		{
			name: "client authorization is not set",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client",
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				return &keycloakapi.APIClient{}
			},
			wantErr: require.NoError,
		},
		{
			name: "policies processed successfully",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client",
					Authorization: &keycloakApi.Authorization{
						Policies: []keycloakApi.Policy{
							{
								Name:        "aggregate-policy",
								Type:        keycloakApi.PolicyTypeAggregate,
								Description: "Aggregate policy",
								AggregatedPolicy: &keycloakApi.AggregatedPolicyData{
									Policies: []string{"role-policy", "user-policy"},
								},
							},
							{
								Name:        "client-policy",
								Description: "Client policy",
								Type:        keycloakApi.PolicyTypeClient,
								ClientPolicy: &keycloakApi.ClientPolicyData{
									Clients: []string{"test-client"},
								},
							},
							{
								Name:        "group-policy",
								Description: "Group policy",
								Type:        keycloakApi.PolicyTypeGroup,
								GroupPolicy: &keycloakApi.GroupPolicyData{
									Groups: []keycloakApi.GroupDefinition{
										{
											Name: "test-group",
										},
									},
								},
							},
							{
								Name:        "role-policy",
								Description: "Role policy",
								Type:        keycloakApi.PolicyTypeRole,
								RolePolicy: &keycloakApi.RolePolicyData{
									Roles: []keycloakApi.RoleDefinition{
										{
											Name:     "test-role",
											Required: true,
										},
									},
								},
							},
							{
								Name:        "time-policy",
								Description: "Time policy",
								Type:        keycloakApi.PolicyTypeTime,
								TimePolicy: &keycloakApi.TimePolicyData{
									NotBefore:    "2024-03-03 00:00:00",
									NotOnOrAfter: "2024-03-03 00:00:00",
								},
							},
							{
								Name:        "user-policy",
								Description: "User policy",
								Type:        keycloakApi.PolicyTypeUser,
								UserPolicy: &keycloakApi.UserPolicyData{
									Users: []string{"test-user"},
								},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)
				groupsMock := keycloakv2Mocks.NewMockGroupsClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)

				authzMock.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{Id: ptr.To("default-policy-id"), Name: ptr.To("Default Policy")},
						{Id: ptr.To("user-policy-id"), Name: ptr.To("user-policy")},
						{Id: ptr.To("role-policy-id"), Name: ptr.To("role-policy")},
						{Id: ptr.To("user-policy2-id"), Name: ptr.To("user-policy2")},
					}, (*keycloakapi.Response)(nil), nil)

				clientsMock.On("GetClients", mock.Anything, "master", (*keycloakapi.GetClientsParams)(nil)).
					Return([]keycloakapi.ClientRepresentation{
						{Id: ptr.To("test-client-id"), ClientId: ptr.To("test-client")},
					}, (*keycloakapi.Response)(nil), nil)

				groupsMock.On("GetGroups", mock.Anything, "master", (*keycloakapi.GetGroupsParams)(nil)).
					Return([]keycloakapi.GroupRepresentation{
						{Id: ptr.To("test-group-id"), Name: ptr.To("test-group")},
					}, (*keycloakapi.Response)(nil), nil)

				rolesMock.On("GetRealmRoles", mock.Anything, "master", (*keycloakapi.GetRealmRolesParams)(nil)).
					Return([]keycloakapi.RoleRepresentation{
						{Id: ptr.To("test-role-id"), Name: ptr.To("test-role")},
					}, (*keycloakapi.Response)(nil), nil)

				usersMock.On("FindUserByUsername", mock.Anything, "master", "test-user").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("test-user-id")}, (*keycloakapi.Response)(nil), nil)

				authzMock.On("CreatePolicy", mock.Anything, "master", "test-client-id", mock.Anything, mock.Anything).
					Return((*keycloakapi.PolicyRepresentation)(nil), (*keycloakapi.Response)(nil), nil).Times(4)

				authzMock.On("UpdatePolicy", mock.Anything, "master", "test-client-id", mock.Anything, mock.Anything, mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Times(2)

				authzMock.On("DeletePolicy", mock.Anything, "master", "test-client-id", "user-policy2-id").
					Return((*keycloakapi.Response)(nil), nil).Once()

				return &keycloakapi.APIClient{
					Clients:       clientsMock,
					Authorization: authzMock,
					Groups:        groupsMock,
					Roles:         rolesMock,
					Users:         usersMock,
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "policies addOnly successful",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
					ClientId:               "test-client-2",
					Authorization: &keycloakApi.Authorization{
						Policies: []keycloakApi.Policy{
							{
								Name:        "aggregate-policy",
								Type:        keycloakApi.PolicyTypeAggregate,
								Description: "Aggregate policy",
								AggregatedPolicy: &keycloakApi.AggregatedPolicyData{
									Policies: []string{"role-policy", "user-policy"},
								},
							},
							{
								Name:        "client-policy",
								Description: "Client policy",
								Type:        keycloakApi.PolicyTypeClient,
								ClientPolicy: &keycloakApi.ClientPolicyData{
									Clients: []string{"test-client-2"},
								},
							},
							{
								Name:        "group-policy",
								Description: "Group policy",
								Type:        keycloakApi.PolicyTypeGroup,
								GroupPolicy: &keycloakApi.GroupPolicyData{
									Groups: []keycloakApi.GroupDefinition{
										{
											Name: "test-group",
										},
									},
								},
							},
							{
								Name:        "role-policy",
								Description: "Role policy",
								Type:        keycloakApi.PolicyTypeRole,
								RolePolicy: &keycloakApi.RolePolicyData{
									Roles: []keycloakApi.RoleDefinition{
										{
											Name:     "test-role",
											Required: true,
										},
									},
								},
							},
							{
								Name:        "time-policy",
								Description: "Time policy",
								Type:        keycloakApi.PolicyTypeTime,
								TimePolicy: &keycloakApi.TimePolicyData{
									NotBefore:    "2024-03-03 00:00:00",
									NotOnOrAfter: "2024-03-03 00:00:00",
								},
							},
							{
								Name:        "user-policy",
								Description: "User policy",
								Type:        keycloakApi.PolicyTypeUser,
								UserPolicy: &keycloakApi.UserPolicyData{
									Users: []string{"test-user"},
								},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)
				groupsMock := keycloakv2Mocks.NewMockGroupsClient(t)
				rolesMock := keycloakv2Mocks.NewMockRolesClient(t)
				usersMock := keycloakv2Mocks.NewMockUsersClient(t)

				authzMock.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{Id: ptr.To("default-policy-id"), Name: ptr.To("Default Policy")},
						{Id: ptr.To("user-policy-id"), Name: ptr.To("user-policy")},
						{Id: ptr.To("role-policy-id"), Name: ptr.To("role-policy")},
						{Id: ptr.To("user-policy2-id"), Name: ptr.To("user-policy2")},
						{Id: ptr.To("existing-policy-id"), Name: ptr.To("existing-policy")},
					}, (*keycloakapi.Response)(nil), nil)

				clientsMock.On("GetClients", mock.Anything, "master", (*keycloakapi.GetClientsParams)(nil)).
					Return([]keycloakapi.ClientRepresentation{
						{Id: ptr.To("test-client-id"), ClientId: ptr.To("test-client-2")},
					}, (*keycloakapi.Response)(nil), nil)

				groupsMock.On("GetGroups", mock.Anything, "master", (*keycloakapi.GetGroupsParams)(nil)).
					Return([]keycloakapi.GroupRepresentation{
						{Id: ptr.To("test-group-id"), Name: ptr.To("test-group")},
					}, (*keycloakapi.Response)(nil), nil)

				rolesMock.On("GetRealmRoles", mock.Anything, "master", (*keycloakapi.GetRealmRolesParams)(nil)).
					Return([]keycloakapi.RoleRepresentation{
						{Id: ptr.To("test-role-id"), Name: ptr.To("test-role")},
					}, (*keycloakapi.Response)(nil), nil)

				usersMock.On("FindUserByUsername", mock.Anything, "master", "test-user").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("test-user-id")}, (*keycloakapi.Response)(nil), nil)

				authzMock.On("CreatePolicy", mock.Anything, "master", "test-client-id", mock.Anything, mock.Anything).
					Return((*keycloakapi.PolicyRepresentation)(nil), (*keycloakapi.Response)(nil), nil).Times(4)

				authzMock.On("UpdatePolicy", mock.Anything, "master", "test-client-id", mock.Anything, mock.Anything, mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Times(2)

				return &keycloakapi.APIClient{
					Clients:       clientsMock,
					Authorization: authzMock,
					Groups:        groupsMock,
					Roles:         rolesMock,
					Users:         usersMock,
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "aggregate policy references policy created in same reconcile",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client",
					Authorization: &keycloakApi.Authorization{
						Policies: []keycloakApi.Policy{
							{
								Name: "client-policy",
								Type: keycloakApi.PolicyTypeClient,
								ClientPolicy: &keycloakApi.ClientPolicyData{
									Clients: []string{"test-client"},
								},
							},
							{
								Name: "aggregate-policy",
								Type: keycloakApi.PolicyTypeAggregate,
								AggregatedPolicy: &keycloakApi.AggregatedPolicyData{
									Policies: []string{"client-policy"},
								},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return([]keycloakapi.AbstractPolicyRepresentation{}, (*keycloakapi.Response)(nil), nil)

				clientsMock.On("GetClients", mock.Anything, "master", (*keycloakapi.GetClientsParams)(nil)).
					Return([]keycloakapi.ClientRepresentation{
						{Id: ptr.To("test-client-id"), ClientId: ptr.To("test-client")},
					}, (*keycloakapi.Response)(nil), nil)

				// CreatePolicy for client-policy returns the policy with an ID so aggregate-policy can reference it.
				authzMock.On("CreatePolicy", mock.Anything, "master", "test-client-id", keycloakApi.PolicyTypeClient, mock.Anything).
					Return(&keycloakapi.PolicyRepresentation{Id: ptr.To("client-policy-id"), Name: ptr.To("client-policy")}, (*keycloakapi.Response)(nil), nil).Once()

				authzMock.On("CreatePolicy", mock.Anything, "master", "test-client-id", keycloakApi.PolicyTypeAggregate, mock.Anything).
					Return((*keycloakapi.PolicyRepresentation)(nil), (*keycloakapi.Response)(nil), nil).Once()

				return &keycloakapi.APIClient{
					Clients:       clientsMock,
					Authorization: authzMock,
				}
			},
			wantErr: require.NoError,
		},
		{
			name: "failed to delete policy",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client",
					Authorization: &keycloakApi.Authorization{
						Policies: []keycloakApi.Policy{
							{
								Name:        "client-policy",
								Description: "Client policy",
								Type:        keycloakApi.PolicyTypeClient,
								ClientPolicy: &keycloakApi.ClientPolicyData{
									Clients: []string{"test-client"},
								},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{Id: ptr.To("user-policy-id"), Name: ptr.To("user-policy")},
						{Id: ptr.To("client-policy-id"), Name: ptr.To("client-policy")},
					}, (*keycloakapi.Response)(nil), nil)

				clientsMock.On("GetClients", mock.Anything, "master", (*keycloakapi.GetClientsParams)(nil)).
					Return([]keycloakapi.ClientRepresentation{
						{Id: ptr.To("test-client-id"), ClientId: ptr.To("test-client")},
					}, (*keycloakapi.Response)(nil), nil)

				authzMock.On("UpdatePolicy", mock.Anything, "master", "test-client-id", mock.Anything, mock.Anything, mock.Anything).
					Return((*keycloakapi.Response)(nil), nil).Times(1)

				authzMock.On("DeletePolicy", mock.Anything, "master", "test-client-id", "user-policy-id").
					Return((*keycloakapi.Response)(nil), errors.New("failed to delete policy")).Once()

				return &keycloakapi.APIClient{
					Clients:       clientsMock,
					Authorization: authzMock,
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to delete policy")
			},
		},
		{
			name: "failed to update policy",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client",
					Authorization: &keycloakApi.Authorization{
						Policies: []keycloakApi.Policy{
							{
								Name:        "client-policy",
								Description: "Client policy",
								Type:        keycloakApi.PolicyTypeClient,
								ClientPolicy: &keycloakApi.ClientPolicyData{
									Clients: []string{"test-client"},
								},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return([]keycloakapi.AbstractPolicyRepresentation{
						{Id: ptr.To("client-policy-id"), Name: ptr.To("client-policy")},
					}, (*keycloakapi.Response)(nil), nil)

				clientsMock.On("GetClients", mock.Anything, "master", (*keycloakapi.GetClientsParams)(nil)).
					Return([]keycloakapi.ClientRepresentation{
						{Id: ptr.To("test-client-id"), ClientId: ptr.To("test-client")},
					}, (*keycloakapi.Response)(nil), nil)

				authzMock.On("UpdatePolicy", mock.Anything, "master", "test-client-id", mock.Anything, mock.Anything, mock.Anything).
					Return((*keycloakapi.Response)(nil), errors.New("failed to update policy")).Times(1)

				return &keycloakapi.APIClient{
					Clients:       clientsMock,
					Authorization: authzMock,
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to update policy")
			},
		},
		{
			name: "failed to create policy",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client",
					Authorization: &keycloakApi.Authorization{
						Policies: []keycloakApi.Policy{
							{
								Name:        "client-policy",
								Description: "Client policy",
								Type:        keycloakApi.PolicyTypeClient,
								ClientPolicy: &keycloakApi.ClientPolicyData{
									Clients: []string{"test-client"},
								},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return([]keycloakapi.AbstractPolicyRepresentation{}, (*keycloakapi.Response)(nil), nil)

				clientsMock.On("GetClients", mock.Anything, "master", (*keycloakapi.GetClientsParams)(nil)).
					Return([]keycloakapi.ClientRepresentation{
						{Id: ptr.To("test-client-id"), ClientId: ptr.To("test-client")},
					}, (*keycloakapi.Response)(nil), nil)

				authzMock.On("CreatePolicy", mock.Anything, "master", "test-client-id", mock.Anything, mock.Anything).
					Return((*keycloakapi.PolicyRepresentation)(nil), (*keycloakapi.Response)(nil), errors.New("failed to crate policy")).Times(1)

				return &keycloakapi.APIClient{
					Clients:       clientsMock,
					Authorization: authzMock,
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to create policy")
			},
		},
		{
			name: "invalid policy type",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client",
					Authorization: &keycloakApi.Authorization{
						Policies: []keycloakApi.Policy{
							{
								Name:        "client-policy",
								Description: "Client policy",
								Type:        "invalid",
								ClientPolicy: &keycloakApi.ClientPolicyData{
									Clients: []string{"test-client"},
								},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return([]keycloakapi.AbstractPolicyRepresentation{}, (*keycloakapi.Response)(nil), nil)

				return &keycloakapi.APIClient{
					Clients:       clientsMock,
					Authorization: authzMock,
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to convert policy")
			},
		},
		{
			name: "failed to get policies",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client",
					Authorization: &keycloakApi.Authorization{
						Policies: []keycloakApi.Policy{
							{
								Name:        "client-policy",
								Description: "Client policy",
								Type:        "invalid",
								ClientPolicy: &keycloakApi.ClientPolicyData{
									Clients: []string{"test-client"},
								},
							},
						},
					},
				},
			},
			kClient: func(t *testing.T) *keycloakapi.APIClient {
				clientsMock := keycloakv2Mocks.NewMockClientsClient(t)
				authzMock := keycloakv2Mocks.NewMockAuthorizationClient(t)

				authzMock.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(nil, (*keycloakapi.Response)(nil), errors.New("failed to get policies"))

				return &keycloakapi.APIClient{
					Clients:       clientsMock,
					Authorization: authzMock,
				}
			},
			wantErr: func(t require.TestingT, err error, i ...any) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get policies")
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

			h := NewProcessPolicy(tt.kClient(t), k8sClient)

			err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloakClient, "master", &ClientContext{ClientUUID: "test-client-id"})
			tt.wantErr(t, err)
		})
	}
}
