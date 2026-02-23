package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	handlermocks "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler/mocks"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

const testRole1 = "role1"

func TestPutUsersRoles_ServeRequest(t *testing.T) {
	ctx := context.Background()
	uid := "user-uid-1"

	tests := []struct {
		name          string
		setupRealm    func() *keycloakApi.KeycloakRealm
		setupMocks    func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler)
		expectError   bool
		errorContains string
	}{
		{
			name: "success - no users",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users:     []keycloakApi.User{},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				nextHandler.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "success - users with no roles",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{}},
							{Username: "user2"},
						},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				nextHandler.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "success - role already exists for user",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1"}},
						},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				roleName := testRole1
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user1").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To(uid)}, nil, nil)
				mockUsers.EXPECT().GetUserRealmRoleMappings(mock.Anything, "test-realm", uid).
					Return([]keycloakv2.RoleRepresentation{{Name: &roleName}}, nil, nil)
				nextHandler.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "success - add role to user",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1"}},
						},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				roleName := testRole1
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user1").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To(uid)}, nil, nil)
				mockUsers.EXPECT().GetUserRealmRoleMappings(mock.Anything, "test-realm", uid).
					Return([]keycloakv2.RoleRepresentation{}, nil, nil)
				mockRoles.EXPECT().GetRealmRole(mock.Anything, "test-realm", "role1").
					Return(&keycloakv2.RoleRepresentation{Name: &roleName}, nil, nil)
				mockUsers.EXPECT().AddUserRealmRoles(mock.Anything, "test-realm", uid, mock.Anything).
					Return(nil, nil)
				nextHandler.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "error - FindUserByUsername fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1"}},
						},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user1").
					Return(nil, nil, errors.New("keycloak connection error"))
			},
			expectError:   true,
			errorContains: "error during putRolesToUsers",
		},
		{
			name: "error - user not found",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1"}},
						},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user1").
					Return(nil, nil, nil)
			},
			expectError:   true,
			errorContains: "user user1 not found in realm",
		},
		{
			name: "error - GetUserRealmRoleMappings fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1"}},
						},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user1").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To(uid)}, nil, nil)
				mockUsers.EXPECT().GetUserRealmRoleMappings(mock.Anything, "test-realm", uid).
					Return(nil, nil, errors.New("role mapping error"))
			},
			expectError:   true,
			errorContains: "unable to get user realm role mappings",
		},
		{
			name: "error - GetRealmRole fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1"}},
						},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user1").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To(uid)}, nil, nil)
				mockUsers.EXPECT().GetUserRealmRoleMappings(mock.Anything, "test-realm", uid).
					Return([]keycloakv2.RoleRepresentation{}, nil, nil)
				mockRoles.EXPECT().GetRealmRole(mock.Anything, "test-realm", "role1").
					Return(nil, nil, errors.New("role fetch failed"))
			},
			expectError:   true,
			errorContains: "unable to get realm role",
		},
		{
			name: "error - AddUserRealmRoles fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1"}},
						},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				roleName := testRole1
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user1").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To(uid)}, nil, nil)
				mockUsers.EXPECT().GetUserRealmRoleMappings(mock.Anything, "test-realm", uid).
					Return([]keycloakv2.RoleRepresentation{}, nil, nil)
				mockRoles.EXPECT().GetRealmRole(mock.Anything, "test-realm", "role1").
					Return(&keycloakv2.RoleRepresentation{Name: &roleName}, nil, nil)
				mockUsers.EXPECT().AddUserRealmRoles(mock.Anything, "test-realm", uid, mock.Anything).
					Return(nil, errors.New("failed to add role"))
			},
			expectError:   true,
			errorContains: "unable to add realm role to user",
		},
		{
			name: "error - next handler fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users:     []keycloakApi.User{},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				nextHandler.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("next handler error"))
			},
			expectError:   true,
			errorContains: "chain failed",
		},
		{
			name: "success - no next handler",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1"}},
						},
					},
				}
			},
			setupMocks: func(mockUsers *v2mocks.MockUsersClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				roleName := testRole1
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user1").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To(uid)}, nil, nil)
				mockUsers.EXPECT().GetUserRealmRoleMappings(mock.Anything, "test-realm", uid).
					Return([]keycloakv2.RoleRepresentation{}, nil, nil)
				mockRoles.EXPECT().GetRealmRole(mock.Anything, "test-realm", "role1").
					Return(&keycloakv2.RoleRepresentation{Name: &roleName}, nil, nil)
				mockUsers.EXPECT().AddUserRealmRoles(mock.Anything, "test-realm", uid, mock.Anything).
					Return(nil, nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsers := v2mocks.NewMockUsersClient(t)
			mockRoles := v2mocks.NewMockRolesClient(t)

			var nextHandler *handlermocks.MockRealmHandler

			var putUsersRoles PutUsersRoles

			if tt.name != "success - no next handler" {
				nextHandler = handlermocks.NewMockRealmHandler(t)
				putUsersRoles = PutUsersRoles{next: nextHandler}
			} else {
				putUsersRoles = PutUsersRoles{next: nil}
			}

			realm := tt.setupRealm()
			tt.setupMocks(mockUsers, mockRoles, nextHandler)

			kClientV2 := &keycloakv2.KeycloakClient{Users: mockUsers, Roles: mockRoles}
			err := putUsersRoles.ServeRequest(ctx, realm, kClientV2)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
