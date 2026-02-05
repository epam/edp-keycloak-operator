package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	handlermocks "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestPutUsersRoles_ServeRequest(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		setupRealm    func() *keycloakApi.KeycloakRealm
		setupMocks    func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler)
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
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				nextHandler.On("ServeRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
							{Username: "user2"}, // No roles field
						},
					},
				}
			},
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				nextHandler.On("ServeRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1"}}, "role1").
					Return(true, nil)
				nextHandler.On("ServeRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1"}}, "role1").
					Return(false, nil)
				kClient.On("AddRealmRoleToUser", ctx, "test-realm", "user1", "role1").
					Return(nil)
				nextHandler.On("ServeRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "success - multiple users with multiple roles",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1", "role2"}},
							{Username: "user2", RealmRoles: []string{"role3"}},
						},
					},
				}
			},
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				// User1 - role1: exists, role2: needs to be added
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1", "role2"}}, "role1").
					Return(true, nil)
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1", "role2"}}, "role2").
					Return(false, nil)
				kClient.On("AddRealmRoleToUser", ctx, "test-realm", "user1", "role2").
					Return(nil)

				// User2 - role3: needs to be added
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user2", RealmRoles: []string{"role3"}}, "role3").
					Return(false, nil)
				kClient.On("AddRealmRoleToUser", ctx, "test-realm", "user2", "role3").
					Return(nil)

				nextHandler.On("ServeRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "error - HasUserRealmRole fails",
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
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1"}}, "role1").
					Return(false, errors.New("keycloak connection error"))
			},
			expectError:   true,
			errorContains: "error during putRolesToUsers",
		},
		{
			name: "error - AddRealmRoleToUser fails",
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
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1"}}, "role1").
					Return(false, nil)
				kClient.On("AddRealmRoleToUser", ctx, "test-realm", "user1", "role1").
					Return(errors.New("failed to add role"))
			},
			expectError:   true,
			errorContains: "error during putRolesToUsers",
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
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				nextHandler.On("ServeRequest", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("next handler error"))
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
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1"}}, "role1").
					Return(false, nil)
				kClient.On("AddRealmRoleToUser", ctx, "test-realm", "user1", "role1").
					Return(nil)
				// nextHandler is nil for this test - no setup needed
			},
			expectError: false,
		},
		{
			name: "error - second user fails after first succeeds",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1"}},
							{Username: "user2", RealmRoles: []string{"role2"}},
						},
					},
				}
			},
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				// First user succeeds
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1"}}, "role1").
					Return(false, nil)
				kClient.On("AddRealmRoleToUser", ctx, "test-realm", "user1", "role1").
					Return(nil)

				// Second user fails
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user2", RealmRoles: []string{"role2"}}, "role2").
					Return(false, errors.New("user2 check failed"))
			},
			expectError:   true,
			errorContains: "error during putRolesToUsers",
		},
		{
			name: "error - second role fails after first succeeds for same user",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return &keycloakApi.KeycloakRealm{
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						Users: []keycloakApi.User{
							{Username: "user1", RealmRoles: []string{"role1", "role2"}},
						},
					},
				}
			},
			setupMocks: func(kClient *mocks.MockClient, nextHandler *handlermocks.MockRealmHandler) {
				// First role succeeds
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1", "role2"}}, "role1").
					Return(false, nil)
				kClient.On("AddRealmRoleToUser", ctx, "test-realm", "user1", "role1").
					Return(nil)

				// Second role fails
				kClient.On("HasUserRealmRole", "test-realm", &dto.User{Username: "user1", RealmRoles: []string{"role1", "role2"}}, "role2").
					Return(false, errors.New("role2 add failed"))
			},
			expectError:   true,
			errorContains: "error during putRolesToUsers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			kClient := mocks.NewMockClient(t)

			var nextHandler *handlermocks.MockRealmHandler
			if tt.name != "success - no next handler" {
				nextHandler = &handlermocks.MockRealmHandler{}
			}

			var putUsersRoles PutUsersRoles
			if nextHandler != nil {
				putUsersRoles = PutUsersRoles{next: nextHandler}
			} else {
				putUsersRoles = PutUsersRoles{next: nil}
			}

			realm := tt.setupRealm()
			tt.setupMocks(kClient, nextHandler)

			// Execute
			err := putUsersRoles.ServeRequest(ctx, realm, kClient, nil)

			// Assert
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
