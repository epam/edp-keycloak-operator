package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestProcessPolicy_Serve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		keycloakClient    *keycloakApi.KeycloakClient
		keycloakApiClient func(t *testing.T) *mocks.MockClient
		wantErr           require.ErrorAssertionFunc
	}{
		{
			name: "client authorization is not set",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientId: "test-client",
				},
			},
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				return mocks.NewMockClient(t)
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{
						"Default Policy": {
							ID: gocloak.StringP("default-policy-id"),
						},
						"user-policy": {
							ID: gocloak.StringP("user-policy-id"),
						},
						"role-policy": {
							ID: gocloak.StringP("role-policy-id"),
						},
						"user-policy2": {
							ID: gocloak.StringP("user-policy2-id"),
						},
					}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("GetGroups", mock.Anything, "master").
					Return(map[string]*gocloak.Group{
						"test-group": {
							ID: gocloak.StringP("test-group-id"),
						},
					}, nil)
				m.On("GetRealmRoles", mock.Anything, "master").
					Return(map[string]gocloak.Role{
						"test-role": {
							ID: gocloak.StringP("test-role-id"),
						},
					}, nil)
				m.On("GetUsersByNames", mock.Anything, "master", []string{"test-user"}).
					Return(map[string]gocloak.User{
						"test-user": {
							ID: gocloak.StringP("test-user-id"),
						},
					}, nil)
				m.On("CreatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(&gocloak.PolicyRepresentation{}, nil).Times(4)
				m.On("UpdatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(nil).Times(2)
				m.On("DeletePolicy", mock.Anything, "master", "test-client-id", "user-policy2-id").
					Return(nil).Once()

				return m
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client-2", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{
						"Default Policy": {
							ID: gocloak.StringP("default-policy-id"),
						},
						"user-policy": {
							ID: gocloak.StringP("user-policy-id"),
						},
						"role-policy": {
							ID: gocloak.StringP("role-policy-id"),
						},
						"user-policy2": {
							ID: gocloak.StringP("user-policy2-id"),
						},
						"existing-policy": {
							ID: gocloak.StringP("existing-policy-id"),
						},
					}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client-2": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("GetGroups", mock.Anything, "master").
					Return(map[string]*gocloak.Group{
						"test-group": {
							ID: gocloak.StringP("test-group-id"),
						},
					}, nil)
				m.On("GetRealmRoles", mock.Anything, "master").
					Return(map[string]gocloak.Role{
						"test-role": {
							ID: gocloak.StringP("test-role-id"),
						},
					}, nil)
				m.On("GetUsersByNames", mock.Anything, "master", []string{"test-user"}).
					Return(map[string]gocloak.User{
						"test-user": {
							ID: gocloak.StringP("test-user-id"),
						},
					}, nil)
				m.On("CreatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(&gocloak.PolicyRepresentation{}, nil).Times(4)
				m.On("UpdatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(nil).Times(2)

				return m
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{
						"user-policy": {
							ID: gocloak.StringP("user-policy-id"),
						},
						"client-policy": {
							ID: gocloak.StringP("client-policy-id"),
						},
					}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("UpdatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(nil).Times(1)
				m.On("DeletePolicy", mock.Anything, "master", "test-client-id", "user-policy-id").
					Return(errors.New("failed to delete policy")).Once()

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{
						"client-policy": {
							ID: gocloak.StringP("client-policy-id"),
						},
					}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("UpdatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(errors.New("failed to update policy")).Times(1)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{}, nil)
				m.On("GetClients", mock.Anything, "master").
					Return(map[string]*gocloak.Client{
						"test-client": {
							ID: gocloak.StringP("test-client-id"),
						},
					}, nil)
				m.On("CreatePolicy", mock.Anything, "master", mock.Anything, mock.Anything).
					Return(nil, errors.New("failed to crate policy")).Times(1)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(map[string]*gocloak.PolicyRepresentation{}, nil)

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("test-client-id", nil)
				m.On("GetPolicies", mock.Anything, "master", "test-client-id").
					Return(nil, errors.New("failed to get policies"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get policies")
			},
		},
		{
			name: "failed to get client id",
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
			keycloakApiClient: func(t *testing.T) *mocks.MockClient {
				m := mocks.NewMockClient(t)

				m.On("GetClientID", "test-client", "master").
					Return("", errors.New("failed to get client id"))

				return m
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get client id")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewProcessPolicy(tt.keycloakApiClient(t))

			err := h.Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.keycloakClient, "master")
			tt.wantErr(t, err)
		})
	}
}
