package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	handlermocks "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	keycloakmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestPutUsers_ServeRequest(t *testing.T) {
	tests := []struct {
		name          string
		realm         *keycloakApi.KeycloakRealm
		mockSetup     func(*keycloakmocks.MockClient, *handlermocks.MockRealmHandler)
		expectedError string
	}{
		{
			name: "success - create single user",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{
						{
							Username:   "testuser1",
							RealmRoles: []string{"role1", "role2"},
						},
					},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				expectedUser := &dto.User{
					Username:   "testuser1",
					RealmRoles: []string{"role1", "role2"},
				}
				mockClient.On("ExistRealmUser", "test-realm", expectedUser).Return(false, nil)
				mockClient.On("CreateRealmUser", "test-realm", expectedUser).Return(nil)
				mockNext.On("ServeRequest", mock.Anything, &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						KeycloakRef: common.KeycloakRef{
							Name: "test-keycloak",
						},
						Users: []keycloakApi.User{
							{
								Username:   "testuser1",
								RealmRoles: []string{"role1", "role2"},
							},
						},
					},
				}, mockClient).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "success - create multiple users",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{
						{
							Username:   "user1",
							RealmRoles: []string{"role1"},
						},
						{
							Username:   "user2",
							RealmRoles: []string{"role2"},
						},
					},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				expectedUser1 := &dto.User{
					Username:   "user1",
					RealmRoles: []string{"role1"},
				}
				expectedUser2 := &dto.User{
					Username:   "user2",
					RealmRoles: []string{"role2"},
				}
				mockClient.On("ExistRealmUser", "test-realm", expectedUser1).Return(false, nil)
				mockClient.On("CreateRealmUser", "test-realm", expectedUser1).Return(nil)
				mockClient.On("ExistRealmUser", "test-realm", expectedUser2).Return(false, nil)
				mockClient.On("CreateRealmUser", "test-realm", expectedUser2).Return(nil)
				mockNext.On("ServeRequest", mock.Anything, &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						KeycloakRef: common.KeycloakRef{
							Name: "test-keycloak",
						},
						Users: []keycloakApi.User{
							{
								Username:   "user1",
								RealmRoles: []string{"role1"},
							},
							{
								Username:   "user2",
								RealmRoles: []string{"role2"},
							},
						},
					},
				}, mockClient).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "success - user without realm roles",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{
						{
							Username: "simpleuser",
						},
					},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				expectedUser := &dto.User{
					Username:   "simpleuser",
					RealmRoles: nil,
				}
				mockClient.On("ExistRealmUser", "test-realm", expectedUser).Return(false, nil)
				mockClient.On("CreateRealmUser", "test-realm", expectedUser).Return(nil)
				mockNext.On("ServeRequest", mock.Anything, &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						KeycloakRef: common.KeycloakRef{
							Name: "test-keycloak",
						},
						Users: []keycloakApi.User{
							{
								Username: "simpleuser",
							},
						},
					},
				}, mockClient).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "success - no users in realm",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				// No user operations should be called
				mockNext.On("ServeRequest", mock.Anything, &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						KeycloakRef: common.KeycloakRef{
							Name: "test-keycloak",
						},
						Users: []keycloakApi.User{},
					},
				}, mockClient).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "success - user already exists",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{
						{
							Username:   "existinguser",
							RealmRoles: []string{"role1"},
						},
					},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				expectedUser := &dto.User{
					Username:   "existinguser",
					RealmRoles: []string{"role1"},
				}
				mockClient.On("ExistRealmUser", "test-realm", expectedUser).Return(true, nil)
				// CreateRealmUser should not be called
				mockNext.On("ServeRequest", mock.Anything, &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						KeycloakRef: common.KeycloakRef{
							Name: "test-keycloak",
						},
						Users: []keycloakApi.User{
							{
								Username:   "existinguser",
								RealmRoles: []string{"role1"},
							},
						},
					},
				}, mockClient).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "success - mixed existing and new users",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{
						{
							Username:   "existinguser",
							RealmRoles: []string{"role1"},
						},
						{
							Username:   "newuser",
							RealmRoles: []string{"role2"},
						},
					},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				expectedExistingUser := &dto.User{
					Username:   "existinguser",
					RealmRoles: []string{"role1"},
				}
				expectedNewUser := &dto.User{
					Username:   "newuser",
					RealmRoles: []string{"role2"},
				}
				mockClient.On("ExistRealmUser", "test-realm", expectedExistingUser).Return(true, nil)
				mockClient.On("ExistRealmUser", "test-realm", expectedNewUser).Return(false, nil)
				mockClient.On("CreateRealmUser", "test-realm", expectedNewUser).Return(nil)
				mockNext.On("ServeRequest", mock.Anything, &keycloakApi.KeycloakRealm{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-realm",
						Namespace: "default",
					},
					Spec: keycloakApi.KeycloakRealmSpec{
						RealmName: "test-realm",
						KeycloakRef: common.KeycloakRef{
							Name: "test-keycloak",
						},
						Users: []keycloakApi.User{
							{
								Username:   "existinguser",
								RealmRoles: []string{"role1"},
							},
							{
								Username:   "newuser",
								RealmRoles: []string{"role2"},
							},
						},
					},
				}, mockClient).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "error - ExistRealmUser fails",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{
						{
							Username:   "testuser",
							RealmRoles: []string{"role1"},
						},
					},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				expectedUser := &dto.User{
					Username:   "testuser",
					RealmRoles: []string{"role1"},
				}
				mockClient.On("ExistRealmUser", "test-realm", expectedUser).Return(false, errors.New("existence check failed"))
				// No further calls should be made
			},
			expectedError: "error during createUsers: error during createOneUser: error during exist ream user check: existence check failed",
		},
		{
			name: "error - CreateRealmUser fails",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{
						{
							Username:   "testuser",
							RealmRoles: []string{"role1"},
						},
					},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				expectedUser := &dto.User{
					Username:   "testuser",
					RealmRoles: []string{"role1"},
				}
				mockClient.On("ExistRealmUser", "test-realm", expectedUser).Return(false, nil)
				mockClient.On("CreateRealmUser", "test-realm", expectedUser).Return(errors.New("user creation failed"))
				// No further calls should be made
			},
			expectedError: "error during createUsers: error during createOneUser: unable to create user in realm: user creation failed",
		},
		{
			name: "error - multiple users, second fails existence check",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{
						{
							Username:   "user1",
							RealmRoles: []string{"role1"},
						},
						{
							Username:   "user2",
							RealmRoles: []string{"role2"},
						},
					},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				expectedUser1 := &dto.User{
					Username:   "user1",
					RealmRoles: []string{"role1"},
				}
				expectedUser2 := &dto.User{
					Username:   "user2",
					RealmRoles: []string{"role2"},
				}
				mockClient.On("ExistRealmUser", "test-realm", expectedUser1).Return(false, nil)
				mockClient.On("CreateRealmUser", "test-realm", expectedUser1).Return(nil)
				mockClient.On("ExistRealmUser", "test-realm", expectedUser2).Return(false, errors.New("second user check failed"))
				// No further calls should be made
			},
			expectedError: "error during createUsers: error during createOneUser: error during exist ream user check: second user check failed",
		},
		{
			name: "success - next handler is nil",
			realm: &keycloakApi.KeycloakRealm{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-realm",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					KeycloakRef: common.KeycloakRef{
						Name: "test-keycloak",
					},
					Users: []keycloakApi.User{
						{
							Username:   "testuser",
							RealmRoles: []string{"role1"},
						},
					},
				},
			},
			mockSetup: func(mockClient *keycloakmocks.MockClient, mockNext *handlermocks.MockRealmHandler) {
				expectedUser := &dto.User{
					Username:   "testuser",
					RealmRoles: []string{"role1"},
				}
				mockClient.On("ExistRealmUser", "test-realm", expectedUser).Return(false, nil)
				mockClient.On("CreateRealmUser", "test-realm", expectedUser).Return(nil)
				// mockNext will be nil in this test case
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := keycloakmocks.NewMockClient(t)

			// Create mock next handler (if needed)
			var mockNext *handlermocks.MockRealmHandler

			var putUsers PutUsers

			if tt.name != "success - next handler is nil" {
				mockNext = &handlermocks.MockRealmHandler{}
				putUsers = PutUsers{next: mockNext}
			} else {
				putUsers = PutUsers{next: nil}
			}

			// Set up mocks
			tt.mockSetup(mockClient, mockNext)

			// Execute the method
			err := putUsers.ServeRequest(context.Background(), tt.realm, mockClient)

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
