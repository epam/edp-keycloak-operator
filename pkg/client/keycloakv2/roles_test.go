package keycloakv2_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestRolesClient_CRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name to avoid conflicts
	realmName := fmt.Sprintf("test-realm-role-crud-%d", time.Now().UnixNano())
	enabled := true

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm first
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	roleName := "test-role"
	roleDescription := "Test role for CRUD operations"

	// 1. Create a realm role
	role := keycloakv2.RoleRepresentation{
		Name:        &roleName,
		Description: &roleDescription,
	}

	resp, err = c.Roles.CreateRealmRole(ctx, realmName, role)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// 2. Get the created role by name
	retrievedRole, resp, err := c.Roles.GetRealmRole(ctx, realmName, roleName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, retrievedRole)
	require.NotNil(t, retrievedRole.Name)
	require.Equal(t, roleName, *retrievedRole.Name)
	require.NotNil(t, retrievedRole.Description)
	require.Equal(t, roleDescription, *retrievedRole.Description)

	// 3. List all realm roles and verify our role is present
	roles, resp, err := c.Roles.GetRealmRoles(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, roles)
	require.Greater(t, len(roles), 0, "Should have at least one role")

	// Verify our role is in the list
	found := false

	for _, r := range roles {
		if r.Name != nil && *r.Name == roleName {
			found = true

			require.NotNil(t, r.Description)
			require.Equal(t, roleDescription, *r.Description)

			break
		}
	}

	require.True(t, found, "Created role should be in the roles list")

	// 4. List roles with search parameter
	search := "test-role"
	params := &keycloakv2.GetRealmRolesParams{
		Search: &search,
	}

	roles, resp, err = c.Roles.GetRealmRoles(ctx, realmName, params)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, roles)
	require.Greater(t, len(roles), 0, "Should find roles matching 'test-role'")

	found = false

	for _, r := range roles {
		if r.Name != nil && *r.Name == roleName {
			found = true

			break
		}
	}

	require.True(t, found, "Should find test-role in search results")

	// 5. Delete the role
	resp, err = c.Roles.DeleteRealmRole(ctx, realmName, roleName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// 6. Verify deletion
	_, _, err = c.Roles.GetRealmRole(ctx, realmName, roleName)
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Expected 404 Not Found error after deletion")
}

func TestRolesClient_GetRealmRole_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Test getting a non-existent role
	role, resp, err := c.Roles.GetRealmRole(ctx, keycloakv2.MasterRealm, "nonexistent-role-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent role")
	require.Nil(t, role, "Role should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestRolesClient_DeleteRealmRole_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Test deleting a non-existent role
	resp, err := c.Roles.DeleteRealmRole(ctx, keycloakv2.MasterRealm, "nonexistent-role-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent role")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestRolesClient_CreateRealmRole_Conflict(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name to avoid conflicts
	realmName := fmt.Sprintf("test-realm-role-conflict-%d", time.Now().UnixNano())
	enabled := true

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	roleName := "duplicate-role"
	role := keycloakv2.RoleRepresentation{
		Name: &roleName,
	}

	// Create the role
	resp, err = c.Roles.CreateRealmRole(ctx, realmName, role)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Create the same role again — should conflict
	resp, err = c.Roles.CreateRealmRole(ctx, realmName, role)
	require.Error(t, err)
	require.True(t, keycloakv2.IsConflict(err), "Should return 409 Conflict error for duplicate role")
	require.NotNil(t, resp, "Response should be present even for error")
}
