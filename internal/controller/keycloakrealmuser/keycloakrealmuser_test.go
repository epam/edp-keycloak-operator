package keycloakrealmuser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
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
