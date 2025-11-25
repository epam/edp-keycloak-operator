package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestNewSyncUserRoles(t *testing.T) {
	h := NewSyncUserRoles()
	require.NotNil(t, h)
}

func TestSyncUserRoles_Serve(t *testing.T) {
	tests := []struct {
		name      string
		user      *keycloakApi.KeycloakRealmUser
		realm     *gocloak.RealmRepresentation
		userCtx   *UserContext
		mockSetup func(*keycloakmocks.MockClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "success - sync user roles with full reconciliation strategy",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					Roles:    []string{"role1", "role2"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-123",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserRoles(
					context.Background(),
					"test-realm",
					"user-123",
					[]string{"role1", "role2"},
					map[string][]string(nil),
					false,
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - sync user roles with add-only reconciliation strategy",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:               "testuser",
					Roles:                  []string{"role1"},
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-456",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserRoles(
					context.Background(),
					"test-realm",
					"user-456",
					[]string{"role1"},
					map[string][]string(nil),
					true,
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - sync user with client roles",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					Roles:    []string{"realm-role"},
					ClientRoles: []keycloakApi.UserClientRole{
						{
							ClientID: "client1",
							Roles:    []string{"client-role1", "client-role2"},
						},
						{
							ClientID: "client2",
							Roles:    []string{"client-role3"},
						},
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-789",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserRoles(
					context.Background(),
					"test-realm",
					"user-789",
					[]string{"realm-role"},
					map[string][]string{
						"client1": {"client-role1", "client-role2"},
						"client2": {"client-role3"},
					},
					false,
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - sync user with empty roles",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					Roles:    []string{},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-empty",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserRoles(
					context.Background(),
					"test-realm",
					"user-empty",
					[]string{},
					map[string][]string(nil),
					false,
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - sync user with nil roles",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-nil-roles",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserRoles(
					context.Background(),
					"test-realm",
					"user-nil-roles",
					[]string(nil),
					map[string][]string(nil),
					false,
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "error - SyncUserRoles fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					Roles:    []string{"role1"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-error",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserRoles(
					context.Background(),
					"test-realm",
					"user-error",
					[]string{"role1"},
					map[string][]string(nil),
					false,
				).Return(errors.New("keycloak api error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to sync user roles: keycloak api error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := keycloakmocks.NewMockClient(t)
			tt.mockSetup(mockClient)

			h := NewSyncUserRoles()

			err := h.Serve(context.Background(), tt.user, mockClient, tt.realm, tt.userCtx)

			tt.wantErr(t, err)
		})
	}
}

func TestConvertClientRoles(t *testing.T) {
	tests := []struct {
		name           string
		apiClientRoles []keycloakApi.UserClientRole
		want           map[string][]string
	}{
		{
			name:           "nil input returns nil",
			apiClientRoles: nil,
			want:           nil,
		},
		{
			name:           "empty slice returns empty map",
			apiClientRoles: []keycloakApi.UserClientRole{},
			want:           map[string][]string{},
		},
		{
			name: "single client role",
			apiClientRoles: []keycloakApi.UserClientRole{
				{
					ClientID: "client1",
					Roles:    []string{"role1", "role2"},
				},
			},
			want: map[string][]string{
				"client1": {"role1", "role2"},
			},
		},
		{
			name: "multiple client roles",
			apiClientRoles: []keycloakApi.UserClientRole{
				{
					ClientID: "client1",
					Roles:    []string{"role1"},
				},
				{
					ClientID: "client2",
					Roles:    []string{"role2", "role3"},
				},
			},
			want: map[string][]string{
				"client1": {"role1"},
				"client2": {"role2", "role3"},
			},
		},
		{
			name: "client role with empty roles",
			apiClientRoles: []keycloakApi.UserClientRole{
				{
					ClientID: "client1",
					Roles:    []string{},
				},
			},
			want: map[string][]string{
				"client1": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertClientRoles(tt.apiClientRoles)
			assert.Equal(t, tt.want, got)
		})
	}
}
