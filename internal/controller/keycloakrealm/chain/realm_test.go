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
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	handlermocks "github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler/mocks"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
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
		setupMocks    func(mockRealms *v2mocks.MockRealmClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler)
		expectError   bool
		errorContains string
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
			setupMocks: func(mockRealms *v2mocks.MockRealmClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				mockRealms.EXPECT().GetRealm(mock.Anything, realmName).
					Return(nil, nil, &keycloakv2.ApiError{Code: 404})
				mockRealms.EXPECT().CreateRealm(mock.Anything, mock.MatchedBy(func(r keycloakv2.RealmRepresentation) bool {
					return r.Realm != nil && *r.Realm == realmName
				})).Return(nil, nil)

				// Role checks (3 unique roles)
				mockRoles.EXPECT().GetRealmRole(mock.Anything, realmName, mock.AnythingOfType("string")).
					Return(nil, nil, &keycloakv2.ApiError{Code: 404}).Times(3)
				mockRoles.EXPECT().CreateRealmRole(mock.Anything, realmName, mock.Anything).
					Return(nil, nil).Times(3)

				nextHandler.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "success - realm already exists, skips creation",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return baseRealm.DeepCopy()
			},
			setupMocks: func(mockRealms *v2mocks.MockRealmClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				mockRealms.EXPECT().GetRealm(mock.Anything, realmName).
					Return(&keycloakv2.RealmRepresentation{}, nil, nil)
				nextHandler.EXPECT().ServeRequest(mock.Anything, mock.Anything, mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name: "error - GetRealm fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return baseRealm.DeepCopy()
			},
			setupMocks: func(mockRealms *v2mocks.MockRealmClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				mockRealms.EXPECT().GetRealm(mock.Anything, realmName).
					Return(nil, nil, errors.New("keycloak connection error"))
			},
			expectError:   true,
			errorContains: "unable to check realm existence",
		},
		{
			name: "error - CreateRealm fails",
			setupRealm: func() *keycloakApi.KeycloakRealm {
				return baseRealm.DeepCopy()
			},
			setupMocks: func(mockRealms *v2mocks.MockRealmClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				mockRealms.EXPECT().GetRealm(mock.Anything, realmName).
					Return(nil, nil, &keycloakv2.ApiError{Code: 404})
				mockRealms.EXPECT().CreateRealm(mock.Anything, mock.Anything).
					Return(nil, errors.New("realm creation failed"))
			},
			expectError:   true,
			errorContains: "unable to create realm with default config",
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
			setupMocks: func(mockRealms *v2mocks.MockRealmClient, mockRoles *v2mocks.MockRolesClient, nextHandler *handlermocks.MockRealmHandler) {
				mockRealms.EXPECT().GetRealm(mock.Anything, realmName).
					Return(nil, nil, &keycloakv2.ApiError{Code: 404})
				mockRealms.EXPECT().CreateRealm(mock.Anything, mock.Anything).Return(nil, nil)
				mockRoles.EXPECT().GetRealmRole(mock.Anything, realmName, "role1").
					Return(nil, nil, &keycloakv2.ApiError{Code: 404})
				mockRoles.EXPECT().CreateRealmRole(mock.Anything, realmName, mock.Anything).
					Return(nil, errors.New("role creation failed"))
			},
			expectError:   true,
			errorContains: "unable to create realm roles on no sso scenario",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRealms := v2mocks.NewMockRealmClient(t)
			mockRoles := v2mocks.NewMockRolesClient(t)

			var nextHandler *handlermocks.MockRealmHandler
			if !tt.expectError || tt.name == "success - realm already exists, skips creation" {
				nextHandler = handlermocks.NewMockRealmHandler(t)
			}

			putRealm := PutRealm{
				next:   nextHandler,
				client: fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build(),
			}

			realm := tt.setupRealm()
			tt.setupMocks(mockRealms, mockRoles, nextHandler)

			kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealms, Roles: mockRoles}
			err := putRealm.ServeRequest(ctx, realm, kClientV2)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPutRealm_putRealmRoles(t *testing.T) {
	realmName := "test-realm"

	tests := []struct {
		name          string
		users         []keycloakApi.User
		setupMocks    func(mockRoles *v2mocks.MockRolesClient)
		expectError   bool
		errorContains string
	}{
		{
			name:  "success - no users",
			users: []keycloakApi.User{},
			setupMocks: func(mockRoles *v2mocks.MockRolesClient) {
			},
			expectError: false,
		},
		{
			name: "success - users with no roles",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{}},
				{Username: "user2"},
			},
			setupMocks: func(mockRoles *v2mocks.MockRolesClient) {
			},
			expectError: false,
		},
		{
			name: "success - create all roles",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1", "role2"}},
				{Username: "user2", RealmRoles: []string{"role3"}},
			},
			setupMocks: func(mockRoles *v2mocks.MockRolesClient) {
				mockRoles.EXPECT().GetRealmRole(mock.Anything, realmName, mock.AnythingOfType("string")).
					Return(nil, nil, &keycloakv2.ApiError{Code: 404}).Times(3)
				mockRoles.EXPECT().CreateRealmRole(mock.Anything, realmName, mock.Anything).
					Return(nil, nil).Times(3)
			},
			expectError: false,
		},
		{
			name: "success - some roles exist",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1", "role2"}},
			},
			setupMocks: func(mockRoles *v2mocks.MockRolesClient) {
				// role1 exists
				mockRoles.EXPECT().GetRealmRole(mock.Anything, realmName, "role1").
					Return(ptr.To(keycloakv2.RoleRepresentation{}), nil, nil)
				// role2 doesn't
				mockRoles.EXPECT().GetRealmRole(mock.Anything, realmName, "role2").
					Return(nil, nil, &keycloakv2.ApiError{Code: 404})
				mockRoles.EXPECT().CreateRealmRole(mock.Anything, realmName, mock.Anything).
					Return(nil, nil)
			},
			expectError: false,
		},
		{
			name: "success - duplicate roles handled correctly",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1", "role1"}},
				{Username: "user2", RealmRoles: []string{"role1", "role2"}},
			},
			setupMocks: func(mockRoles *v2mocks.MockRolesClient) {
				mockRoles.EXPECT().GetRealmRole(mock.Anything, realmName, mock.AnythingOfType("string")).
					Return(nil, nil, &keycloakv2.ApiError{Code: 404}).Times(2)
				mockRoles.EXPECT().CreateRealmRole(mock.Anything, realmName, mock.Anything).
					Return(nil, nil).Times(2)
			},
			expectError: false,
		},
		{
			name: "error - GetRealmRole fails",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1"}},
			},
			setupMocks: func(mockRoles *v2mocks.MockRolesClient) {
				mockRoles.EXPECT().GetRealmRole(mock.Anything, realmName, "role1").
					Return(nil, nil, errors.New("role check failed"))
			},
			expectError:   true,
			errorContains: "unable to check realm role existence",
		},
		{
			name: "error - CreateRealmRole fails",
			users: []keycloakApi.User{
				{Username: "user1", RealmRoles: []string{"role1"}},
			},
			setupMocks: func(mockRoles *v2mocks.MockRolesClient) {
				mockRoles.EXPECT().GetRealmRole(mock.Anything, realmName, "role1").
					Return(nil, nil, &keycloakv2.ApiError{Code: 404})
				mockRoles.EXPECT().CreateRealmRole(mock.Anything, realmName, mock.Anything).
					Return(nil, errors.New("role creation failed"))
			},
			expectError:   true,
			errorContains: "unable to create new realm role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRoles := v2mocks.NewMockRolesClient(t)
			putRealm := PutRealm{}

			realm := &keycloakApi.KeycloakRealm{
				Spec: keycloakApi.KeycloakRealmSpec{
					RealmName: realmName,
					Users:     tt.users,
				},
			}

			tt.setupMocks(mockRoles)

			kClientV2 := &keycloakv2.KeycloakClient{Roles: mockRoles}
			err := putRealm.putRealmRoles(context.Background(), realm, kClientV2)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
