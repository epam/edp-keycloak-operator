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

func TestNewSyncUserIdentityProviders(t *testing.T) {
	h := NewSyncUserIdentityProviders()
	require.NotNil(t, h)
}

func TestSyncUserIdentityProviders_Serve(t *testing.T) {
	tests := []struct {
		name      string
		user      *keycloakApi.KeycloakRealmUser
		realm     *gocloak.RealmRepresentation
		userCtx   *UserContext
		mockSetup func(*keycloakmocks.MockClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "success - sync user identity providers",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{"idp1", "idp2"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-123",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserIdentityProviders(
					context.Background(),
					"test-realm",
					"user-123",
					"testuser",
					[]string{"idp1", "idp2"},
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - nil identity providers skips sync",
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
				UserID: "user-456",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				// No mock setup needed - should return early
			},
			wantErr: require.NoError,
		},
		{
			name: "success - empty identity providers list",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-789",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserIdentityProviders(
					context.Background(),
					"test-realm",
					"user-789",
					"testuser",
					[]string{},
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "success - single identity provider",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{"google"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-single-idp",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserIdentityProviders(
					context.Background(),
					"test-realm",
					"user-single-idp",
					"testuser",
					[]string{"google"},
				).Return(nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "error - SyncUserIdentityProviders fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:          "testuser",
					IdentityProviders: &[]string{"idp1"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			userCtx: &UserContext{
				UserID: "user-error",
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SyncUserIdentityProviders(
					context.Background(),
					"test-realm",
					"user-error",
					"testuser",
					[]string{"idp1"},
				).Return(errors.New("keycloak api error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to sync user identity providers: keycloak api error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := keycloakmocks.NewMockClient(t)
			tt.mockSetup(mockClient)

			h := NewSyncUserIdentityProviders()

			err := h.Serve(context.Background(), tt.user, mockClient, tt.realm, tt.userCtx)

			tt.wantErr(t, err)
		})
	}
}
