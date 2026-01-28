package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	helpermock "github.com/epam/edp-keycloak-operator/internal/controller/helper/mocks"
	handlermocks "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestPutRealm_ServeRequest(t *testing.T) {
	ctx := context.Background()
	realmName := "test-realm"
	namespace := "test-namespace"
	keycloakRefName := "test-keycloak"

	baseRealm := &keycloakApi.KeycloakRealm{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-realm-cr",
			Namespace: namespace,
		},
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: realmName,
			KeycloakRef: common.KeycloakRef{
				Name: keycloakRefName,
			},
		},
	}

	tests := []struct {
		name          string
		setupRealm    func() *keycloakApi.KeycloakRealm
		setupMocks    func(kClient *mocks.MockClient, hlp *helpermock.MockControllerHelper, nextHandler *handlermocks.MockRealmHandler)
		expectError   bool
		errorContains string
		checkHelper   func(hlp *helpermock.MockControllerHelper)
	}{
		{
			name: "success - realm does not exist, creates realm and roles",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				realm := baseRealm.DeepCopy()
				realm.Spec.Users = []keycloakApi.User{
					{Username: "user1", RealmRoles: []string{"role1", "role2"}},
					{Username: "user2", RealmRoles: []string{"role2", "role3"}},
				}
				return realm
			},
			setupMocks: func(kClient *mocks.MockClient, hlp *helpermock.MockControllerHelper, nextHandler *handlermocks.MockRealmHandler) {
				expectedRealm := &dto.Realm{
					Name: realmName,
					Users: []dto.User{
						{Username: "user1", RealmRoles: []string{"role1", "role2"}},
						{Username: "user2", RealmRoles: []string{"role2", "role3"}},
					},
				}

				kClient.On("ExistRealm", realmName).Return(false, nil)
				kClient.On("CreateRealmWithDefaultConfig", expectedRealm).Return(nil)

				// Role existence checks
				kClient.On("ExistRealmRole", realmName, "role1").Return(false, nil)
				kClient.On("ExistRealmRole", realmName, "role2").Return(false, nil)
				kClient.On("ExistRealmRole", realmName, "role3").Return(false, nil)

				// Role creation
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role1"}).Return(nil)
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role2"}).Return(nil)
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role3"}).Return(nil)

				hlp.On("InvalidateKeycloakClientTokenSecret", ctx, namespace, keycloakRefName).Return(nil)
				nextHandler.On("ServeRequest", mock.Anything, mock.Anything, kClient).Return(nil)
			},
			expectError: false,
		},
		{
			name: "success - realm already exists, skips creation",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return baseRealm.DeepCopy()
			},
			setupMocks: func(kClient *mocks.MockClient, hlp *helpermock.MockControllerHelper, nextHandler *handlermocks.MockRealmHandler) {
				kClient.On("ExistRealm", realmName).Return(true, nil)
				nextHandler.On("ServeRequest", mock.Anything, mock.Anything, kClient).Return(nil)
			},
			expectError: false,
			checkHelper: func(hlp *helpermock.MockControllerHelper) {
				hlp.AssertNotCalled(t, "InvalidateKeycloakClientTokenSecret")
			},
		},
		{
			name: "error - ExistRealm fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return baseRealm.DeepCopy()
			},
			setupMocks: func(kClient *mocks.MockClient, hlp *helpermock.MockControllerHelper, nextHandler *handlermocks.MockRealmHandler) {
				kClient.On("ExistRealm", realmName).Return(false, errors.New("keycloak connection error"))
			},
			expectError:   true,
			errorContains: "unable to check realm existence",
			checkHelper: func(hlp *helpermock.MockControllerHelper) {
				hlp.AssertNotCalled(t, "InvalidateKeycloakClientTokenSecret")
			},
		},
		{
			name: "error - CreateRealmWithDefaultConfig fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return baseRealm.DeepCopy()
			},
			setupMocks: func(kClient *mocks.MockClient, hlp *helpermock.MockControllerHelper, nextHandler *handlermocks.MockRealmHandler) {
				expectedRealm := &dto.Realm{
					Name:  realmName,
					Users: []dto.User{},
				}
				kClient.On("ExistRealm", realmName).Return(false, nil)
				kClient.On("CreateRealmWithDefaultConfig", expectedRealm).Return(errors.New("realm creation failed"))
			},
			expectError:   true,
			errorContains: "unable to create realm with default config",
			checkHelper: func(hlp *helpermock.MockControllerHelper) {
				hlp.AssertNotCalled(t, "InvalidateKeycloakClientTokenSecret")
			},
		},
		{
			name: "error - role creation fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				realm := baseRealm.DeepCopy()
				realm.Spec.Users = []keycloakApi.User{
					{Username: "user1", RealmRoles: []string{"role1"}},
				}
				return realm
			},
			setupMocks: func(kClient *mocks.MockClient, hlp *helpermock.MockControllerHelper, nextHandler *handlermocks.MockRealmHandler) {
				expectedRealm := &dto.Realm{
					Name: realmName,
					Users: []dto.User{
						{Username: "user1", RealmRoles: []string{"role1"}},
					},
				}
				kClient.On("ExistRealm", realmName).Return(false, nil)
				kClient.On("CreateRealmWithDefaultConfig", expectedRealm).Return(nil)
				kClient.On("ExistRealmRole", realmName, "role1").Return(false, nil)
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role1"}).Return(errors.New("role creation failed"))
			},
			expectError:   true,
			errorContains: "unable to create realm roles on no sso scenario",
			checkHelper: func(hlp *helpermock.MockControllerHelper) {
				hlp.AssertNotCalled(t, "InvalidateKeycloakClientTokenSecret")
			},
		},
		{
			name: "error - InvalidateKeycloakClientTokenSecret fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return baseRealm.DeepCopy()
			},
			setupMocks: func(kClient *mocks.MockClient, hlp *helpermock.MockControllerHelper, nextHandler *handlermocks.MockRealmHandler) {
				expectedRealm := &dto.Realm{
					Name:  realmName,
					Users: []dto.User{},
				}
				kClient.On("ExistRealm", realmName).Return(false, nil)
				kClient.On("CreateRealmWithDefaultConfig", expectedRealm).Return(nil)
				hlp.On("InvalidateKeycloakClientTokenSecret", ctx, namespace, keycloakRefName).Return(errors.New("token invalidation failed"))
			},
			expectError:   true,
			errorContains: "unable invalidate keycloak client token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			kClient := mocks.NewMockClient(t)
			hlp := helpermock.NewMockControllerHelper(t)

			var nextHandler *handlermocks.MockRealmHandler

			// Only create next handler if we expect success scenarios or specific error scenarios
			if !tt.expectError || tt.name == "success - realm already exists, skips creation" {
				nextHandler = &handlermocks.MockRealmHandler{}
			}

			putRealm := PutRealm{
				next:   nextHandler,
				client: fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(),
				hlp:    hlp,
			}

			realm := tt.setupRealm()
			tt.setupMocks(kClient, hlp, nextHandler)

			// Execute
			err := putRealm.ServeRequest(ctx, realm, kClient)

			// Assert
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)

			} else {
				require.NoError(t, err)
			}

			// Additional checks
			if tt.checkHelper != nil {
				tt.checkHelper(hlp)
			}
		})
	}
}

