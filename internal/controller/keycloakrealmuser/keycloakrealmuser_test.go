package keycloakrealmuser

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	helpermock "github.com/epam/edp-keycloak-operator/internal/controller/helper/mocks"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealmuser/chain"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
)

func TestReconcileKeycloakRealmUser_migrateAttributes(t *testing.T) {
	tests := []struct {
		name              string
		keycloakRealmUser *keycloakApi.KeycloakRealmUser
		expectedAttrs     map[string][]string
		shouldMigrate     bool
	}{
		{
			name: "should migrate Attributes to AttributesV2",
			keycloakRealmUser: &keycloakApi.KeycloakRealmUser{
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Attributes: map[string]string{
						"foo": "bar",
					},
				},
			},
			expectedAttrs: map[string][]string{
				"foo": {"bar"},
			},
			shouldMigrate: true,
		},
		{
			name: "should not migrate when AttributesV2 is already populated",
			keycloakRealmUser: &keycloakApi.KeycloakRealmUser{
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Attributes: map[string]string{
						"attr1": "test-value-1",
					},
					AttributesV2: map[string][]string{
						"attr2": {"test-value-2"},
					},
				},
			},
			expectedAttrs: map[string][]string{
				"attr2": {"test-value-2"},
			},
			shouldMigrate: false,
		},
		{
			name: "should not migrate when Attributes is empty",
			keycloakRealmUser: &keycloakApi.KeycloakRealmUser{
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Attributes: map[string]string{},
				},
			},
			expectedAttrs: nil,
			shouldMigrate: false,
		},
		{
			name: "should not migrate when both fields are empty",
			keycloakRealmUser: &keycloakApi.KeycloakRealmUser{
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Attributes:   map[string]string{},
					AttributesV2: map[string][]string{},
				},
			},
			expectedAttrs: map[string][]string{},
			shouldMigrate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a controller instance
			controller := &Reconcile{}

			// Perform migration
			migrated := controller.migrateAttributes(tt.keycloakRealmUser)

			// Assert migration result
			assert.Equal(t, tt.shouldMigrate, migrated)

			// Assert Attributes contains expected roles
			assert.Equal(t, tt.expectedAttrs, tt.keycloakRealmUser.Spec.AttributesV2)

			// If migration occurred, Attributes should remain unchanged for backward compatibility
			if tt.shouldMigrate {
				assert.NotNil(t, tt.keycloakRealmUser.Spec.Attributes)
			}
		})
	}
}

func TestMigratePasswordSecret(t *testing.T) {
	tests := []struct {
		name           string
		passwordSecret *keycloakApi.PasswordSecret
		wantMigrate    bool
		wantSecret     *keycloakApi.PasswordSecret
	}{
		{
			name: "nil",
		},
		{
			name:           "legacy stub",
			passwordSecret: &keycloakApi.PasswordSecret{},
			wantMigrate:    true,
		},
		{
			name:           "empty name and key with temporary",
			passwordSecret: &keycloakApi.PasswordSecret{Temporary: true},
			wantSecret:     &keycloakApi.PasswordSecret{Temporary: true},
		},
		{
			name:           "name only",
			passwordSecret: &keycloakApi.PasswordSecret{Name: "secret"},
			wantSecret:     &keycloakApi.PasswordSecret{Name: "secret"},
		},
		{
			name:           "key only",
			passwordSecret: &keycloakApi.PasswordSecret{Key: "password"},
			wantSecret:     &keycloakApi.PasswordSecret{Key: "password"},
		},
		{
			name:           "valid reference",
			passwordSecret: &keycloakApi.PasswordSecret{Name: "secret", Key: "password"},
			wantSecret:     &keycloakApi.PasswordSecret{Name: "secret", Key: "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &keycloakApi.KeycloakRealmUser{
				Spec: keycloakApi.KeycloakRealmUserSpec{PasswordSecret: tt.passwordSecret},
			}

			assert.Equal(t, tt.wantMigrate, migratePasswordSecret(user))
			assert.Equal(t, tt.wantSecret, user.Spec.PasswordSecret)
		})
	}
}

