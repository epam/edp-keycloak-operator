package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const testPasswordSecretName = "user-password"

func TestNewSetUserPassword(t *testing.T) {
	h := NewSetUserPassword(nil, nil)
	require.NotNil(t, h)
}

func TestSetUserPassword_Serve(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	pwSecret := func(rv string, data map[string]string) *corev1.Secret {
		sec := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: testPasswordSecretName, Namespace: defaultNamespace, ResourceVersion: rv},
			Data:       map[string][]byte{},
		}
		for k, v := range data {
			sec.Data[k] = []byte(v)
		}

		return sec
	}
	srcHash := func(sec *corev1.Secret, key string, temporary bool) string {
		return passwordSourceHash(sec, &keycloakApi.PasswordSecret{Name: testPasswordSecretName, Key: key, Temporary: temporary}, "test-realm", "user-123")
	}
	pwUser := func(gen int64, key string, temporary bool, status keycloakApi.KeycloakRealmUserStatus) *keycloakApi.KeycloakRealmUser {
		return &keycloakApi.KeycloakRealmUser{
			ObjectMeta: metav1.ObjectMeta{Name: testUserCRName, Namespace: defaultNamespace, Generation: gen},
			Spec: keycloakApi.KeycloakRealmUserSpec{
				Username:       testUsername,
				PasswordSecret: &keycloakApi.PasswordSecret{Name: testPasswordSecretName, Key: key, Temporary: temporary},
			},
			Status: status,
		}
	}
	synced := func(reason string, gen int64) []metav1.Condition {
		return []metav1.Condition{{Type: ConditionPasswordSynced, Status: metav1.ConditionTrue, Reason: reason, ObservedGeneration: gen}}
	}
	failed := func(reason string) []metav1.Condition {
		return []metav1.Condition{{Type: ConditionPasswordSynced, Status: metav1.ConditionFalse, Reason: reason}}
	}
	expectSet := func(value string, temporary bool, err error) func(*v2mocks.MockUsersClient) {
		return func(m *v2mocks.MockUsersClient) {
			m.EXPECT().SetUserPassword(
				context.Background(),
				"test-realm",
				"user-123",
				keycloakapi.CredentialRepresentation{
					Type:      ptr.To("password"),
					Value:     ptr.To(value),
					Temporary: ptr.To(temporary),
				},
			).Return(nil, err)
		}
	}
	ab := map[string]string{"a": "value-a", "b": "value-b"}

	tests := []struct {
		name                    string
		user                    *keycloakApi.KeycloakRealmUser
		secret                  *corev1.Secret
		mockSetup               func(*v2mocks.MockUsersClient)
		wantErr                 require.ErrorAssertionFunc
		wantCondition           bool
		wantConditionStatus     metav1.ConditionStatus
		wantConditionMsg        string
		wantConditionReason     string
		wantStatusSecretVersion string
		wantPasswordSecretHash  string
		wantSkipSetPwd          bool
	}{
		{
			name: "success - no password configured",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
				},
			},
			mockSetup:     func(m *v2mocks.MockUsersClient) {},
			wantErr:       require.NoError,
			wantCondition: false,
		},
		{
			name: "success - set password from secret (non-temporary)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       testUserCRName,
					Namespace:  defaultNamespace,
					Generation: 1,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      testPasswordSecretName,
						Key:       "password",
						Temporary: false,
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testPasswordSecretName,
					Namespace:       defaultNamespace,
					ResourceVersion: "12345",
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().SetUserPassword(
					context.Background(),
					"test-realm",
					"user-123",
					keycloakapi.CredentialRepresentation{
						Type:      ptr.To("password"),
						Value:     ptr.To("secret-password"),
						Temporary: ptr.To(false),
					},
				).Return(nil, nil)
			},
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionMsg:        "Password synced from secret user-password",
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "12345",
			wantPasswordSecretHash:  srcHash(pwSecret("12345", nil), "password", false),
		},
		{
			name: "success - set temporary password from secret",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       testUserCRName,
					Namespace:  defaultNamespace,
					Generation: 1,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      testPasswordSecretName,
						Key:       "password",
						Temporary: true,
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testPasswordSecretName,
					Namespace:       defaultNamespace,
					ResourceVersion: "12345",
				},
				Data: map[string][]byte{
					"password": []byte("temp-password"),
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().SetUserPassword(
					context.Background(),
					"test-realm",
					"user-123",
					keycloakapi.CredentialRepresentation{
						Type:      ptr.To("password"),
						Value:     ptr.To("temp-password"),
						Temporary: ptr.To(true),
					},
				).Return(nil, nil)
			},
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionMsg:        "Temporary password set from secret user-password (written once per source)",
			wantConditionReason:     ReasonTemporaryPasswordSet,
			wantStatusSecretVersion: "12345",
			wantPasswordSecretHash:  srcHash(pwSecret("12345", nil), "password", true),
		},
		{
			name: "skip - temporary password already synced",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      testPasswordSecretName,
						Key:       "password",
						Temporary: true,
					},
				},
				Status: keycloakApi.KeycloakRealmUserStatus{
					PasswordSecretHash: srcHash(pwSecret("12345", nil), "password", true),
					Conditions: []metav1.Condition{
						{
							Type:   ConditionPasswordSynced,
							Status: metav1.ConditionTrue,
							Reason: ReasonTemporaryPasswordSet,
						},
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testPasswordSecretName,
					Namespace:       defaultNamespace,
					ResourceVersion: "12345",
				},
				Data: map[string][]byte{
					"password": []byte("temp-password"),
				},
			},
			mockSetup:              func(m *v2mocks.MockUsersClient) {},
			wantErr:                require.NoError,
			wantSkipSetPwd:         true,
			wantCondition:          true,
			wantConditionStatus:    metav1.ConditionTrue,
			wantConditionReason:    ReasonTemporaryPasswordSet,
			wantPasswordSecretHash: srcHash(pwSecret("12345", nil), "password", true),
		},
		{
			name: "skip - non-temporary password secret unchanged",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testUserCRName,
					Namespace: defaultNamespace,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      testPasswordSecretName,
						Key:       "password",
						Temporary: false,
					},
				},
				Status: keycloakApi.KeycloakRealmUserStatus{
					LastSyncedPasswordSecretVersion: "12345",
					PasswordSecretHash:              srcHash(pwSecret("12345", nil), "password", false),
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
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testPasswordSecretName,
					Namespace:       defaultNamespace,
					ResourceVersion: "12345",
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			},
			mockSetup:               func(m *v2mocks.MockUsersClient) {},
			wantErr:                 require.NoError,
			wantSkipSetPwd:          true,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "12345",
			wantPasswordSecretHash:  srcHash(pwSecret("12345", nil), "password", false),
		},
		{
			name: "success - non-temporary password secret changed",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       testUserCRName,
					Namespace:  defaultNamespace,
					Generation: 1,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      testPasswordSecretName,
						Key:       "password",
						Temporary: false,
					},
				},
				Status: keycloakApi.KeycloakRealmUserStatus{
					LastSyncedPasswordSecretVersion: "12345",
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
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testPasswordSecretName,
					Namespace:       defaultNamespace,
					ResourceVersion: "67890",
				},
				Data: map[string][]byte{
					"password": []byte("new-password"),
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().SetUserPassword(
					context.Background(),
					"test-realm",
					"user-123",
					keycloakapi.CredentialRepresentation{
						Type:      ptr.To("password"),
						Value:     ptr.To("new-password"),
						Temporary: ptr.To(false),
					},
				).Return(nil, nil)
			},
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionMsg:        "Password synced from secret user-password",
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "67890",
			wantPasswordSecretHash:  srcHash(pwSecret("67890", nil), "password", false),
		},
		{
			name: "success - set password from spec (deprecated)",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       testUserCRName,
					Namespace:  defaultNamespace,
					Generation: 1,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
					Password: "inline-password",
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().SetUserPassword(
					context.Background(),
					"test-realm",
					"user-123",
					keycloakapi.CredentialRepresentation{
						Type:      ptr.To("password"),
						Value:     ptr.To("inline-password"),
						Temporary: ptr.To(false),
					},
				).Return(nil, nil)
			},
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionMsg:        "Password set from spec.password field (deprecated)",
			wantConditionReason:     ReasonPasswordSetFromSpec,
			wantStatusSecretVersion: "",
		},
		{
			name: "error - passwordSecret without key",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: testUserCRName, Namespace: defaultNamespace},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:       testUsername,
					PasswordSecret: &keycloakApi.PasswordSecret{Name: "half-ref"},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "half-ref", Namespace: defaultNamespace},
				Data:       map[string][]byte{"pwd": []byte("secret-password")},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.ErrorIs(t, err, errSecretKeyNotFound)
			},
			wantCondition:       true,
			wantConditionStatus: metav1.ConditionFalse,
			wantConditionReason: ReasonSecretKeyMissing,
		},
		{
			name: "error - passwordSecret without name",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: testUserCRName, Namespace: defaultNamespace},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username:       testUsername,
					Password:       "inline-password",
					PasswordSecret: &keycloakApi.PasswordSecret{Key: "pwd"},
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {},
			wantErr: func(t require.TestingT, err error, _ ...any) {
				require.True(t, k8serrors.IsNotFound(err), "expected NotFound, got %v", err)
			},
			wantCondition:       true,
			wantConditionStatus: metav1.ConditionFalse,
			wantConditionReason: ReasonSecretNotFound,
		},
		{
			name: "error - secret not found",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:       testUserCRName,
					Namespace:  defaultNamespace,
					Generation: 5,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name: "missing-secret",
						Key:  "password",
					},
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {},
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
					Name:       testUserCRName,
					Namespace:  defaultNamespace,
					Generation: 5,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name: testPasswordSecretName,
						Key:  "wrong-key",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testPasswordSecretName,
					Namespace: defaultNamespace,
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {},
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
					Name:       testUserCRName,
					Namespace:  defaultNamespace,
					Generation: 5,
				},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Username: testUsername,
					PasswordSecret: &keycloakApi.PasswordSecret{
						Name:      testPasswordSecretName,
						Key:       "password",
						Temporary: false,
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:            testPasswordSecretName,
					Namespace:       defaultNamespace,
					ResourceVersion: "12345",
				},
				Data: map[string][]byte{
					"password": []byte("secret-password"),
				},
			},
			mockSetup: func(m *v2mocks.MockUsersClient) {
				m.EXPECT().SetUserPassword(
					context.Background(),
					"test-realm",
					"user-123",
					keycloakapi.CredentialRepresentation{
						Type:      ptr.To("password"),
						Value:     ptr.To("secret-password"),
						Temporary: ptr.To(false),
					},
				).Return(nil, errors.New("keycloak error"))
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
		{
			name: "non-temporary key change writes the new key",
			user: pwUser(2, "b", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", false),
				Conditions:                      synced(ReasonPasswordSetFromSecret, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-b", false, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "b", false),
		},
		{
			name: "temporary key change writes the new key",
			user: pwUser(2, "b", true, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", true),
				Conditions:                      synced(ReasonTemporaryPasswordSet, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-b", true, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonTemporaryPasswordSet,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "b", true),
		},
		{
			name: "non-temporary to temporary writes with temporary=true",
			user: pwUser(2, "a", true, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", false),
				Conditions:                      synced(ReasonPasswordSetFromSecret, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-a", true, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonTemporaryPasswordSet,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", true),
		},
		{
			name: "temporary to non-temporary writes with temporary=false",
			user: pwUser(2, "a", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", true),
				Conditions:                      synced(ReasonTemporaryPasswordSet, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-a", false, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", false),
		},
		{
			name: "skip - temporary password ignores Secret rotation",
			user: pwUser(1, "a", true, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", true),
				Conditions:                      synced(ReasonTemporaryPasswordSet, 1),
			}),
			secret:                  pwSecret("8", ab),
			mockSetup:               func(m *v2mocks.MockUsersClient) {},
			wantErr:                 require.NoError,
			wantSkipSetPwd:          true,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonTemporaryPasswordSet,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", true),
		},
		{
			name: "skip - generation bump refreshes observedGeneration",
			user: pwUser(2, "a", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", false),
				Conditions:                      synced(ReasonPasswordSetFromSecret, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               func(m *v2mocks.MockUsersClient) {},
			wantErr:                 require.NoError,
			wantSkipSetPwd:          true,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", false),
		},
		{
			name: "target user change writes the password",
			user: pwUser(1, "a", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash: passwordSourceHash(pwSecret("7", ab),
					&keycloakApi.PasswordSecret{Name: testPasswordSecretName, Key: "a"}, "test-realm", "old-user"),
				Conditions: synced(ReasonPasswordSetFromSecret, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-a", false, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", false),
		},
		{
			name: "temporary error fixed with unchanged source does not rewrite",
			user: pwUser(3, "a", true, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", true),
				Conditions:                      failed(ReasonSecretKeyMissing),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               func(m *v2mocks.MockUsersClient) {},
			wantErr:                 require.NoError,
			wantSkipSetPwd:          true,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonTemporaryPasswordSet,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", true),
		},
		{
			name: "non-temporary error fixed with unchanged source sets the condition True",
			user: pwUser(3, "a", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", false),
				Conditions:                      failed(ReasonSecretKeyMissing),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               func(m *v2mocks.MockUsersClient) {},
			wantErr:                 require.NoError,
			wantSkipSetPwd:          true,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", false),
		},
		{
			name: "keycloak error on a changed source keeps the stored hash and version",
			user: pwUser(2, "b", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", false),
				Conditions:                      synced(ReasonPasswordSetFromSecret, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-b", false, errors.New("keycloak error")),
			wantErr:                 require.Error,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionFalse,
			wantConditionReason:     ReasonKeycloakAPIError,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", false),
		},
		{
			name: "spec.password clears the stored hash and version",
			user: &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: testUserCRName, Namespace: defaultNamespace, Generation: 2},
				Spec:       keycloakApi.KeycloakRealmUserSpec{Username: testUsername, Password: "inline-password"},
				Status: keycloakApi.KeycloakRealmUserStatus{
					LastSyncedPasswordSecretVersion: "7",
					PasswordSecretHash:              srcHash(pwSecret("7", ab), "a", false),
					Conditions:                      synced(ReasonPasswordSetFromSecret, 1),
				},
			},
			mockSetup:           expectSet("inline-password", false, nil),
			wantErr:             require.NoError,
			wantCondition:       true,
			wantConditionStatus: metav1.ConditionTrue,
			wantConditionReason: ReasonPasswordSetFromSpec,
		},
		{
			name: "temporary passwordSecret after spec.password writes with temporary=true",
			user: pwUser(2, "a", true, keycloakApi.KeycloakRealmUserStatus{
				Conditions: synced(ReasonPasswordSetFromSpec, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-a", true, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonTemporaryPasswordSet,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", true),
		},
		{
			name: "adopt - non-temporary synced status without hash",
			user: pwUser(1, "a", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				Conditions:                      synced(ReasonPasswordSetFromSecret, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               func(m *v2mocks.MockUsersClient) {},
			wantErr:                 require.NoError,
			wantSkipSetPwd:          true,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", false),
		},
		{
			name: "write - non-temporary without hash and Secret rotated",
			user: pwUser(1, "a", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "6",
				Conditions:                      synced(ReasonPasswordSetFromSecret, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-a", false, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", false),
		},
		{
			name: "adopt - temporary synced status without hash",
			user: pwUser(1, "a", true, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				Conditions:                      synced(ReasonTemporaryPasswordSet, 1),
			}),
			secret:                  pwSecret("8", ab),
			mockSetup:               func(m *v2mocks.MockUsersClient) {},
			wantErr:                 require.NoError,
			wantSkipSetPwd:          true,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonTemporaryPasswordSet,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("8", ab), "a", true),
		},
		{
			name: "write - temporary spec after non-temporary status without hash",
			user: pwUser(1, "a", true, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				Conditions:                      synced(ReasonPasswordSetFromSecret, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-a", true, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonTemporaryPasswordSet,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", true),
		},
		{
			name: "write - non-temporary spec after temporary status without hash",
			user: pwUser(1, "a", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				Conditions:                      synced(ReasonTemporaryPasswordSet, 1),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-a", false, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", false),
		},
		{
			name: "write - non-temporary without hash and unsettled condition",
			user: pwUser(1, "a", false, keycloakApi.KeycloakRealmUserStatus{
				LastSyncedPasswordSecretVersion: "7",
				Conditions:                      failed(ReasonSecretKeyMissing),
			}),
			secret:                  pwSecret("7", ab),
			mockSetup:               expectSet("value-a", false, nil),
			wantErr:                 require.NoError,
			wantCondition:           true,
			wantConditionStatus:     metav1.ConditionTrue,
			wantConditionReason:     ReasonPasswordSetFromSecret,
			wantStatusSecretVersion: "7",
			wantPasswordSecretHash:  srcHash(pwSecret("7", ab), "a", false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.user).WithStatusSubresource(tt.user)
			if tt.secret != nil {
				builder = builder.WithObjects(tt.secret)
			}

			k8sClient := builder.Build()
			mockUsers := v2mocks.NewMockUsersClient(t)
			tt.mockSetup(mockUsers)

			h := NewSetUserPassword(k8sClient, &keycloakapi.KeycloakClient{Users: mockUsers})
			userCtx := &UserContext{UserID: "user-123"}

			err := h.Serve(
				context.Background(),
				tt.user,
				"test-realm",
				userCtx,
			)

			tt.wantErr(t, err)

			if tt.wantSkipSetPwd {
				mockUsers.AssertNotCalled(t, "SetUserPassword", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}

			if tt.wantCondition {
				condition := meta.FindStatusCondition(tt.user.Status.Conditions, ConditionPasswordSynced)
				require.NotNil(t, condition, "expected PasswordSynced condition to be set")
				assert.Equal(t, tt.wantConditionStatus, condition.Status, "condition status mismatch")
				assert.Equal(t, tt.wantConditionReason, condition.Reason, "condition reason mismatch")

				if tt.wantConditionMsg != "" {
					assert.Contains(t, condition.Message, tt.wantConditionMsg, "condition message mismatch")
				}

				assert.Equal(t, tt.user.Generation, condition.ObservedGeneration)
			}

			assert.Equal(t, tt.wantStatusSecretVersion, tt.user.Status.LastSyncedPasswordSecretVersion)
			assert.Equal(t, tt.wantPasswordSecretHash, tt.user.Status.PasswordSecretHash)
		})
	}
}

func TestSetUserPassword_Serve_AdoptsTemporaryBeforeSecretRead(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	user := &keycloakApi.KeycloakRealmUser{
		ObjectMeta: metav1.ObjectMeta{Name: testUserCRName, Namespace: defaultNamespace, Generation: 1},
		Spec: keycloakApi.KeycloakRealmUserSpec{
			Username:       testUsername,
			PasswordSecret: &keycloakApi.PasswordSecret{Name: testPasswordSecretName, Key: "a", Temporary: true},
		},
		Status: keycloakApi.KeycloakRealmUserStatus{
			LastSyncedPasswordSecretVersion: "7",
			Conditions: []metav1.Condition{
				{Type: ConditionPasswordSynced, Status: metav1.ConditionTrue, Reason: ReasonTemporaryPasswordSet, ObservedGeneration: 1},
			},
		},
	}
	wantHash := passwordSourceHash(nil, user.Spec.PasswordSecret, "test-realm", "user-123")

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(user).Build()
	mockUsers := v2mocks.NewMockUsersClient(t)
	h := NewSetUserPassword(k8sClient, &keycloakapi.KeycloakClient{Users: mockUsers})
	userCtx := &UserContext{UserID: "user-123"}

	err := h.Serve(context.Background(), user, "test-realm", userCtx)
	require.Error(t, err)

	cond := meta.FindStatusCondition(user.Status.Conditions, ConditionPasswordSynced)
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Equal(t, ReasonSecretNotFound, cond.Reason)
	assert.Equal(t, wantHash, user.Status.PasswordSecretHash)

	require.NoError(t, k8sClient.Create(context.Background(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testPasswordSecretName, Namespace: defaultNamespace},
		Data:       map[string][]byte{"a": []byte("value-a")},
	}))

	require.NoError(t, h.Serve(context.Background(), user, "test-realm", userCtx))

	cond = meta.FindStatusCondition(user.Status.Conditions, ConditionPasswordSynced)
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, ReasonTemporaryPasswordSet, cond.Reason)
	assert.Equal(t, wantHash, user.Status.PasswordSecretHash)
}

func TestPasswordSourceHash(t *testing.T) {
	secret := func(name, uid, rv string, data string) *corev1.Secret {
		return &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(uid), ResourceVersion: rv},
			Data:       map[string][]byte{"k": []byte(data)},
		}
	}
	ps := func(name, key string, temporary bool) *keycloakApi.PasswordSecret {
		return &keycloakApi.PasswordSecret{Name: name, Key: key, Temporary: temporary}
	}
	hash := func(sec *corev1.Secret, p *keycloakApi.PasswordSecret, realm, userID string) string {
		return passwordSourceHash(sec, p, realm, userID)
	}

	base := secret("s", "uid-1", "1", "v")

	for _, temporary := range []bool{false, true} {
		want := hash(base, ps("s", "k", temporary), "r", "u")

		assert.Equal(t, want, hash(base, ps("s", "k", temporary), "r", "u"), "stable")
		assert.NotEqual(t, want, hash(secret("s2", "uid-1", "1", "v"), ps("s2", "k", temporary), "r", "u"), "name")
		assert.NotEqual(t, want, hash(base, ps("s", "k2", temporary), "r", "u"), "key")
		assert.NotEqual(t, want, hash(base, ps("s", "k", !temporary), "r", "u"), "temporary flag")
		assert.NotEqual(t, want, hash(base, ps("s", "k", temporary), "r2", "u"), "realm")
		assert.NotEqual(t, want, hash(base, ps("s", "k", temporary), "r", "u2"), "user ID")
		assert.Equal(t, want, hash(secret("s", "uid-1", "1", "other"), ps("s", "k", temporary), "r", "u"), "Secret data is not an input")
	}

	nonTemp := hash(base, ps("s", "k", false), "r", "u")
	assert.NotEqual(t, nonTemp, hash(secret("s", "uid-1", "2", "v"), ps("s", "k", false), "r", "u"), "non-temporary: resourceVersion")
	assert.NotEqual(t, nonTemp, hash(secret("s", "uid-2", "1", "v"), ps("s", "k", false), "r", "u"), "non-temporary: uid")

	temp := hash(base, ps("s", "k", true), "r", "u")
	assert.Equal(t, temp, hash(secret("s", "uid-2", "2", "v"), ps("s", "k", true), "r", "u"), "temporary: ignores uid and resourceVersion")
}
