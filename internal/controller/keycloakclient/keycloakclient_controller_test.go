package keycloakclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

func TestReconcileKeycloakClient_migrateClientRoles(t *testing.T) {
	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		expectedRoles  []keycloakApi.ClientRole
		shouldMigrate  bool
	}{
		{
			name: "should migrate ClientRoles to ClientRolesV2",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientRoles: []string{"role1", "role2", "role3"},
				},
			},
			expectedRoles: []keycloakApi.ClientRole{
				{Name: "role1"},
				{Name: "role2"},
				{Name: "role3"},
			},
			shouldMigrate: true,
		},
		{
			name: "should not migrate when ClientRolesV2 is already populated",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientRoles:   []string{"role1", "role2"},
					ClientRolesV2: []keycloakApi.ClientRole{{Name: "existing-role"}},
				},
			},
			expectedRoles: []keycloakApi.ClientRole{{Name: "existing-role"}},
			shouldMigrate: false,
		},
		{
			name: "should not migrate when ClientRoles is empty",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientRoles: []string{},
				},
			},
			expectedRoles: nil,
			shouldMigrate: false,
		},
		{
			name: "should not migrate when both fields are empty",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ClientRoles:   []string{},
					ClientRolesV2: []keycloakApi.ClientRole{},
				},
			},
			expectedRoles: []keycloakApi.ClientRole{},
			shouldMigrate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a controller instance
			controller := &ReconcileKeycloakClient{}

			// Perform migration
			migrated := controller.migrateClientRoles(tt.keycloakClient)

			// Assert migration result
			assert.Equal(t, tt.shouldMigrate, migrated)

			// Assert ClientRolesV2 contains expected roles
			assert.Equal(t, tt.expectedRoles, tt.keycloakClient.Spec.ClientRolesV2)

			// If migration occurred, ClientRoles should remain unchanged for backward compatibility
			if tt.shouldMigrate {
				assert.NotNil(t, tt.keycloakClient.Spec.ClientRoles)
			}
		})
	}
}

func TestReconcileKeycloakClient_migrateClientRoles_EmptyRoleName(t *testing.T) {
	keycloakClient := &keycloakApi.KeycloakClient{
		Spec: keycloakApi.KeycloakClientSpec{
			ClientRoles: []string{"role1", "", "role3"},
		},
	}

	controller := &ReconcileKeycloakClient{}
	migrated := controller.migrateClientRoles(keycloakClient)

	require.True(t, migrated)
	require.Equal(t, []keycloakApi.ClientRole{
		{Name: "role1"},
		{Name: ""},
		{Name: "role3"},
	}, keycloakClient.Spec.ClientRolesV2)
	require.NotNil(t, keycloakClient.Spec.ClientRoles)
}

func TestReconcileKeycloakClient_migrateServiceAccountAttributes(t *testing.T) {
	tests := []struct {
		name           string
		keycloakClient *keycloakApi.KeycloakClient
		expectedAttrs  map[string][]string
		shouldMigrate  bool
	}{
		{
			name: "should migrate ClientRoles to ClientRolesV2",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Attributes: map[string]string{
							"attr1": "test-value",
						},
						AttributesV2: map[string][]string{},
					},
				},
			},
			expectedAttrs: map[string][]string{
				"attr1": {"test-value"},
			},
			shouldMigrate: true,
		},
		{
			name: "should not migrate when AttributesV2 is already populated",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Attributes: map[string]string{
							"attr1": "test-value",
						},
						AttributesV2: map[string][]string{
							"attr2": {"test-value-2"},
						},
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
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Attributes: map[string]string{},
					},
				},
			},
			expectedAttrs: nil,
			shouldMigrate: false,
		},
		{
			name: "should not migrate when both fields are empty",
			keycloakClient: &keycloakApi.KeycloakClient{
				Spec: keycloakApi.KeycloakClientSpec{
					ServiceAccount: &keycloakApi.ServiceAccount{
						Attributes:   map[string]string{},
						AttributesV2: map[string][]string{},
					},
				},
			},
			expectedAttrs: map[string][]string{},
			shouldMigrate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a controller instance
			controller := &ReconcileKeycloakClient{}

			// Perform migration
			migrated := controller.migrateServiceAccountAttributes(tt.keycloakClient)

			// Assert migration result
			assert.Equal(t, tt.shouldMigrate, migrated)

			// Assert ClientRolesV2 contains expected roles
			assert.Equal(t, tt.expectedAttrs, tt.keycloakClient.Spec.ServiceAccount.AttributesV2)

			// If migration occurred, ClientRoles should remain unchanged for backward compatibility
			if tt.shouldMigrate {
				assert.NotNil(t, tt.keycloakClient.Spec.ServiceAccount.Attributes)
			}
		})
	}
}
