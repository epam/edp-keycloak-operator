package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestNewCreateOrUpdateUser(t *testing.T) {
	h := NewCreateOrUpdateUser(nil, nil)
	require.NotNil(t, h)
}

func TestCreateOrUpdateUser_Serve(t *testing.T) {
	tests := []struct {
		name       string
		user       *keycloakApi.KeycloakRealmUser
		mockSetup  func(*v2mocks.MockUsersClient)
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
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", keycloakapi.UserRepresentation{
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(true),
					EmailVerified:   ptr.To(true),
					FirstName:       ptr.To("Test"),
					LastName:        ptr.To("User"),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To("test@example.com"),
				}).Return(&keycloakapi.Response{
					HTTPResponse: &http.Response{
						Header: http.Header{"Location": []string{"http://keycloak/users/user-123"}},
					},
				}, nil)
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
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", keycloakapi.UserRepresentation{
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(false),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"}),
					Email:           ptr.To(""),
				}).Return(&keycloakapi.Response{
					HTTPResponse: &http.Response{
						Header: http.Header{"Location": []string{"http://keycloak/users/user-456"}},
					},
				}, nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-456",
		},
		{
			name: "success - update existing user",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:      "testuser",
					Email:         "updated@example.com",
					FirstName:     "Updated",
					LastName:      "User",
					Enabled:       true,
					EmailVerified: true,
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				existingUser := &keycloakapi.UserRepresentation{
					Id:       ptr.To("existing-user-id"),
					Username: ptr.To("testuser"),
					Email:    ptr.To("old@example.com"),
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(existingUser, nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(true),
					EmailVerified:   ptr.To(true),
					FirstName:       ptr.To("Updated"),
					LastName:        ptr.To("User"),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To("updated@example.com"),
				}).Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
		{
			name: "success - update user preserves UPDATE_PASSWORD action",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:            "testuser",
					RequiredUserActions: []string{"VERIFY_EMAIL"},
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				existingUser := &keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To("testuser"),
					RequiredActions: ptr.To([]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"}),
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(existingUser, nil, nil)
				// UPDATE_PASSWORD should be preserved
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(false),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string{"VERIFY_EMAIL", "UPDATE_PASSWORD"}),
					Email:           ptr.To(""),
				}).Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
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
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", keycloakapi.UserRepresentation{
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(false),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To(""),
				}).Return(&keycloakapi.Response{
					HTTPResponse: &http.Response{
						Header: http.Header{"Location": []string{"http://keycloak/users/user-add-only"}},
					},
				}, nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-add-only",
		},
		{
			name: "error - FindUserByUsername fails",
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
					Return(nil, nil, errors.New("keycloak api error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to find user by username")
			},
		},
		{
			name: "error - CreateUser fails",
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
				m.EXPECT().CreateUser(context.Background(), "test-realm", keycloakapi.UserRepresentation{
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(false),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To(""),
				}).Return(nil, errors.New("keycloak api error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to create user")
			},
		},
		{
			name: "error - UpdateUser fails",
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
				existingUser := &keycloakapi.UserRepresentation{
					Id:       ptr.To("existing-user-id"),
					Username: ptr.To("testuser"),
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(existingUser, nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(false),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To(""),
				}).Return(nil, errors.New("keycloak api error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to update user")
			},
		},
		{
			name: "error - CreateUser returns no Location header (empty userID)",
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
				m.EXPECT().CreateUser(context.Background(), "test-realm", keycloakapi.UserRepresentation{
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(false),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To(""),
				}).Return(&keycloakapi.Response{
					HTTPResponse: &http.Response{Header: http.Header{}},
				}, nil)
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to get user ID from response")
			},
		},
		{
			name: "success - create user with attributes",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:     "testuser",
					AttributesV2: map[string][]string{"dept": {"engineering"}},
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(nil, nil, keycloakapi.ErrNotFound)
				attrs := map[string][]string{"dept": {"engineering"}}
				m.EXPECT().CreateUser(context.Background(), "test-realm", keycloakapi.UserRepresentation{
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(false),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To(""),
					Attributes:      &attrs,
				}).Return(&keycloakapi.Response{
					HTTPResponse: &http.Response{
						Header: http.Header{"Location": []string{"http://keycloak/users/user-attr-123"}},
					},
				}, nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-attr-123",
		},
		{
			name: "success - update user with attributes (non-addOnly removes stale keys)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:     "testuser",
					AttributesV2: map[string][]string{"dept": {"engineering"}},
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				existingAttrs := map[string][]string{
					"dept":  {"old-value"},
					"stale": {"should-be-removed"},
				}
				existingUser := &keycloakapi.UserRepresentation{
					Id:         ptr.To("existing-user-id"),
					Username:   ptr.To("testuser"),
					Attributes: &existingAttrs,
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(existingUser, nil, nil)
				mergedAttrs := map[string][]string{"dept": {"engineering"}}
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(false),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To(""),
					Attributes:      &mergedAttrs,
				}).Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
		{
			name: "success - update user with attributes (addOnly merges without removing)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:               "testuser",
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
					AttributesV2:           map[string][]string{"dept": {"engineering", "existing-val"}},
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				existingAttrs := map[string][]string{
					"dept":  {"existing-val"},
					"extra": {"kept"},
				}
				existingUser := &keycloakapi.UserRepresentation{
					Id:         ptr.To("existing-user-id"),
					Username:   ptr.To("testuser"),
					Attributes: &existingAttrs,
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", "testuser").
					Return(existingUser, nil, nil)
				// addOnly: merges new values, keeps extra key, deduplicates
				mergedAttrs := map[string][]string{
					"dept":  {"existing-val", "engineering"},
					"extra": {"kept"},
				}
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To("testuser"),
					Enabled:         ptr.To(false),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To(""),
					Attributes:      &mergedAttrs,
				}).Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsers := v2mocks.NewMockUsersClient(t)
			tt.mockSetup(mockUsers)

			h := NewCreateOrUpdateUser(nil, &keycloakapi.APIClient{Users: mockUsers})
			userCtx := &UserContext{}

			err := h.Serve(
				context.Background(),
				tt.user,
				"test-realm",
				userCtx,
			)

			tt.wantErr(t, err)

			if err == nil && tt.expectedID != "" {
				assert.Equal(t, tt.expectedID, userCtx.UserID)
			}
		})
	}
}
