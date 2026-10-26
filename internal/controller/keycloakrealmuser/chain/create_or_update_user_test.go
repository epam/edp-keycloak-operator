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

func enabledUser(enabled *bool) *keycloakApi.KeycloakRealmUser {
	return &keycloakApi.KeycloakRealmUser{
		ObjectMeta: metav1.ObjectMeta{Name: testUserCRName, Namespace: defaultNamespace},
		Spec:       keycloakApi.KeycloakRealmUserSpec{Username: testUsername, Enabled: enabled},
	}
}

func createRep(enabled *bool) keycloakapi.UserRepresentation {
	return keycloakapi.UserRepresentation{
		Username:        ptr.To(testUsername),
		Enabled:         enabled,
		EmailVerified:   ptr.To(false),
		FirstName:       ptr.To(""),
		LastName:        ptr.To(""),
		RequiredActions: ptr.To([]string(nil)),
		Email:           ptr.To(""),
	}
}

func createdResponse(userID string) *keycloakapi.Response {
	return &keycloakapi.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{"Location": []string{"http://keycloak/users/" + userID}},
		},
	}
}

func existingRep(enabled *bool) *keycloakapi.UserRepresentation {
	return &keycloakapi.UserRepresentation{
		Id:       ptr.To("existing-user-id"),
		Username: ptr.To(testUsername),
		Enabled:  enabled,
	}
}

