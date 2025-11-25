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

func TestNewSyncUserGroups(t *testing.T) {
	h := NewSyncUserGroups()
	require.NotNil(t, h)
}

func TestSyncUserGroups_Serve(t *testing.T) {
	tests := []struct {
		name      string
		user      *keycloakApi.KeycloakRealmUser
		realm     *gocloak.RealmRepresentation
		userCtx   *UserContext
		mockSetup func(*keycloakmocks.MockClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "success - sync user groups with full reconciliation strategy",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					Groups:   []string{"group1", "group2"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-123",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserGroups(
					context.Background(),
					"test-realm",
					"user-123",
					[]string{"group1", "group2"},
					false,
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - sync user groups with add-only reconciliation strategy",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:               "testuser",
					Groups:                 []string{"group1"},
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
				m.EXPECT().SyncUserGroups(
					context.Background(),
					"test-realm",
					"user-456",
					[]string{"group1"},
					true,
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - sync user with empty groups",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					Groups:   []string{},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-789",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserGroups(
					context.Background(),
					"test-realm",
					"user-789",
					[]string{},
					false,
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - sync user with nil groups",
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
				UserID: "user-nil-groups",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserGroups(
					context.Background(),
					"test-realm",
					"user-nil-groups",
					[]string(nil),
					false,
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "error - SyncUserGroups fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					Groups:   []string{"group1"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-error",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserGroups(
					context.Background(),
					"test-realm",
					"user-error",
					[]string{"group1"},
					false,
				).Return(errors.New("keycloak api error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to sync user groups: keycloak api error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := keycloakmocks.NewMockClient(t)
			tt.mockSetup(mockClient)

			h := NewSyncUserGroups()

			err := h.Serve(context.Background(), tt.user, mockClient, tt.realm, tt.userCtx)

			tt.wantErr(t, err)
		})
	}
}