func TestPutRealm_putRealmRoles(t *testing.T) {
	realmName := "test-realm"

	tests := []struct {
		name          string
		users         []keycloakApi.User
		setupMocks    func(kClient *mocks.MockClient)
		expectError   bool
		errorContains string
	}{
		{
			name:  "success - no users",
			users: []keycloakApi.User{},
			setupMocks: func(kClient *mocks.MockClient) {
				// No mocks needed for empty users
			},
			expectError: false,
		},
		{
			name: "success - users with no roles",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{}},
				{Username: "user2"}, // No roles field
			},
			setupMocks: func(kClient *mocks.MockClient) {
				// No mocks needed for users without roles
			},
			expectError: false,
		},
		{
			name: "success - create all roles",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1", "role2"}},
				{Username: "user2", RealmRoles: []string{"role3"}},
			},
			setupMocks: func(kClient *mocks.MockClient) {
				kClient.On("ExistRealmRole", realmName, "role1").Return(false, nil)
				kClient.On("ExistRealmRole", realmName, "role2").Return(false, nil)
				kClient.On("ExistRealmRole", realmName, "role3").Return(false, nil)

				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role1"}).Return(nil)
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role2"}).Return(nil)
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role3"}).Return(nil)
			},
			expectError: false,
		},
		{
			name: "success - some roles exist",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1", "role2"}},
			},
			setupMocks: func(kClient *mocks.MockClient) {
				// role1 exists, role2 doesn't
				kClient.On("ExistRealmRole", realmName, "role1").Return(true, nil)
				kClient.On("ExistRealmRole", realmName, "role2").Return(false, nil)

				// Only role2 should be created
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role2"}).Return(nil)
			},
			expectError: false,
		},
		{
			name: "success - duplicate roles handled correctly",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1", "role1"}}, // Duplicate
				{Username: "user2", RealmRoles: []string{"role1", "role2"}}, // role1 again
			},
			setupMocks: func(kClient *mocks.MockClient) {
				// Each unique role should only be checked once
				kClient.On("ExistRealmRole", realmName, "role1").Return(false, nil)
				kClient.On("ExistRealmRole", realmName, "role2").Return(false, nil)

				// Each unique role should only be created once
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role1"}).Return(nil)
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role2"}).Return(nil)
			},
			expectError: false,
		},
		{
			name: "error - ExistRealmRole fails",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1"}},
			},
			setupMocks: func(kClient *mocks.MockClient) {
				kClient.On("ExistRealmRole", realmName, "role1").Return(false, errors.New("role check failed"))
			},
			expectError:   true,
			errorContains: "unable to check realm role existence",
		},
		{
			name: "error - CreateIncludedRealmRole fails",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1"}},
			},
			setupMocks: func(kClient *mocks.MockClient) {
				kClient.On("ExistRealmRole", realmName, "role1").Return(false, nil)
				kClient.On("CreateIncludedRealmRole", realmName, &dto.IncludedRealmRole{Name: "role1"}).Return(errors.New("role creation failed"))
			},
			expectError:   true,
			errorContains: "unable to create new realm role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			kClient := mocks.NewMockClient(t)
			putRealm := PutRealm{}

			realm := &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: realmName,
					Users:     tt.users,
				},
			}

			tt.setupMocks(kClient)

			// Execute
			err := putRealm.putRealmRoles(realm, kClient)

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
