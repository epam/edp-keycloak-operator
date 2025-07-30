package keycloakrealmuser

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
