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
		{
			name:    "role without a name is incomplete",
			role:    &RoleRepresentation{Id: ptr.To("id-2")},
			wantErr: `realm role "admin" has no name`,
		},
		{
			name:    "role with another name is not the one asked for",
			role:    &RoleRepresentation{Id: ptr.To("id-3"), Name: ptr.To("other")},
			wantErr: `realm role lookup for "admin" returned "other"`,
		},
		{
			name:    "role with an empty id is incomplete",
			role:    &RoleRepresentation{Id: ptr.To(""), Name: ptr.To("admin")},
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

func TestRequireClientRoleWithID(t *testing.T) {
	tests := []struct {
		name    string
		role    *RoleRepresentation
		wantID  string
		wantErr string
	}{
		{
			name:   "role with an id is returned",
			role:   &RoleRepresentation{Id: ptr.To("id-1"), Name: ptr.To("viewer")},
			wantID: "id-1",
		},
		{
			name:    "no role is not found",
			role:    nil,
			wantErr: `client role "viewer" not found for client "my-client"`,
		},
		{
			name:    "role without an id is incomplete, not missing",
			role:    &RoleRepresentation{Name: ptr.To("viewer")},
			wantErr: `client role "viewer" has no id`,
		},
		{
			name:    "role without a name is incomplete",
			role:    &RoleRepresentation{Id: ptr.To("id-2")},
			wantErr: `client role "viewer" has no name`,
		},
		{
			name:    "role with another name is not the one asked for",
			role:    &RoleRepresentation{Id: ptr.To("id-3"), Name: ptr.To("other")},
			wantErr: `client role lookup for "viewer" returned "other"`,
		},
		{
			name:    "role with an empty id is incomplete",
			role:    &RoleRepresentation{Id: ptr.To(""), Name: ptr.To("viewer")},
			wantErr: `client role "viewer" has no id`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequireClientRoleWithID(tt.role, "my-client", "viewer")

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
