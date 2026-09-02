package keycloakapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestRequireRealmRoleWithID(t *testing.T) {
	tests := []struct {
		name    string
		role    *RoleRepresentation
		wantID  string
		wantErr string
	}{
		{
			name:   "role with an id is returned",
			role:   &RoleRepresentation{Id: ptr.To("id-1"), Name: ptr.To("admin")},
			wantID: "id-1",
		},
		{
			name:    "no role is not found",
			role:    nil,
			wantErr: `realm role "admin" not found in realm "test-realm"`,
		},
		{
			name:    "role without an id is incomplete, not missing",
			role:    &RoleRepresentation{Name: ptr.To("admin")},
			wantErr: `realm role "admin" has no id`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequireRealmRoleWithID(tt.role, "test-realm", "admin")

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got.Id)
			assert.Equal(t, tt.wantID, *got.Id)
		})
	}
}