func TestReconcile_applyDefaults_passwordSecret(t *testing.T) {
	tests := []struct {
		name           string
		passwordSecret *keycloakApi.PasswordSecret
		wantUpdated    bool
		wantSecret     *keycloakApi.PasswordSecret
	}{
		{
			name:           "stub is removed from the stored object",
			passwordSecret: &keycloakApi.PasswordSecret{},
			wantUpdated:    true,
		},
		{
			name:           "valid reference is kept",
			passwordSecret: &keycloakApi.PasswordSecret{Name: "secret", Key: "password"},
			wantSecret:     &keycloakApi.PasswordSecret{Name: "secret", Key: "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, keycloakApi.AddToScheme(scheme))

			user := &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: "default"},
				Spec:       keycloakApi.KeycloakRealmUserSpec{PasswordSecret: tt.passwordSecret},
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(user).Build()

			updated, err := NewReconcile(k8sClient, nil).applyDefaults(context.Background(), user)
			require.NoError(t, err)
			assert.Equal(t, tt.wantUpdated, updated)

			got := &keycloakApi.KeycloakRealmUser{}
			require.NoError(t, k8sClient.Get(context.Background(), types.NamespacedName{Name: "user", Namespace: "default"}, got))
			assert.Equal(t, tt.wantSecret, got.Spec.PasswordSecret)
		})
	}
}

func TestReconcile_applyDefaults_optionalBools(t *testing.T) {
	tests := []struct {
		name              string
		specJSON          string
		wantEnabled       *bool
		wantEmailVerified *bool
	}{
		{
			name:     "absent stays absent",
			specJSON: `{"username":"u","passwordSecret":{"name":"","key":""}}`,
		},
		{
			name:        "enabled false is kept",
			specJSON:    `{"username":"u","enabled":false,"passwordSecret":{"name":"","key":""}}`,
			wantEnabled: ptr.To(false),
		},
		{
			name:        "enabled true is kept",
			specJSON:    `{"username":"u","enabled":true,"passwordSecret":{"name":"","key":""}}`,
			wantEnabled: ptr.To(true),
		},
		{
			name:              "emailVerified false is kept",
			specJSON:          `{"username":"u","emailVerified":false,"passwordSecret":{"name":"","key":""}}`,
			wantEmailVerified: ptr.To(false),
		},
		{
			name:              "emailVerified true is kept",
			specJSON:          `{"username":"u","emailVerified":true,"passwordSecret":{"name":"","key":""}}`,
			wantEmailVerified: ptr.To(true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, keycloakApi.AddToScheme(scheme))

			user := &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Name: "user", Namespace: "default"},
			}
			require.NoError(t, json.Unmarshal([]byte(tt.specJSON), &user.Spec))

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(user).Build()

			updated, err := NewReconcile(k8sClient, nil).applyDefaults(context.Background(), user)
			require.NoError(t, err)
			require.True(t, updated, "the passwordSecret stub must force the full-object update")

			got := &unstructured.Unstructured{}
			got.SetGroupVersionKind(keycloakApi.GroupVersion.WithKind("KeycloakRealmUser"))
			require.NoError(t, k8sClient.Get(context.Background(), types.NamespacedName{Name: "user", Namespace: "default"}, got))

			for field, want := range map[string]*bool{"enabled": tt.wantEnabled, "emailVerified": tt.wantEmailVerified} {
				value, found, err := unstructured.NestedBool(got.Object, "spec", field)
				require.NoError(t, err)
				assert.Equal(t, want != nil, found, field)
				assert.Equal(t, ptr.Deref(want, false), value, field)
			}
		})
	}
}

func TestReconcile_ApplyDefaultsErrorWritesStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		statusFails bool
		runs        int
		wantValue   string
		wantErrMsg  string
		wantWrites  int
	}{
		{
			name:       "update error is written to status",
			wantValue:  "failed to update KeycloakRealmUser default values",
			wantErrMsg: "failed to update KeycloakRealmUser default values",
			wantWrites: 1,
		},
		{
			name:       "same error twice writes once",
			runs:       2,
			wantValue:  "failed to update KeycloakRealmUser default values",
			wantErrMsg: "failed to update KeycloakRealmUser default values",
			wantWrites: 1,
		},
		{
			name:        "status write conflict returns the status error and stores nothing",
			statusFails: true,
			wantValue:   "OK",
			wantErrMsg:  "failed to update KeycloakRealmUser status",
			wantWrites:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			require.NoError(t, keycloakApi.AddToScheme(scheme))

			key := client.ObjectKey{Namespace: "default", Name: "user"}
			user := &keycloakApi.KeycloakRealmUser{
				ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name},
				Spec: keycloakApi.KeycloakRealmUserSpec{
					Attributes: map[string]string{"foo": "bar"},
				},
				Status: keycloakApi.KeycloakRealmUserStatus{Value: "OK"},
			}

			writes := 0

			cl := fake.NewClientBuilder().
				WithScheme(scheme).
				WithStatusSubresource(&keycloakApi.KeycloakRealmUser{}).
				WithObjects(user).
				WithInterceptorFuncs(interceptor.Funcs{
					Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error {
						return k8sErrors.NewInternalError(errors.New("update denied"))
					},
					SubResourceUpdate: func(
						ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption,
					) error {
						writes++

						if tt.statusFails {
							return k8sErrors.NewConflict(keycloakApi.GroupVersion.WithResource("keycloakrealmusers").GroupResource(), obj.GetName(), errors.New("conflict"))
						}

						return c.SubResource(subResourceName).Update(ctx, obj, opts...)
					},
				}).
				Build()

			ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
			r := NewReconcile(cl, helpermock.NewMockControllerHelper(t))

			for range max(tt.runs, 1) {
				_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
				require.ErrorContains(t, err, tt.wantErrMsg)
			}

			assert.Equal(t, tt.wantWrites, writes)

			got := &keycloakApi.KeycloakRealmUser{}
			require.NoError(t, cl.Get(ctx, key, got))
			assert.Contains(t, got.Status.Value, tt.wantValue)
		})
	}
}

