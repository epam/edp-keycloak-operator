package adapter

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

// createRoleAssertionFunc creates a role assertion function for testing
func createRoleAssertionFunc(
	expectedToken, expectedRealm, expectedEntityID string,
) func(t *testing.T) func(
	ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
) error {
	return func(t *testing.T) func(
		ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
	) error {
		return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
			assert.Equal(t, expectedToken, token)
			assert.Equal(t, expectedRealm, realm)
			assert.Equal(t, expectedEntityID, entityID)

			switch clientID {
			case "client1-uuid":
				assert.Len(t, roles, 2)

				roleNames := make([]string, len(roles))
				for i, role := range roles {
					roleNames[i] = *role.Name
				}

				assert.ElementsMatch(t, []string{"role1", "role2"}, roleNames)
			case "client2-uuid":
				assert.Len(t, roles, 1)
				assert.Equal(t, "role3", *roles[0].Name)
			default:
				t.Errorf("unexpected clientID: %s", clientID)
			}

			return nil
		}
	}
}

func TestGoCloakAdapter_syncEntityRealmRoles(t *testing.T) {
	tests := []struct {
		name              string
		entityID          string
		realm             string
		claimedRealmRoles []string
		currentRealmRoles *[]gocloak.Role
		setupMock         func(t *testing.T) *mocks.MockGoCloak
		setupAddRoleFunc  func(t *testing.T) func(
			ctx context.Context,
			token, realm, entityID string,
			roles []gocloak.Role,
		) error
		setupDelRoleFunc func(t *testing.T) func(
			ctx context.Context,
			token, realm, entityID string, roles []gocloak.Role) error
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:              "successful role addition - no current roles",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{"role1", "role2"},
			currentRealmRoles: &[]gocloak.Role{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				role1 := &gocloak.Role{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				}
				role2 := &gocloak.Role{
					ID:   gocloak.StringP("role2-id"),
					Name: gocloak.StringP("role2"),
				}

				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role1").Return(role1, nil)
				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role2").Return(role2, nil)

				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					assert.Equal(t, "access-token", token)
					assert.Equal(t, "test-realm", realm)
					assert.Equal(t, "entity-123", entityID)
					assert.Len(t, roles, 2)

					roleNames := make([]string, len(roles))
					for i, role := range roles {
						roleNames[i] = *role.Name
					}
					assert.ElementsMatch(t, []string{"role1", "role2"}, roleNames)

					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:              "successful role deletion - no claimed roles",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{},
			currentRealmRoles: &[]gocloak.Role{
				{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				},
				{
					ID:   gocloak.StringP("role2-id"),
					Name: gocloak.StringP("role2"),
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					assert.Equal(t, "access-token", token)
					assert.Equal(t, "test-realm", realm)
					assert.Equal(t, "entity-123", entityID)
					assert.Len(t, roles, 2)

					roleNames := make([]string, len(roles))
					for i, role := range roles {
						roleNames[i] = *role.Name
					}
					assert.ElementsMatch(t, []string{"role1", "role2"}, roleNames)

					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:              "mixed addition and deletion",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{"role2", "role3"},
			currentRealmRoles: &[]gocloak.Role{
				{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				},
				{
					ID:   gocloak.StringP("role2-id"),
					Name: gocloak.StringP("role2"),
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				role3 := &gocloak.Role{
					ID:   gocloak.StringP("role3-id"),
					Name: gocloak.StringP("role3"),
				}

				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role3").Return(role3, nil)

				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					assert.Equal(t, "access-token", token)
					assert.Equal(t, "test-realm", realm)
					assert.Equal(t, "entity-123", entityID)
					assert.Len(t, roles, 1)
					assert.Equal(t, "role3", *roles[0].Name)

					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					assert.Equal(t, "access-token", token)
					assert.Equal(t, "test-realm", realm)
					assert.Equal(t, "entity-123", entityID)
					assert.Len(t, roles, 1)
					assert.Equal(t, "role1", *roles[0].Name)

					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:              "no changes needed - roles already synced",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{"role1", "role2"},
			currentRealmRoles: &[]gocloak.Role{
				{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				},
				{
					ID:   gocloak.StringP("role2-id"),
					Name: gocloak.StringP("role2"),
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:              "error in makeEntityRolesToAdd - GetRealmRole fails",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{"non-existent-role"},
			currentRealmRoles: &[]gocloak.Role{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)
				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "non-existent-role").
					Return(nil, errors.New("role not found"))
				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.Error,
		},
		{
			name:              "error in addRoleFunc",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{"role1"},
			currentRealmRoles: &[]gocloak.Role{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)
				role1 := &gocloak.Role{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				}
				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role1").Return(role1, nil)
				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					return errors.New("failed to add roles")
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.Error,
		},
		{
			name:              "error in delRoleFunc",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{},
			currentRealmRoles: &[]gocloak.Role{
				{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					return errors.New("failed to delete roles")
				}
			},
			wantErr: require.Error,
		},
		{
			name:              "nil current roles",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{"role1"},
			currentRealmRoles: nil,
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)
				role1 := &gocloak.Role{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				}
				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role1").Return(role1, nil)
				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					assert.Len(t, roles, 1)
					assert.Equal(t, "role1", *roles[0].Name)
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:              "empty claimed roles with current roles",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{},
			currentRealmRoles: &[]gocloak.Role{
				{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				},
				{
					ID:   gocloak.StringP("role2-id"),
					Name: gocloak.StringP("role2"),
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					assert.Len(t, roles, 2)
					roleNames := make([]string, len(roles))
					for i, role := range roles {
						roleNames[i] = *role.Name
					}
					assert.ElementsMatch(t, []string{"role1", "role2"}, roleNames)
					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:              "both empty - no operations",
			entityID:          "entity-123",
			realm:             "test-realm",
			claimedRealmRoles: []string{},
			currentRealmRoles: &[]gocloak.Role{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context,
				token, realm, entityID string,
				roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock(t)
			addRoleFunc := tt.setupAddRoleFunc(t)
			delRoleFunc := tt.setupDelRoleFunc(t)

			adapter := GoCloakAdapter{
				client: mockClient,
				token: &gocloak.JWT{
					AccessToken: "access-token",
				},
			}

			err := adapter.syncEntityRealmRoles(
				tt.entityID,
				tt.realm,
				tt.claimedRealmRoles,
				tt.currentRealmRoles,
				addRoleFunc,
				delRoleFunc,
			)

			tt.wantErr(t, err)
		})
	}
}

func TestGoCloakAdapter_makeCurrentEntityRoles(t *testing.T) {
	tests := []struct {
		name              string
		currentRealmRoles *[]gocloak.Role
		want              map[string]gocloak.Role
	}{
		{
			name:              "nil roles",
			currentRealmRoles: nil,
			want:              map[string]gocloak.Role{},
		},
		{
			name:              "empty roles",
			currentRealmRoles: &[]gocloak.Role{},
			want:              map[string]gocloak.Role{},
		},
		{
			name: "single role",
			currentRealmRoles: &[]gocloak.Role{
				{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				},
			},
			want: map[string]gocloak.Role{
				"role1": {
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				},
			},
		},
		{
			name: "multiple roles",
			currentRealmRoles: &[]gocloak.Role{
				{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				},
				{
					ID:   gocloak.StringP("role2-id"),
					Name: gocloak.StringP("role2"),
				},
			},
			want: map[string]gocloak.Role{
				"role1": {
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				},
				"role2": {
					ID:   gocloak.StringP("role2-id"),
					Name: gocloak.StringP("role2"),
				},
			},
		},
		{
			name: "roles with duplicate names - last one wins",
			currentRealmRoles: &[]gocloak.Role{
				{
					ID:   gocloak.StringP("role1-id-first"),
					Name: gocloak.StringP("role1"),
				},
				{
					ID:   gocloak.StringP("role1-id-second"),
					Name: gocloak.StringP("role1"),
				},
			},
			want: map[string]gocloak.Role{
				"role1": {
					ID:   gocloak.StringP("role1-id-second"),
					Name: gocloak.StringP("role1"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := GoCloakAdapter{}
			got := adapter.makeCurrentEntityRoles(tt.currentRealmRoles)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoCloakAdapter_makeClimedEntityRoles(t *testing.T) {
	tests := []struct {
		name              string
		claimedRealmRoles []string
		want              map[string]struct{}
	}{
		{
			name:              "nil roles",
			claimedRealmRoles: nil,
			want:              map[string]struct{}{},
		},
		{
			name:              "empty roles",
			claimedRealmRoles: []string{},
			want:              map[string]struct{}{},
		},
		{
			name:              "single role",
			claimedRealmRoles: []string{"role1"},
			want: map[string]struct{}{
				"role1": {},
			},
		},
		{
			name:              "multiple roles",
			claimedRealmRoles: []string{"role1", "role2", "role3"},
			want: map[string]struct{}{
				"role1": {},
				"role2": {},
				"role3": {},
			},
		},
		{
			name:              "duplicate roles - deduplication",
			claimedRealmRoles: []string{"role1", "role2", "role1", "role3", "role2"},
			want: map[string]struct{}{
				"role1": {},
				"role2": {},
				"role3": {},
			},
		},
		{
			name:              "empty string role",
			claimedRealmRoles: []string{"role1", "", "role2"},
			want: map[string]struct{}{
				"role1": {},
				"":      {},
				"role2": {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := GoCloakAdapter{}
			got := adapter.makeClimedEntityRoles(tt.claimedRealmRoles)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoCloakAdapter_makeEntityRolesToAdd(t *testing.T) {
	tests := []struct {
		name                string
		realm               string
		claimedRealmRoles   []string
		currentRealmRoleMap map[string]gocloak.Role
		setupMock           func(t *testing.T) *mocks.MockGoCloak
		wantRoles           []gocloak.Role
		wantErr             require.ErrorAssertionFunc
	}{
		{
			name:                "no claimed roles",
			realm:               "test-realm",
			claimedRealmRoles:   []string{},
			currentRealmRoleMap: map[string]gocloak.Role{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			wantRoles: []gocloak.Role{},
			wantErr:   require.NoError,
		},
		{
			name:              "all roles already exist",
			realm:             "test-realm",
			claimedRealmRoles: []string{"role1", "role2"},
			currentRealmRoleMap: map[string]gocloak.Role{
				"role1": {ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
				"role2": {ID: gocloak.StringP("role2-id"), Name: gocloak.StringP("role2")},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			wantRoles: []gocloak.Role{},
			wantErr:   require.NoError,
		},
		{
			name:              "some roles need to be added",
			realm:             "test-realm",
			claimedRealmRoles: []string{"role1", "role2", "role3"},
			currentRealmRoleMap: map[string]gocloak.Role{
				"role1": {ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				role2 := &gocloak.Role{
					ID:   gocloak.StringP("role2-id"),
					Name: gocloak.StringP("role2"),
				}
				role3 := &gocloak.Role{
					ID:   gocloak.StringP("role3-id"),
					Name: gocloak.StringP("role3"),
				}

				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role2").Return(role2, nil)
				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role3").Return(role3, nil)

				return m
			},
			wantRoles: []gocloak.Role{
				{ID: gocloak.StringP("role2-id"), Name: gocloak.StringP("role2")},
				{ID: gocloak.StringP("role3-id"), Name: gocloak.StringP("role3")},
			},
			wantErr: require.NoError,
		},
		{
			name:                "all roles need to be added",
			realm:               "test-realm",
			claimedRealmRoles:   []string{"role1", "role2"},
			currentRealmRoleMap: map[string]gocloak.Role{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				role1 := &gocloak.Role{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				}
				role2 := &gocloak.Role{
					ID:   gocloak.StringP("role2-id"),
					Name: gocloak.StringP("role2"),
				}

				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role1").Return(role1, nil)
				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role2").Return(role2, nil)

				return m
			},
			wantRoles: []gocloak.Role{
				{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
				{ID: gocloak.StringP("role2-id"), Name: gocloak.StringP("role2")},
			},
			wantErr: require.NoError,
		},
		{
			name:                "GetRealmRole fails for one role",
			realm:               "test-realm",
			claimedRealmRoles:   []string{"role1", "non-existent-role"},
			currentRealmRoleMap: map[string]gocloak.Role{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				role1 := &gocloak.Role{
					ID:   gocloak.StringP("role1-id"),
					Name: gocloak.StringP("role1"),
				}

				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "role1").Return(role1, nil)
				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "non-existent-role").
					Return(nil, errors.New("role not found"))

				return m
			},
			wantRoles: nil,
			wantErr:   require.Error,
		},
		{
			name:                "GetRealmRole fails for all roles",
			realm:               "test-realm",
			claimedRealmRoles:   []string{"non-existent-role1", "non-existent-role2"},
			currentRealmRoleMap: map[string]gocloak.Role{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				m.On("GetRealmRole", mock.Anything, "access-token", "test-realm", "non-existent-role1").
					Return(nil, errors.New("role not found"))

				return m
			},
			wantRoles: nil,
			wantErr:   require.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock(t)

			adapter := GoCloakAdapter{
				client: mockClient,
				token: &gocloak.JWT{
					AccessToken: "access-token",
				},
			}

			got, err := adapter.makeEntityRolesToAdd(
				tt.realm,
				tt.claimedRealmRoles,
				tt.currentRealmRoleMap,
			)

			tt.wantErr(t, err)

			if err == nil {
				// Sort both slices by name for comparison since order might vary
				gotCopy := slices.Clone(got)
				wantCopy := slices.Clone(tt.wantRoles)

				slices.SortFunc(gotCopy, func(a, b gocloak.Role) int {
					return cmp.Compare(*a.Name, *b.Name)
				})

				slices.SortFunc(wantCopy, func(a, b gocloak.Role) int {
					return cmp.Compare(*a.Name, *b.Name)
				})

				assert.Equal(t, wantCopy, gotCopy)
			}
		})
	}
}

func TestGoCloakAdapter_syncEntityClientRoles(t *testing.T) {
	tests := []struct {
		name             string
		realm            string
		entityID         string
		claimedRoles     map[string][]string
		currentRoles     map[string]*gocloak.ClientMappingsRepresentation
		setupMock        func(t *testing.T) *mocks.MockGoCloak
		setupAddRoleFunc func(t *testing.T) func(
			ctx context.Context,
			token, realm, clientID, entityID string,
			roles []gocloak.Role,
		) error
		setupDelRoleFunc func(t *testing.T) func(
			ctx context.Context,
			token, realm, clientID, entityID string,
			roles []gocloak.Role,
		) error
		wantErr require.ErrorAssertionFunc
	}{
		{
			name:     "successful client role addition - no current roles",
			realm:    "test-realm",
			entityID: "entity-123",
			claimedRoles: map[string][]string{
				"client1": {"role1", "role2"},
				"client2": {"role3"},
			},
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock GetClients for client1 - match on specific ClientID
				m.On("GetClients", mock.Anything, "access-token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetClientsParams) bool {
						return params.ClientID != nil && *params.ClientID == "client1"
					})).
					Return([]*gocloak.Client{
						{ID: gocloak.StringP("client1-uuid"), ClientID: gocloak.StringP("client1")},
					}, nil).Once()

				// Mock GetClients for client2 - match on specific ClientID
				m.On("GetClients", mock.Anything, "access-token", "test-realm",
					mock.MatchedBy(func(params gocloak.GetClientsParams) bool {
						return params.ClientID != nil && *params.ClientID == "client2"
					})).
					Return([]*gocloak.Client{
						{ID: gocloak.StringP("client2-uuid"), ClientID: gocloak.StringP("client2")},
					}, nil).Once()

				// Mock GetClientRole calls
				m.On("GetClientRole", mock.Anything, "access-token", "test-realm", "client1-uuid", "role1").
					Return(&gocloak.Role{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")}, nil)
				m.On("GetClientRole", mock.Anything, "access-token", "test-realm", "client1-uuid", "role2").
					Return(&gocloak.Role{ID: gocloak.StringP("role2-id"), Name: gocloak.StringP("role2")}, nil)
				m.On("GetClientRole", mock.Anything, "access-token", "test-realm", "client2-uuid", "role3").
					Return(&gocloak.Role{ID: gocloak.StringP("role3-id"), Name: gocloak.StringP("role3")}, nil)

				return m
			},
			setupAddRoleFunc: createRoleAssertionFunc("access-token", "test-realm", "entity-123"),
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:         "successful client role deletion - no claimed roles",
			realm:        "test-realm",
			entityID:     "entity-123",
			claimedRoles: map[string][]string{},
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{
				"client1": {
					ID: gocloak.StringP("client1-uuid"),
					Mappings: &[]gocloak.Role{
						{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
						{ID: gocloak.StringP("role2-id"), Name: gocloak.StringP("role2")},
					},
				},
				"client2": {
					ID: gocloak.StringP("client2-uuid"),
					Mappings: &[]gocloak.Role{
						{ID: gocloak.StringP("role3-id"), Name: gocloak.StringP("role3")},
					},
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: createRoleAssertionFunc("access-token", "test-realm", "entity-123"),
			wantErr:          require.NoError,
		},
		{
			name:     "mixed client role operations",
			realm:    "test-realm",
			entityID: "entity-123",
			claimedRoles: map[string][]string{
				"client1": {"role2", "role3"}, // keep role2, add role3, remove role1
			},
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{
				"client1": {
					ID: gocloak.StringP("client1-uuid"),
					Mappings: &[]gocloak.Role{
						{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
						{ID: gocloak.StringP("role2-id"), Name: gocloak.StringP("role2")},
					},
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock GetClients for client1
				m.On("GetClients", mock.Anything, "access-token", "test-realm", mock.Anything).
					Return([]*gocloak.Client{
						{ID: gocloak.StringP("client1-uuid"), ClientID: gocloak.StringP("client1")},
					}, nil).Once()

				// Mock GetClientRole for role3 (new role to add)
				m.On("GetClientRole", mock.Anything, "access-token", "test-realm", "client1-uuid", "role3").
					Return(&gocloak.Role{ID: gocloak.StringP("role3-id"), Name: gocloak.StringP("role3")}, nil)

				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					assert.Equal(t, "access-token", token)
					assert.Equal(t, "test-realm", realm)
					assert.Equal(t, "client1-uuid", clientID)
					assert.Equal(t, "entity-123", entityID)
					assert.Len(t, roles, 1)
					assert.Equal(t, "role3", *roles[0].Name)
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					assert.Equal(t, "access-token", token)
					assert.Equal(t, "test-realm", realm)
					assert.Equal(t, "client1-uuid", clientID)
					assert.Equal(t, "entity-123", entityID)
					assert.Len(t, roles, 1)
					assert.Equal(t, "role1", *roles[0].Name)
					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:     "orphaned client roles cleanup",
			realm:    "test-realm",
			entityID: "entity-123",
			claimedRoles: map[string][]string{
				"client1": {"role1"}, // only client1 is claimed
			},
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{
				"client1": {
					ID: gocloak.StringP("client1-uuid"),
					Mappings: &[]gocloak.Role{
						{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
					},
				},
				"client2": { // client2 should be cleaned up
					ID: gocloak.StringP("client2-uuid"),
					Mappings: &[]gocloak.Role{
						{ID: gocloak.StringP("role2-id"), Name: gocloak.StringP("role2")},
						{ID: gocloak.StringP("role3-id"), Name: gocloak.StringP("role3")},
					},
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock GetClients for client1
				m.On("GetClients", mock.Anything, "access-token", "test-realm", mock.Anything).
					Return([]*gocloak.Client{
						{ID: gocloak.StringP("client1-uuid"), ClientID: gocloak.StringP("client1")},
					}, nil).Once()

				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					assert.Equal(t, "access-token", token)
					assert.Equal(t, "test-realm", realm)
					assert.Equal(t, "client2-uuid", clientID)
					assert.Equal(t, "entity-123", entityID)
					assert.Len(t, roles, 2)
					roleNames := make([]string, len(roles))
					for i, role := range roles {
						roleNames[i] = *role.Name
					}
					assert.ElementsMatch(t, []string{"role2", "role3"}, roleNames)
					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:         "error in syncOneEntityClientRole - GetClientID fails",
			realm:        "test-realm",
			entityID:     "entity-123",
			claimedRoles: map[string][]string{"non-existent-client": {"role1"}},
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock GetClients to return empty result (client not found)
				m.On("GetClients", mock.Anything, "access-token", "test-realm", mock.Anything).
					Return([]*gocloak.Client{}, nil).Once()

				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.Error,
		},
		{
			name:     "error in addRoleFunc",
			realm:    "test-realm",
			entityID: "entity-123",
			claimedRoles: map[string][]string{
				"client1": {"role1"},
			},
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock GetClients for client1
				m.On("GetClients", mock.Anything, "access-token", "test-realm", mock.Anything).
					Return([]*gocloak.Client{
						{ID: gocloak.StringP("client1-uuid"), ClientID: gocloak.StringP("client1")},
					}, nil).Once()

				// Mock GetClientRole
				m.On("GetClientRole", mock.Anything, "access-token", "test-realm", "client1-uuid", "role1").
					Return(&gocloak.Role{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")}, nil)

				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					return errors.New("failed to add client roles")
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.Error,
		},
		{
			name:         "error in delRoleFunc for orphaned client",
			realm:        "test-realm",
			entityID:     "entity-123",
			claimedRoles: map[string][]string{},
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{
				"client1": {
					ID: gocloak.StringP("client1-uuid"),
					Mappings: &[]gocloak.Role{
						{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
					},
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					return errors.New("failed to delete client roles")
				}
			},
			wantErr: require.Error,
		},
		{
			name:         "empty scenarios - no operations",
			realm:        "test-realm",
			entityID:     "entity-123",
			claimedRoles: map[string][]string{},
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				return mocks.NewMockGoCloak(t)
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("add function should not be called")
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.NoError,
		},
		{
			name:     "nil mappings in current roles - should be ignored",
			realm:    "test-realm",
			entityID: "entity-123",
			claimedRoles: map[string][]string{
				"client1": {"role1"},
			},
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{
				"client1": {
					ID:       gocloak.StringP("client1-uuid"),
					Mappings: nil, // nil mappings should be handled gracefully
				},
				"client2": {
					ID:       gocloak.StringP("client2-uuid"),
					Mappings: nil, // nil mappings should be handled gracefully
				},
			},
			setupMock: func(t *testing.T) *mocks.MockGoCloak {
				m := mocks.NewMockGoCloak(t)

				// Mock GetClients for client1
				m.On("GetClients", mock.Anything, "access-token", "test-realm", mock.Anything).
					Return([]*gocloak.Client{
						{ID: gocloak.StringP("client1-uuid"), ClientID: gocloak.StringP("client1")},
					}, nil).Once()

				// Mock GetClientRole
				m.On("GetClientRole", mock.Anything, "access-token", "test-realm", "client1-uuid", "role1").
					Return(&gocloak.Role{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")}, nil)

				return m
			},
			setupAddRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					assert.Equal(t, "client1-uuid", clientID)
					assert.Len(t, roles, 1)
					assert.Equal(t, "role1", *roles[0].Name)
					return nil
				}
			},
			setupDelRoleFunc: func(t *testing.T) func(
				ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role,
			) error {
				return func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error {
					t.Error("delete function should not be called")
					return nil
				}
			},
			wantErr: require.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := tt.setupMock(t)
			addRoleFunc := tt.setupAddRoleFunc(t)
			delRoleFunc := tt.setupDelRoleFunc(t)

			adapter := GoCloakAdapter{
				client: mockClient,
				token: &gocloak.JWT{
					AccessToken: "access-token",
				},
			}

			err := adapter.syncEntityClientRoles(
				tt.realm,
				tt.entityID,
				tt.claimedRoles,
				tt.currentRoles,
				addRoleFunc,
				delRoleFunc,
			)

			tt.wantErr(t, err)
		})
	}
}

func TestGoCloakAdapter_makeCurrentClientRoles(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		currentRoles map[string]*gocloak.ClientMappingsRepresentation
		want         map[string]*gocloak.Role
	}{
		{
			name:         "empty current roles",
			clientID:     "client1",
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{},
			want:         map[string]*gocloak.Role{},
		},
		{
			name:     "client not in current roles",
			clientID: "client1",
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{
				"client2": {
					ID: gocloak.StringP("client2-uuid"),
					Mappings: &[]gocloak.Role{
						{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
					},
				},
			},
			want: map[string]*gocloak.Role{},
		},
		{
			name:     "client with nil mappings",
			clientID: "client1",
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{
				"client1": {
					ID:       gocloak.StringP("client1-uuid"),
					Mappings: nil,
				},
			},
			want: map[string]*gocloak.Role{},
		},
		{
			name:     "client with single role",
			clientID: "client1",
			currentRoles: map[string]*gocloak.ClientMappingsRepresentation{
				"client1": {
					ID: gocloak.StringP("client1-uuid"),
					Mappings: &[]gocloak.Role{
						{ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
					},
				},
			},
			want: map[string]*gocloak.Role{
				"role1": {ID: gocloak.StringP("role1-id"), Name: gocloak.StringP("role1")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := GoCloakAdapter{}
			got := adapter.makeCurrentClientRoles(tt.clientID, tt.currentRoles)
			assert.Equal(t, tt.want, got)
		})
	}
}