func updateRep(enabled *bool) *keycloakapi.UserRepresentation {
	rep := createRep(enabled)
	rep.Id = ptr.To("existing-user-id")

	return &rep
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
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:      testUsername,
					Email:         "test@example.com",
					FirstName:     "Test",
					LastName:      "User",
					Enabled:       ptr.To(true),
					EmailVerified: true,
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", keycloakapi.UserRepresentation{
					Username:        ptr.To(testUsername),
					Enabled:         ptr.To(true),
					EmailVerified:   ptr.To(true),
					FirstName:       ptr.To("Test"),
					LastName:        ptr.To("User"),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To("test@example.com"),
				}).Return(createdResponse("user-123"), nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-123",
		},
		{
			name: "success - create user with required actions",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:            testUsername,
					RequiredUserActions: []string{"UPDATE_PASSWORD", "VERIFY_EMAIL"},
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", keycloakapi.UserRepresentation{
					Username:        ptr.To(testUsername),
					Enabled:         ptr.To(true),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"}),
					Email:           ptr.To(""),
				}).Return(createdResponse("user-456"), nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-456",
		},
		{
			name: "success - update existing user",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:      testUsername,
					Email:         "updated@example.com",
					FirstName:     "Updated",
					LastName:      "User",
					Enabled:       ptr.To(true),
					EmailVerified: true,
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				existingUser := &keycloakapi.UserRepresentation{
					Id:       ptr.To("existing-user-id"),
					Username: ptr.To(testUsername),
					Email:    ptr.To("old@example.com"),
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingUser, nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To(testUsername),
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
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:            testUsername,
					RequiredUserActions: []string{"VERIFY_EMAIL"},
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				existingUser := &keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To(testUsername),
					RequiredActions: ptr.To([]string{"UPDATE_PASSWORD", "VERIFY_EMAIL"}),
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingUser, nil, nil)
				// UPDATE_PASSWORD should be preserved
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To(testUsername),
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
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:               testUsername,
					ReconciliationStrategy: keycloakApi.ReconciliationStrategyAddOnly,
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", createRep(ptr.To(true))).Return(createdResponse("user-add-only"), nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-add-only",
		},
		{
			name: "error - FindUserByUsername fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
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
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", createRep(ptr.To(true))).Return(nil, errors.New("keycloak api error"))
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
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				existingUser := &keycloakapi.UserRepresentation{
					Id:       ptr.To("existing-user-id"),
					Username: ptr.To(testUsername),
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingUser, nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", *updateRep(nil)).Return(nil, errors.New("keycloak api error"))
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
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", createRep(ptr.To(true))).Return(&keycloakapi.Response{
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
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:     testUsername,
					AttributesV2: map[string][]string{"dept": {"engineering"}},
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(nil, nil, keycloakapi.ErrNotFound)
				attrs := map[string][]string{"dept": {"engineering"}}
				m.EXPECT().CreateUser(context.Background(), "test-realm", keycloakapi.UserRepresentation{
					Username:        ptr.To(testUsername),
					Enabled:         ptr.To(true),
					EmailVerified:   ptr.To(false),
					FirstName:       ptr.To(""),
					LastName:        ptr.To(""),
					RequiredActions: ptr.To([]string(nil)),
					Email:           ptr.To(""),
					Attributes:      &attrs,
				}).Return(createdResponse("user-attr-123"), nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-attr-123",
		},
		{
			name: "success - update user with attributes (non-addOnly removes stale keys)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:     testUsername,
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
					Username:   ptr.To(testUsername),
					Attributes: &existingAttrs,
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingUser, nil, nil)
				mergedAttrs := map[string][]string{"dept": {"engineering"}}
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To(testUsername),
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
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:               testUsername,
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
					Username:   ptr.To(testUsername),
					Attributes: &existingAttrs,
				}
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingUser, nil, nil)
				// addOnly: merges new values, keeps extra key, deduplicates
				mergedAttrs := map[string][]string{
					"dept":  {"existing-val", "engineering"},
					"extra": {"kept"},
				}
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", keycloakapi.UserRepresentation{
					Id:              ptr.To("existing-user-id"),
					Username:        ptr.To(testUsername),
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
			name: "create - enabled omitted is sent as true",
			user: enabledUser(nil),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", createRep(ptr.To(true))).
					Return(createdResponse("user-h1"), nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-h1",
		},
		{
			name: "create - enabled false is sent as false",
			user: enabledUser(ptr.To(false)),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", createRep(ptr.To(false))).
					Return(createdResponse("user-h2"), nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-h2",
		},
		{
			name: "create - enabled true is sent as true",
			user: enabledUser(ptr.To(true)),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(nil, nil, keycloakapi.ErrNotFound)
				m.EXPECT().CreateUser(context.Background(), "test-realm", createRep(ptr.To(true))).
					Return(createdResponse("user-h3"), nil)
			},
			wantErr:    require.NoError,
			expectedID: "user-h3",
		},
		{
			name: "update - enabled omitted sends no enabled when fetched true",
			user: enabledUser(nil),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingRep(ptr.To(true)), nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", *updateRep(nil)).
					Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
		{
			name: "update - enabled omitted sends no enabled when fetched false",
			user: enabledUser(nil),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingRep(ptr.To(false)), nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", *updateRep(nil)).
					Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
		{
			name: "update - enabled true sends no enabled when fetched true",
			user: enabledUser(ptr.To(true)),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingRep(ptr.To(true)), nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", *updateRep(nil)).
					Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
		{
			name: "update - enabled false is sent when fetched nil",
			user: enabledUser(ptr.To(false)),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingRep(nil), nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", *updateRep(ptr.To(false))).
					Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
		{
			name: "update - enabled false sends no enabled when fetched false",
			user: enabledUser(ptr.To(false)),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingRep(ptr.To(false)), nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", *updateRep(nil)).
					Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
		{
			name: "update - enabled false overrides fetched true",
			user: enabledUser(ptr.To(false)),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingRep(ptr.To(true)), nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", *updateRep(ptr.To(false))).
					Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
		{
			name: "update - enabled true overrides fetched false",
			user: enabledUser(ptr.To(true)),
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().FindUserByUsername(context.Background(), "test-realm", testUsername).
					Return(existingRep(ptr.To(false)), nil, nil)
				m.EXPECT().UpdateUser(context.Background(), "test-realm", "existing-user-id", *updateRep(ptr.To(true))).
					Return(nil, nil)
			},
			wantErr:    require.NoError,
			expectedID: "existing-user-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockUsers := v2mocks.NewMockUsersClient(t)
			tt.mockSetup(mockUsers)

			h := NewCreateOrUpdateUser(nil, &keycloakapi.KeycloakClient{Users: mockUsers})
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
