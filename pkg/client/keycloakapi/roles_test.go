package keycloakapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestRolesClient_CRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name to avoid conflicts
	realmName := fmt.Sprintf("test-realm-role-crud-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	roleName := "test-role"
	roleDescription := "Test role for CRUD operations"

	// 1. Create a realm role
	role := keycloakapi.RoleRepresentation{
		Name:        &roleName,
		Description: &roleDescription,
	}

	resp, err := c.Roles.CreateRealmRole(ctx, realmName, role)
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
	params := &keycloakapi.GetRealmRolesParams{
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
	require.True(t, keycloakapi.IsNotFound(err), "Expected 404 Not Found error after deletion")
}

func TestRolesClient_GetRealmRole_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Test getting a non-existent role
	role, resp, err := c.Roles.GetRealmRole(ctx, keycloakapi.MasterRealm, "nonexistent-role-12345")
	require.Error(t, err)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 error for non-existent role")
	require.Nil(t, role, "Role should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestRolesClient_DeleteRealmRole_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Test deleting a non-existent role
	resp, err := c.Roles.DeleteRealmRole(ctx, keycloakapi.MasterRealm, "nonexistent-role-12345")
	require.Error(t, err)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 error for non-existent role")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestRolesClient_CreateRealmRole_Conflict(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name to avoid conflicts
	realmName := fmt.Sprintf("test-realm-role-conflict-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	roleName := "duplicate-role"
	role := keycloakapi.RoleRepresentation{
		Name: &roleName,
	}

	// Create the role
	resp, err := c.Roles.CreateRealmRole(ctx, realmName, role)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Create the same role again — should conflict
	resp, err = c.Roles.CreateRealmRole(ctx, realmName, role)
	require.Error(t, err)
	require.True(t, keycloakapi.IsConflict(err), "Should return 409 Conflict error for duplicate role")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestRolesClient_UpdateAndComposites(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-role-composites-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	mainRoleName := "main-role"
	subRole1Name := "sub-role-1"
	subRole2Name := "sub-role-2"
	isComposite := true

	// Create main composite role
	_, err = c.Roles.CreateRealmRole(ctx, realmName, keycloakapi.RoleRepresentation{
		Name:      &mainRoleName,
		Composite: &isComposite,
	})
	require.NoError(t, err)

	// Create two sub-roles
	_, err = c.Roles.CreateRealmRole(ctx, realmName, keycloakapi.RoleRepresentation{Name: &subRole1Name})
	require.NoError(t, err)

	_, err = c.Roles.CreateRealmRole(ctx, realmName, keycloakapi.RoleRepresentation{Name: &subRole2Name})
	require.NoError(t, err)

	subRole1, _, err := c.Roles.GetRealmRole(ctx, realmName, subRole1Name)
	require.NoError(t, err)

	subRole2, _, err := c.Roles.GetRealmRole(ctx, realmName, subRole2Name)
	require.NoError(t, err)

	// Test UpdateRealmRole
	updatedDesc := "updated description"

	mainRole, _, err := c.Roles.GetRealmRole(ctx, realmName, mainRoleName)
	require.NoError(t, err)

	mainRole.Description = &updatedDesc
	_, err = c.Roles.UpdateRealmRole(ctx, realmName, mainRoleName, *mainRole)
	require.NoError(t, err)

	fetchedRole, _, err := c.Roles.GetRealmRole(ctx, realmName, mainRoleName)
	require.NoError(t, err)
	require.Equal(t, updatedDesc, *fetchedRole.Description)

	// Test AddRealmRoleComposites
	_, err = c.Roles.AddRealmRoleComposites(
		ctx, realmName, mainRoleName, []keycloakapi.RoleRepresentation{*subRole1, *subRole2},
	)
	require.NoError(t, err)

	// Test GetRealmRoleComposites
	composites, _, err := c.Roles.GetRealmRoleComposites(ctx, realmName, mainRoleName)
	require.NoError(t, err)
	require.Len(t, composites, 2)

	// Test DeleteRealmRoleComposites
	_, err = c.Roles.DeleteRealmRoleComposites(ctx, realmName, mainRoleName, []keycloakapi.RoleRepresentation{*subRole2})
	require.NoError(t, err)

	composites, _, err = c.Roles.GetRealmRoleComposites(ctx, realmName, mainRoleName)
	require.NoError(t, err)
	require.Len(t, composites, 1)
	require.Equal(t, subRole1Name, *composites[0].Name)
}

func TestRolesClient_GetRealmRoleComposites_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	_, resp, err := c.Roles.GetRealmRoleComposites(context.Background(), keycloakapi.MasterRealm, "nonexistent-role-12345")
	require.Error(t, err)
	require.True(t, keycloakapi.IsNotFound(err))
	require.NotNil(t, resp)
}

func TestRolesClient_AddRealmRoleComposites_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Roles.AddRealmRoleComposites(
		context.Background(), keycloakapi.MasterRealm, "nonexistent-role-12345", []keycloakapi.RoleRepresentation{})
	require.Error(t, err)
	require.NotNil(t, resp)
}

func TestRolesClient_DeleteRealmRoleComposites_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Roles.DeleteRealmRoleComposites(
		context.Background(), keycloakapi.MasterRealm, "nonexistent-role-12345", []keycloakapi.RoleRepresentation{})
	require.Error(t, err)
	require.NotNil(t, resp)
}
