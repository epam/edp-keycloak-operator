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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	keycloakmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestNewCreateOrUpdateUser(t *testing.T) {
	h := NewCreateOrUpdateUser(nil)
	require.NotNil(t, h)
}

func TestCreateOrUpdateUser_Serve(t *testing.T) {
	tests := []struct {
		name       string
		user       *keycloakApi.KeycloakRealmUser
		realm      *gocloak.RealmRepresentation
		mockSetup  func(*keycloakmocks.MockClient)
		wantErr    require.ErrorAssertionFunc
		expectedID string
	}{
		{
			name: "success - create new user",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:      "testuser",
					Email:         "test@example.com",
					FirstName:     "Test",
					LastName:      "User",
					Enabled:       true,
					EmailVerified: true,
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().CreateOrUpdateUser(
					context.Background(),
					"test-realm",
					&adapter.KeycloakUser{
						Username:      "testuser",
						Email:         "test@example.com",
						FirstName:     "Test",
						LastName:      "User",
						Enabled:       true,
						EmailVerified: true,
					},
					false,
				).Return("user-123", nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-123",
		},
		{
			name: "success - create user with required actions",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:            "testuser",
					RequiredUserActions: []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().CreateOrUpdateUser(
					context.Background(),
					"test-realm",
					&adapter.KeycloakUser{
						Username:            "testuser",
						RequiredUserActions: []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
					},
					false,
				).Return("user-456", nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-456",
		},
		{
			name: "success - create user with attributes",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					AttributesV2: map[string][]string{
						"department": {"engineering"},
						"team":       {"platform", "infra"},
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().CreateOrUpdateUser(
					context.Background(),
					"test-realm",
					&adapter.KeycloakUser{
						Username: "testuser",
						Attributes: map[string][]string{
							"department": {"engineering"},
							"team":       {"platform", "infra"},
						},
					},
					false,
				).Return("user-789", nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-789",
		},
		{
			name: "success - create user with add-only reconciliation strategy",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:               "testuser",
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().CreateOrUpdateUser(
					context.Background(),
					"test-realm",
					&adapter.KeycloakUser{
						Username: "testuser",
					},
					true,
				).Return("user-add-only", nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-add-only",
		},
		{
			name: "success - minimal user with only username",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "minimal-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "minimaluser",
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().CreateOrUpdateUser(
					context.Background(),
					"test-realm",
					&adapter.KeycloakUser{
						Username: "minimaluser",
					},
					false,
				).Return("minimal-user-id", nil)
			},
			wantErr:    require.NoError,
			expectedID: "minimal-user-id",
		},
		{
			name: "error - CreateOrUpdateUser fails",
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
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().CreateOrUpdateUser(
					context.Background(),
					"test-realm",
					&adapter.KeycloakUser{
						Username: "testuser",
					},
					false,
				).Return("", errors.New("keycloak api error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...interface{}) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to create or update user: keycloak api error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := keycloakmocks.NewMockClient(t)
			tt.mockSetup(mockClient)

			h := NewCreateOrUpdateUser(nil)
			userCtx := &UserContext{}

			err := h.Serve(context.Background(), tt.user, mockClient, tt.realm, userCtx)

			tt.wantErr(t, err)

			if err == nil {
				assert.Equal(t, tt.expectedID, userCtx.UserID)
			}
		})
	}
}
