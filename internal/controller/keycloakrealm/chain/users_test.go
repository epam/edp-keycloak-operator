package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	handlermocks "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler/mocks"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestPutUsers_ServeRequest(t *testing.T) {
	tests := []struct {
		name          string
		realm         *keycloakApi.KeycloakRealm
		mockSetup     func(*v2mocks.MockUsersClient, *handlermocks.MockRealmHandler)
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
			mockSetup: func(mockUsers *v2mocks.MockUsersClient, mockNext *handlermocks.MockRealmHandler) {
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "testuser1").
					Return(nil, nil, nil)
				mockUsers.EXPECT().CreateUser(mock.Anything, "test-realm", mock.MatchedBy(func(u keycloakv2.UserRepresentation) bool {
					return u.Username != nil && *u.Username == "testuser1"
				})).Return(nil, nil)
				mockNext.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
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
					Users: []keycloakApi.User{
						{Username: "user1"},
						{Username: "user2"},
					},
				},
			},
			mockSetup: func(mockUsers *v2mocks.MockUsersClient, mockNext *handlermocks.MockRealmHandler) {
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user1").
					Return(nil, nil, nil)
				mockUsers.EXPECT().CreateUser(mock.Anything, "test-realm", mock.Anything).
					Return(nil, nil).Once()
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "user2").
					Return(nil, nil, nil)
				mockUsers.EXPECT().CreateUser(mock.Anything, "test-realm", mock.Anything).
					Return(nil, nil).Once()
				mockNext.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "success - no users in realm",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					Users:     []keycloakApi.User{},
				},
			},
			mockSetup: func(mockUsers *v2mocks.MockUsersClient, mockNext *handlermocks.MockRealmHandler) {
				mockNext.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "success - user already exists",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					Users: []keycloakApi.User{
						{Username: "existinguser"},
					},
				},
			},
			mockSetup: func(mockUsers *v2mocks.MockUsersClient, mockNext *handlermocks.MockRealmHandler) {
				uid := "some-uid"
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "existinguser").
					Return(&keycloakv2.UserRepresentation{Id: ptr.To(uid)}, nil, nil)
				mockNext.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectedError: "",
		},
		{
			name: "error - FindUserByUsername fails",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					Users: []keycloakApi.User{
						{Username: "testuser"},
					},
				},
			},
			mockSetup: func(mockUsers *v2mocks.MockUsersClient, mockNext *handlermocks.MockRealmHandler) {
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "testuser").
					Return(nil, nil, errors.New("existence check failed"))
			},
			expectedError: "error during exist realm user check",
		},
		{
			name: "error - CreateUser fails",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					Users: []keycloakApi.User{
						{Username: "testuser"},
					},
				},
			},
			mockSetup: func(mockUsers *v2mocks.MockUsersClient, mockNext *handlermocks.MockRealmHandler) {
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "testuser").
					Return(nil, nil, nil)
				mockUsers.EXPECT().CreateUser(mock.Anything, "test-realm", mock.Anything).
					Return(nil, errors.New("user creation failed"))
			},
			expectedError: "unable to create user in realm",
		},
		{
			name: "success - next handler is nil",
			realm: &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: "test-realm",
					Users: []keycloakApi.User{
						{Username: "testuser"},
					},
				},
			},
			mockSetup: func(mockUsers *v2mocks.MockUsersClient, mockNext *handlermocks.MockRealmHandler) {
				mockUsers.EXPECT().FindUserByUsername(mock.Anything, "test-realm", "testuser").
					Return(nil, nil, nil)
				mockUsers.EXPECT().CreateUser(mock.Anything, "test-realm", mock.Anything).
					Return(nil, nil)
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsers := v2mocks.NewMockUsersClient(t)

			var mockNext *handlermocks.MockRealmHandler

			var putUsers PutUsers

			if tt.name != "success - next handler is nil" {
				mockNext = handlermocks.NewMockRealmHandler(t)
				putUsers = PutUsers{next: mockNext}
			} else {
				putUsers = PutUsers{next: nil}
			}

			tt.mockSetup(mockUsers, mockNext)

			kClientV2 := &keycloakv2.KeycloakClient{Users: mockUsers}
			err := putUsers.ServeRequest(context.Background(), tt.realm, kClientV2)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
