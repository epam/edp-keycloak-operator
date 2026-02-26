package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	keycloakmocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestNewSetUserPassword(t *testing.T) {
	h := NewSetUserPassword(nil)
	require.NotNil(t, h)
}

func TestSetUserPassword_Serve(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	tests := []struct {
		name                    string
		user                    *keycloakApi.KeycloakRealmUser
		realm                   *gocloak.RealmRepresentation
		secret                  *corev1.Secret
		mockSetup               func(*keycloakmocks.MockClient)
		wantErr                 require.ErrorAssertionFunc
		wantCondition           bool
		wantConditionStatus     metav1.ConditionStatus
		wantConditionMsg        string
		wantConditionReason     string
		wantStatusSecretVersion string
		wantSkipSetPwd          bool
	}{
		{
			name: "success - no password configured",
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
			mockSetup:     func(m *keycloakmocks.MockClient) {},
			wantErr:       require.NoError,
			wantCondition: false,
		},
		{
			name: "success - set password from secret (non-temporary)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-user",
					Namespace:  "default",
					Generation: 1,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      "user-password",
						Key:       "password",
						Temporary: false,
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "user-password",
					Namespace:       "default",
					ResourceVersion: "12345",
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SetUserPassword(
					"test-realm",
					"user-123",
					&adapter.KeycloakUserPassword{
						Value:     "secret-password",
						Temporary: false,
					},
				).Return(nil)
			},
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionMsg:        "Password synced from secret user-password",
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "12345",
		},
		{
			name: "success - set temporary password from secret",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-user",
					Namespace:  "default",
					Generation: 1,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      "user-password",
						Key:       "password",
						Temporary: true,
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "user-password",
					Namespace:       "default",
					ResourceVersion: "12345",
				},
				Data: map[string][]byte{
					"password": []byte("temp-password"),
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SetUserPassword(
					"test-realm",
					"user-123",
					&adapter.KeycloakUserPassword{
						Value:     "temp-password",
						Temporary: true,
					},
				).Return(nil)
			},
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionMsg:        "Temporary password set from secret user-password (will not reset)",
			wantConditionReason:     ReasonTemporaryPasswordSet,
			wantStatusSecretVersion: "12345",
		},
		{
			name: "skip - temporary password already synced",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      "user-password",
						Key:       "password",
						Temporary: true,
					},
				},
				Status: keycloakApi.KeycloakRealmUserStatus{
					Conditions: []metav1.Condition{
						{
							Type:   ConditionPasswordSynced,
							Status: metav1.ConditionTrue,
							Reason: ReasonTemporaryPasswordSet,
						},
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "user-password",
					Namespace:       "default",
					ResourceVersion: "12345",
				},
				Data: map[string][]byte{
					"password": []byte("temp-password"),
				},
			},
			mockSetup:      func(m *keycloakmocks.MockClient) {},
			wantErr:        require.NoError,
			wantSkipSetPwd: true,
		},
		{
			name: "skip - non-temporary password secret unchanged",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      "user-password",
						Key:       "password",
						Temporary: false,
					},
				},
				Status: keycloakApi.KeycloakRealmUserStatus{
					LastSyncedPasswordSecretVersion: "12345", // same resourceVersion
					Conditions: []metav1.Condition{
						{
							Type:    ConditionPasswordSynced,
							Status:  metav1.ConditionTrue,
							Reason:  ReasonPasswordSetFromSecret,
							Message: "Password synced from secret user-password",
						},
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "user-password",
					Namespace:       "default",
					ResourceVersion: "12345", // unchanged
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			},
			mockSetup:      func(m *keycloakmocks.MockClient) {},
			wantErr:        require.NoError,
			wantSkipSetPwd: true,
		},
		{
			name: "success - non-temporary password secret changed",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-user",
					Namespace:  "default",
					Generation: 1,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      "user-password",
						Key:       "password",
						Temporary: false,
					},
				},
				Status: keycloakApi.KeycloakRealmUserStatus{
					LastSyncedPasswordSecretVersion: "12345", // old resourceVersion
					Conditions: []metav1.Condition{
						{
							Type:    ConditionPasswordSynced,
							Status:  metav1.ConditionTrue,
							Reason:  ReasonPasswordSetFromSecret,
							Message: "Password synced from secret user-password",
						},
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "user-password",
					Namespace:       "default",
					ResourceVersion: "67890", // changed!
				},
				Data: map[string][]byte{
					"password": []byte("new-password"),
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SetUserPassword(
					"test-realm",
					"user-123",
					&adapter.KeycloakUserPassword{
						Value:     "new-password",
						Temporary: false,
					},
				).Return(nil)
			},
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionMsg:        "Password synced from secret user-password",
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "67890",
		},
		{
			name: "success - set password from spec (deprecated)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-user",
					Namespace:  "default",
					Generation: 1,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					Password: "inline-password",
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SetUserPassword(
					"test-realm",
					"user-123",
					&adapter.KeycloakUserPassword{
						Value:     "inline-password",
						Temporary: false,
					},
				).Return(nil)
			},
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionMsg:        "Password set from spec.password field (deprecated)",
			wantConditionReason:     ReasonPasswordSetFromSpec,
			wantStatusSecretVersion: "",
		},
		{
			name: "error - secret not found",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-user",
					Namespace:  "default",
					Generation: 5,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name: "missing-secret",
						Key:  "password",
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			mockSetup: func(m *keycloakmocks.MockClient) {},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to get secret")
			},
			wantCondition:       true,
			wantConditionStatus: metav1.ConditionFalse,
			wantConditionReason: ReasonSecretNotFound,
			wantConditionMsg:    "Password secret",
		},
		{
			name: "error - key not found in secret",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-user",
					Namespace:  "default",
					Generation: 5,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name: "user-password",
						Key:  "wrong-key",
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-password",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "key wrong-key not found in secret")
			},
			wantCondition:       true,
			wantConditionStatus: metav1.ConditionFalse,
			wantConditionReason: ReasonSecretKeyMissing,
			wantConditionMsg:    "Key",
		},
		{
			name: "error - keycloak SetUserPassword fails",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-user",
					Namespace:  "default",
					Generation: 5,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: "testuser",
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      "user-password",
						Key:       "password",
						Temporary: false,
					},
				},
			},
			realm: &gocloak.RealmRepresentation{
				Realm: gocloak.StringP("test-realm"),
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "user-password",
					Namespace:       "default",
					ResourceVersion: "12345",
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			},
			mockSetup: func(m *keycloakmocks.MockClient) {
				m.EXPECT().SetUserPassword(
					"test-realm",
					"user-123",
					&adapter.KeycloakUserPassword{
						Value:     "secret-password",
						Temporary: false,
					},
				).Return(errors.New("keycloak error"))
			},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unable to set user password")
			},
			wantCondition:       true,
			wantConditionStatus: metav1.ConditionFalse,
			wantConditionReason: ReasonKeycloakAPIError,
			wantConditionMsg:    "Failed to set password in Keycloak",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.user).WithStatusSubresource(tt.user)
			if tt.secret != nil {
				builder = builder.WithObjects(tt.secret)
			}

			k8sClient := builder.Build()
			mockClient := keycloakmocks.NewMockClient(t)
			tt.mockSetup(mockClient)

			h := NewSetUserPassword(k8sClient)
			userCtx := &UserContext{UserID: "user-123"}

			err := h.Serve(context.Background(), tt.user, mockClient, tt.realm, userCtx)

			tt.wantErr(t, err)

			if tt.wantCondition {
				condition := meta.FindStatusCondition(tt.user.Status.Conditions, ConditionPasswordSynced)
				require.NotNil(t, condition, "expected PasswordSynced condition to be set")
				assert.Equal(t, tt.wantConditionStatus, condition.Status, "condition status mismatch")
				assert.Equal(t, tt.wantConditionReason, condition.Reason, "condition reason mismatch")

				if tt.wantConditionMsg != "" {
					assert.Contains(t, condition.Message, tt.wantConditionMsg, "condition message mismatch")
				}

				assert.Equal(t, tt.user.Generation, condition.ObservedGeneration)

				if tt.wantConditionStatus == metav1.ConditionTrue {
					assert.Equal(t, tt.wantStatusSecretVersion, tt.user.Status.LastSyncedPasswordSecretVersion)
				}
			}
		})
	}
}