func TestReconcile_DeletingRealmGoneWithOwnerRefEnabled(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))

	now := metav1.Now()
	key := client.ObjectKey{Namespace: "default", Name: "user"}
	user := &keycloakApi.KeycloakRealmUser{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         key.Namespace,
			Name:              key.Name,
			DeletionTimestamp: &now,
			Finalizers:        []string{finalizerName},
		},
		Spec: keycloakApi.KeycloakRealmUserSpec{
			Username: "user",
			RealmRef: common.RealmRef{Kind: keycloakApi.KeycloakRealmKind, Name: "missing-realm"},
		},
		// A non-OK status makes the reconcile write status on the deleted object after the release.
		Status: keycloakApi.KeycloakRealmUserStatus{Value: common.StatusOK},
	}
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&keycloakApi.KeycloakRealmUser{}).
		WithObjects(user).
		Build()

	h, err := helper.MakeHelper(cl, scheme, "default", destination.AllowAll(), helper.EnableOwnerRef(true))
	require.NoError(t, err)

	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())

	_, err = NewReconcile(cl, h).Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)

	err = cl.Get(ctx, key, &keycloakApi.KeycloakRealmUser{})
	require.True(t, k8sErrors.IsNotFound(err), "user must be released, got: %v", err)
}

func TestReconcile_ConditionOnlyStatusChangeIsPersisted(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))

	key := client.ObjectKey{Namespace: "default", Name: "user"}
	user := &keycloakApi.KeycloakRealmUser{
		ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: key.Name, Generation: 1},
		Spec: keycloakApi.KeycloakRealmUserSpec{
			Username:       "user",
			KeepResource:   true,
			PasswordSecret: &keycloakApi.PasswordSecret{Name: "user-password", Key: "password", Temporary: true},
		},
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: key.Namespace, Name: "user-password"},
		Data:       map[string][]byte{"password": []byte("value")},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&keycloakApi.KeycloakRealmUser{}).
		WithObjects(user, secret).
		Build()

	users := v2mocks.NewMockUsersClient(t)
	users.EXPECT().FindUserByUsername(mock.Anything, "realm", "user").
		Return(&keycloakapi.UserRepresentation{Id: ptr.To("user-id")}, nil, nil)
	users.EXPECT().UpdateUser(mock.Anything, "realm", "user-id", mock.Anything).Return(nil, nil)
	users.EXPECT().SetUserPassword(mock.Anything, "realm", "user-id", mock.Anything).Return(nil, nil).Once()

	h := helpermock.NewMockControllerHelper(t)
	h.EXPECT().SetRealmOwnerRef(mock.Anything, mock.Anything).Return(nil)
	h.EXPECT().CreateKeycloakClientFromRealmRef(mock.Anything, mock.Anything).
		Return(&keycloakapi.KeycloakClient{Users: users}, nil)
	h.EXPECT().GetRealmNameFromRef(mock.Anything, mock.Anything).Return("realm", nil)

	ctx := ctrl.LoggerInto(context.Background(), logr.Discard())
	r := NewReconcile(cl, h)

	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)

	got := &keycloakApi.KeycloakRealmUser{}
	require.NoError(t, cl.Get(ctx, key, got))
	require.NotEmpty(t, got.Status.PasswordSecretHash)

	got.Generation = 2
	require.NoError(t, cl.Update(ctx, got))

	_, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: key})
	require.NoError(t, err)

	require.NoError(t, cl.Get(ctx, key, got))

	cond := meta.FindStatusCondition(got.Status.Conditions, chain.ConditionPasswordSynced)
	require.NotNil(t, cond)
	assert.Equal(t, int64(2), cond.ObservedGeneration)
}
