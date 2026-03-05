package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestNewSyncUserRoles(t *testing.T) {
	h := NewSyncUserRoles(nil)
	require.NotNil(t, h)
}

func TestSyncUserRoles_Serve(t *testing.T) {
	tests := []struct {
		name      string
		user      *keycloakApi.KeycloakRealmUser
		userCtx   *UserContext
		mockSetup func(u *v2mocks.MockUsersClient, r *v2mocks.MockRolesClient, c *v2mocks.MockClientsClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "success - add missing realm roles (full strategy)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Roles: []string{"role1", "role2"},
				},
			},
			userCtx: &UserContext{UserID: "user-1"},
			mockSetup: func(u *v2mocks.MockUsersClient, r *v2mocks.MockRolesClient, c *v2mocks.MockClientsClient) {
				u.EXPECT().GetUserRealmRoleMappings(context.Background(), "test-realm", "user-1").
					Return(nil, nil, nil)
				r.EXPECT().GetRealmRole(context.Background(), "test-realm", "role1").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("id1"), Name: ptr.To("role1")}, nil, nil)
				r.EXPECT().GetRealmRole(context.Background(), "test-realm", "role2").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("id2"), Name: ptr.To("role2")}, nil, nil)
				u.EXPECT().AddUserRealmRoles(context.Background(), "test-realm", "user-1", []keycloakv2.RoleRepresentation{
					{Id: ptr.To("id1"), Name: ptr.To("role1")},
					{Id: ptr.To("id2"), Name: ptr.To("role2")},
				}).Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - remove extra realm roles (full strategy)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Roles: []string{"role1"},
				},
			},
			userCtx: &UserContext{UserID: "user-2"},
			mockSetup: func(u *v2mocks.MockUsersClient, r *v2mocks.MockRolesClient, c *v2mocks.MockClientsClient) {
				u.EXPECT().GetUserRealmRoleMappings(context.Background(), "test-realm", "user-2").
					Return([]keycloakv2.RoleRepresentation{
						{Id: ptr.To("id1"), Name: ptr.To("role1")},
						{Id: ptr.To("id2"), Name: ptr.To("role2")},
					}, nil, nil)
				// role1 already assigned, role2 is extra
				u.EXPECT().DeleteUserRealmRoles(context.Background(), "test-realm", "user-2", []keycloakv2.RoleRepresentation{
					{Id: ptr.To("id2"), Name: ptr.To("role2")},
				}).Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - add-only: no removal of extra roles",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Roles:                  []string{"role1"},
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
				},
			},
			userCtx: &UserContext{UserID: "user-3"},
			mockSetup: func(u *v2mocks.MockUsersClient, r *v2mocks.MockRolesClient, c *v2mocks.MockClientsClient) {
				u.EXPECT().GetUserRealmRoleMappings(context.Background(), "test-realm", "user-3").
					Return([]keycloakv2.RoleRepresentation{
						{Id: ptr.To("id2"), Name: ptr.To("role2")},
					}, nil, nil)
				r.EXPECT().GetRealmRole(context.Background(), "test-realm", "role1").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("id1"), Name: ptr.To("role1")}, nil, nil)
				u.EXPECT().AddUserRealmRoles(context.Background(), "test-realm", "user-3", []keycloakv2.RoleRepresentation{
					{Id: ptr.To("id1"), Name: ptr.To("role1")},
				}).Return(nil, nil)
				// DeleteUserRealmRoles must NOT be called
			},
			wantErr: require.NoError,
		},
		{
			name: "success - sync client roles",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					ClientRoles: []keycloakApi.UserClientRole{
						{ClientID: "client1", Roles: []string{"crole1"}},
					},
				},
			},
			userCtx: &UserContext{UserID: "user-4"},
			mockSetup: func(u *v2mocks.MockUsersClient, r *v2mocks.MockRolesClient, c *v2mocks.MockClientsClient) {
				// no realm roles
				u.EXPECT().GetUserRealmRoleMappings(context.Background(), "test-realm", "user-4").
					Return(nil, nil, nil)
				// client lookup
				c.EXPECT().GetClients(context.Background(), "test-realm", &keycloakv2.GetClientsParams{ClientId: ptr.To("client1")}).
					Return([]keycloakv2.ClientRepresentation{{Id: ptr.To("client-uuid-1")}}, nil, nil)
				// current client role mappings
				u.EXPECT().GetUserClientRoleMappings(context.Background(), "test-realm", "user-4", "client-uuid-1").
					Return(nil, nil, nil)
				// lookup missing role
				c.EXPECT().GetClientRole(context.Background(), "test-realm", "client-uuid-1", "crole1").
					Return(&keycloakv2.RoleRepresentation{Id: ptr.To("crid1"), Name: ptr.To("crole1")}, nil, nil)
				// add role
				u.EXPECT().AddUserClientRoles(context.Background(), "test-realm", "user-4", "client-uuid-1", []keycloakv2.RoleRepresentation{
					{Id: ptr.To("crid1"), Name: ptr.To("crole1")},
				}).Return(nil, nil)
				// remove extra — none
			},
			wantErr: require.NoError,
		},
		{
			name: "success - remove extra client roles (full strategy)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					ClientRoles: []keycloakApi.UserClientRole{
						{ClientID: "client1", Roles: []string{"crole1"}},
					},
				},
			},
			userCtx: &UserContext{UserID: "user-5"},
			mockSetup: func(u *v2mocks.MockUsersClient, r *v2mocks.MockRolesClient, c *v2mocks.MockClientsClient) {
				u.EXPECT().GetUserRealmRoleMappings(context.Background(), "test-realm", "user-5").
					Return(nil, nil, nil)
				c.EXPECT().GetClients(context.Background(), "test-realm", &keycloakv2.GetClientsParams{ClientId: ptr.To("client1")}).
					Return([]keycloakv2.ClientRepresentation{{Id: ptr.To("client-uuid-1")}}, nil, nil)
				u.EXPECT().GetUserClientRoleMappings(context.Background(), "test-realm", "user-5", "client-uuid-1").
					Return([]keycloakv2.RoleRepresentation{
						{Id: ptr.To("crid1"), Name: ptr.To("crole1")},
						{Id: ptr.To("crid2"), Name: ptr.To("crole2")},
					}, nil, nil)
				// crole2 is extra
				u.EXPECT().DeleteUserClientRoles(context.Background(), "test-realm", "user-5", "client-uuid-1", []keycloakv2.RoleRepresentation{
					{Id: ptr.To("crid2"), Name: ptr.To("crole2")},
				}).Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "error - GetUserRealmRoleMappings fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Roles: []string{"role1"},
				},
			},
			userCtx: &UserContext{UserID: "user-err"},
			mockSetup: func(u *v2mocks.MockUsersClient, r *v2mocks.MockRolesClient, c *v2mocks.MockClientsClient) {
				u.EXPECT().GetUserRealmRoleMappings(context.Background(), "test-realm", "user-err").
					Return(nil, nil, errors.New("api error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to sync user realm roles")
			},
		},
		{
			name: "error - GetRealmRole fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Roles: []string{"missing-role"},
				},
			},
			userCtx: &UserContext{UserID: "user-err2"},
			mockSetup: func(u *v2mocks.MockUsersClient, r *v2mocks.MockRolesClient, c *v2mocks.MockClientsClient) {
				u.EXPECT().GetUserRealmRoleMappings(context.Background(), "test-realm", "user-err2").
					Return(nil, nil, nil)
				r.EXPECT().GetRealmRole(context.Background(), "test-realm", "missing-role").
					Return(nil, nil, errors.New("not found"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to sync user realm roles")
			},
		},
		{
			name: "error - client not found",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "u", Namespace: "default"},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					ClientRoles: []keycloakApi.UserClientRole{
						{ClientID: "no-such-client", Roles: []string{"role1"}},
					},
				},
			},
			userCtx: &UserContext{UserID: "user-err3"},
			mockSetup: func(u *v2mocks.MockUsersClient, r *v2mocks.MockRolesClient, c *v2mocks.MockClientsClient) {
				u.EXPECT().GetUserRealmRoleMappings(context.Background(), "test-realm", "user-err3").
					Return(nil, nil, nil)
				c.EXPECT().GetClients(context.Background(), "test-realm", &keycloakv2.GetClientsParams{ClientId: ptr.To("no-such-client")}).
					Return(nil, nil, nil)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to sync user client roles")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsers := v2mocks.NewMockUsersClient(t)
			mockRoles := v2mocks.NewMockRolesClient(t)
			mockClients := v2mocks.NewMockClientsClient(t)
			tt.mockSetup(mockUsers, mockRoles, mockClients)

			h := NewSyncUserRoles(&keycloakv2.KeycloakClient{
				Users:   mockUsers,
				Roles:   mockRoles,
				Clients: mockClients,
			})

			err := h.Serve(
				context.Background(),
				tt.user,
				"test-realm",
				tt.userCtx,
			)

			tt.wantErr(t, err)
		})
	}
}
