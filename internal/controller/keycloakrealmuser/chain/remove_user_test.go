package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestNewRemoveUser(t *testing.T) {
	h := NewRemoveUser(nil)
	require.NotNil(t, h)
}

func TestRemoveUser_ServeRequest(t *testing.T) {
	tests := []struct {
		name      string
		user      *keycloakApi.KeycloakRealmUser
		mockSetup func(*v2mocks.MockUsersClient)
		wantErr   require.ErrorAssertionFunc
	}{
		{
			name: "success - user deleted",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("user-id-123")}, nil, nil)
				m.EXPECT().DeleteUser(context.Background(), "test-realm", "user-id-123").
					Return(nil, nil)
			},
			wantErr: require.NoError,
		},
		{
			name: "user not found on FindUserByUsername - skip silently",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(nil, nil, keycloakapi.ErrNotFound)
			},
			wantErr: require.NoError,
		},
		{
			name: "user not found on DeleteUser - skip silently",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("user-id-123")}, nil, nil)
				m.EXPECT().DeleteUser(context.Background(), "test-realm", "user-id-123").
					Return(nil, keycloakapi.ErrNotFound)
			},
			wantErr: require.NoError,
		},
		{
			name: "preserve resources on deletion annotation set - skip",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
					Annotations: map[string]string{
						"edp.epam.com/preserve-resources-on-deletion": "true",
					},
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {},
			wantErr:   require.NoError,
		},
		{
			name: "FindUserByUsername returns unexpected error",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(nil, nil, errors.New("connection error"))
			},
			wantErr: require.Error,
		},
		{
			name: "DeleteUser returns unexpected error",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(&keycloakapi.UserRepresentation{Id: ptr.To("user-id-123")}, nil, nil)
				m.EXPECT().DeleteUser(context.Background(), "test-realm", "user-id-123").
					Return(nil, errors.New("delete error"))
			},
			wantErr: require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsers := v2mocks.NewMockUsersClient(t)
			tt.mockSetup(mockUsers)

			h := NewRemoveUser(&keycloakapi.KeycloakClient{
				Users: mockUsers,
			})

			err := h.ServeRequest(context.Background(), tt.user, "test-realm")
			tt.wantErr(t, err)
		})
	}
}
