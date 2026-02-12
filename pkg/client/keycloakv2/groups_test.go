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

const (
	testRoleName = "test-role"
)

func TestGroupsClient_CRUD(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-grp-crud-%d", time.Now().UnixNano())
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

	// 1. Create a top-level group
	groupName := "test-group"
	group := keycloakv2.GroupRepresentation{
		Name: &groupName,
	}

	resp, err = c.Groups.CreateGroup(ctx, realmName, group)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// Extract group ID from Location header
	groupID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, groupID, "Location header should be present")

	// 2. Get the created group
	retrievedGroup, resp, err := c.Groups.GetGroup(ctx, realmName, groupID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, retrievedGroup)
	require.NotNil(t, retrievedGroup.Name)
	require.Equal(t, "test-group", *retrievedGroup.Name)

	// 3. List all groups and verify our group is present
	groups, resp, err := c.Groups.GetGroups(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, groups)
	require.Greater(t, len(groups), 0, "Should have at least one group")

	// Verify our group is in the list
	found := false

	for _, g := range groups {
		if g.Id != nil && *g.Id == groupID {
			found = true

			require.Equal(t, "test-group", *g.Name)

			break
		}
	}

	require.True(t, found, "Created group should be in the groups list")

	// 4. Update the group
	updatedName := "test-group-updated"
	group = keycloakv2.GroupRepresentation{
		Name: &updatedName,
	}

	resp, err = c.Groups.UpdateGroup(ctx, realmName, groupID, group)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// 5. Verify the update
	retrievedGroup, resp, err = c.Groups.GetGroup(ctx, realmName, groupID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, retrievedGroup)
	require.NotNil(t, retrievedGroup.Name)
	require.Equal(t, "test-group-updated", *retrievedGroup.Name, "Group name should be updated")

	// 6. Delete the group
	resp, err = c.Groups.DeleteGroup(ctx, realmName, groupID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// 7. Verify deletion
	_, _, err = c.Groups.GetGroup(ctx, realmName, groupID)
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Expected 404 Not Found error after deletion")
}

func TestGroupsClient_ChildGroups(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-grp-child-%d", time.Now().UnixNano())
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

	var parentGroupID *string

	var childGroupID *string

	// 1. Create parent group
	t.Run("Create Parent Group", func(t *testing.T) {
		groupName := "parent-group"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateGroup(ctx, realmName, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		extractedID := keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, extractedID)
		parentGroupID = &extractedID
	})

	// 2. Create child group
	t.Run("Create Child Group", func(t *testing.T) {
		require.NotNil(t, parentGroupID)

		childName := "child-group"
		child := keycloakv2.GroupRepresentation{
			Name: &childName,
		}

		resp, err := c.Groups.CreateChildGroup(ctx, realmName, *parentGroupID, child)
		require.NoError(t, err)
		require.NotNil(t, resp)

		extractedID := keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, extractedID)
		childGroupID = &extractedID
	})

	// 3. Get child groups
	t.Run("Get Child Groups", func(t *testing.T) {
		require.NotNil(t, parentGroupID)

		children, resp, err := c.Groups.GetChildGroups(ctx, realmName, *parentGroupID, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, children)
		require.Equal(t, 1, len(children), "Should have exactly one child group")
		require.Equal(t, "child-group", *children[0].Name)
	})

	// 4. Get child by ID
	t.Run("Get Child by ID", func(t *testing.T) {
		require.NotNil(t, childGroupID)

		child, resp, err := c.Groups.GetGroup(ctx, realmName, *childGroupID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, child)
		require.Equal(t, "child-group", *child.Name)
	})

	// 5. Delete child group
	t.Run("Delete Child", func(t *testing.T) {
		require.NotNil(t, childGroupID)

		resp, err := c.Groups.DeleteGroup(ctx, realmName, *childGroupID)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	// 6. Verify child deleted
	t.Run("Verify Child Deleted", func(t *testing.T) {
		require.NotNil(t, parentGroupID)

		children, resp, err := c.Groups.GetChildGroups(ctx, realmName, *parentGroupID, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 0, len(children), "Should have no child groups after deletion")
	})
}

func TestGroupsClient_GetGroup_NotFound(t *testing.T) {
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

	// Test getting a non-existent group
	group, resp, err := c.Groups.GetGroup(ctx, keycloakv2.MasterRealm, "nonexistent-group-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent group")
	require.Nil(t, group, "Group should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestGroupsClient_UpdateGroup_NotFound(t *testing.T) {
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

	// Create a minimal group representation for testing
	updatedName := "test-group"
	group := keycloakv2.GroupRepresentation{
		Name: &updatedName,
	}

	// Test updating a non-existent group
	resp, err := c.Groups.UpdateGroup(ctx, keycloakv2.MasterRealm, "nonexistent-group-12345", group)
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent group")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestGroupsClient_DeleteGroup_NotFound(t *testing.T) {
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

	// Test deleting a non-existent group
	resp, err := c.Groups.DeleteGroup(ctx, keycloakv2.MasterRealm, "nonexistent-group-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent group")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestGroupsClient_GetChildGroups_NotFound(t *testing.T) {
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

	// Test getting children of a non-existent group
	children, resp, err := c.Groups.GetChildGroups(ctx, keycloakv2.MasterRealm, "nonexistent-group-12345", nil)
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent group")
	require.Nil(t, children, "Children should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestGroupsClient_GetGroups_WithParams(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-grp-params-%d", time.Now().UnixNano())
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

	// Create multiple groups for testing
	groupNames := []string{"alpha-group", "beta-group", "gamma-group"}
	for _, name := range groupNames {
		gName := name
		group := keycloakv2.GroupRepresentation{
			Name: &gName,
		}
		resp, err := c.Groups.CreateGroup(ctx, realmName, group)
		require.NoError(t, err)
		require.NotNil(t, resp)
	}

	// Test: Get all groups (no params)
	t.Run("Get All Groups", func(t *testing.T) {
		groups, resp, err := c.Groups.GetGroups(ctx, realmName, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 3, len(groups), "Should have exactly 3 groups")
	})

	// Test: Get groups with search parameter
	t.Run("Search Groups", func(t *testing.T) {
		searchTerm := "beta"
		params := &keycloakv2.GetGroupsParams{
			Search: &searchTerm,
		}

		groups, resp, err := c.Groups.GetGroups(ctx, realmName, params)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Greater(t, len(groups), 0, "Should find groups matching 'beta'")

		// Verify the result contains our beta-group
		found := false

		for _, g := range groups {
			if g.Name != nil && *g.Name == "beta-group" {
				found = true
				break
			}
		}

		require.True(t, found, "Should find beta-group in search results")
	})

	// Test: Get groups with pagination
	t.Run("Pagination", func(t *testing.T) {
		first := int32(0)
		max := int32(2)
		params := &keycloakv2.GetGroupsParams{
			First: &first,
			Max:   &max,
		}

		groups, resp, err := c.Groups.GetGroups(ctx, realmName, params)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.LessOrEqual(t, len(groups), 2, "Should return at most 2 groups")
	})
}

func TestGroupsClient_RealmRoleMappings(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-grp-realm-roles-%d", time.Now().UnixNano())
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

	var groupID *string

	var testRole *keycloakv2.RoleRepresentation

	// 1. Create test role
	t.Run("Create Test Role", func(t *testing.T) {
		roleName := testRoleName
		roleDescription := "Test role for group mapping"
		role := keycloakv2.RoleRepresentation{
			Name:        &roleName,
			Description: &roleDescription,
		}

		resp, err := c.Roles.CreateRealmRole(ctx, realmName, role)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Fetch the created role to get its ID
		createdRole, resp, err := c.Roles.GetRealmRole(ctx, realmName, roleName)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createdRole)
		testRole = createdRole
	})

	// 2. Create test group
	t.Run("Create Group", func(t *testing.T) {
		groupName := "test-group-for-roles"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateGroup(ctx, realmName, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		extractedID := keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, extractedID)
		groupID = &extractedID
	})

	// 3. Get initial realm role mappings (should be empty)
	t.Run("Get Initial Mappings", func(t *testing.T) {
		require.NotNil(t, groupID)

		roles, resp, err := c.Groups.GetRealmRoleMappings(ctx, realmName, *groupID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		// May be nil or empty
		if roles != nil {
			require.Equal(t, 0, len(roles), "Should have no realm role mappings initially")
		}
	})

	// 4. Add realm role mapping
	t.Run("Add Role Mappings", func(t *testing.T) {
		require.NotNil(t, groupID)
		require.NotNil(t, testRole)

		roles := []keycloakv2.RoleRepresentation{*testRole}
		resp, err := c.Groups.AddRealmRoleMappings(ctx, realmName, *groupID, roles)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	// 5. Get realm role mappings and verify
	t.Run("Get Realm Role Mappings", func(t *testing.T) {
		require.NotNil(t, groupID)

		roles, resp, err := c.Groups.GetRealmRoleMappings(ctx, realmName, *groupID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, roles)
		require.Equal(t, 1, len(roles), "Should have exactly one realm role mapping")
		require.Equal(t, *testRole.Name, *roles[0].Name)
	})

	// 6. Get all role mappings (realm + client)
	t.Run("Get All Mappings", func(t *testing.T) {
		require.NotNil(t, groupID)

		mappings, resp, err := c.Groups.GetRoleMappings(ctx, realmName, *groupID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, mappings)
		require.NotNil(t, mappings.RealmMappings)
		require.Equal(t, 1, len(*mappings.RealmMappings), "Should have one realm role in mappings")
	})

	// 7. Delete realm role mapping
	t.Run("Delete Role Mappings", func(t *testing.T) {
		require.NotNil(t, groupID)
		require.NotNil(t, testRole)

		roles := []keycloakv2.RoleRepresentation{*testRole}
		resp, err := c.Groups.DeleteRealmRoleMappings(ctx, realmName, *groupID, roles)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	// 8. Verify deletion
	t.Run("Verify Deletion", func(t *testing.T) {
		require.NotNil(t, groupID)

		roles, resp, err := c.Groups.GetRealmRoleMappings(ctx, realmName, *groupID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		// Should be nil or empty
		if roles != nil {
			require.Equal(t, 0, len(roles), "Should have no realm role mappings after deletion")
		}
	})
}

func TestGroupsClient_ClientRoleMappings(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-grp-client-roles-%d", time.Now().UnixNano())
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

	var groupID *string

	var testClientID *string

	var testClientUUID *string

	var testClientRole *keycloakv2.RoleRepresentation

	// 1. Create test client
	t.Run("Create Test Client", func(t *testing.T) {
		clientId := fmt.Sprintf("test-client-%d", time.Now().UnixNano())
		clientName := "Test Client for Role Mapping"
		publicClient := false
		client := keycloakv2.ClientRepresentation{
			ClientId:     &clientId,
			Name:         &clientName,
			PublicClient: &publicClient,
		}

		resp, err := c.Clients.CreateClient(ctx, realmName, client)
		require.NoError(t, err)
		require.NotNil(t, resp)

		testClientID = &clientId
	})

	// 2. Get client UUID
	t.Run("Get Client UUID", func(t *testing.T) {
		require.NotNil(t, testClientID)

		// Query for the client by clientId
		params := &keycloakv2.GetClientsParams{
			ClientId: testClientID,
		}

		clients, resp, err := c.Clients.GetClients(ctx, realmName, params)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 1, len(clients), "Should find exactly one client")
		require.NotNil(t, clients[0].Id)

		testClientUUID = clients[0].Id
	})

	// 3. Create client role
	t.Run("Create Client Role", func(t *testing.T) {
		require.NotNil(t, testClientUUID)

		roleName := "test-client-role"
		roleDescription := "Test client role for group mapping"
		role := keycloakv2.RoleRepresentation{
			Name:        &roleName,
			Description: &roleDescription,
		}

		resp, err := c.Clients.CreateClientRole(ctx, realmName, *testClientUUID, role)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Fetch the created role to get its ID
		createdRole, resp, err := c.Clients.GetClientRole(ctx, realmName, *testClientUUID, roleName)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, createdRole)
		testClientRole = createdRole
	})

	// 4. Create test group
	t.Run("Create Group", func(t *testing.T) {
		groupName := "test-group-for-client-roles"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateGroup(ctx, realmName, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		extractedID := keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, extractedID)
		groupID = &extractedID
	})

	// 5. Get initial client role mappings (should be empty)
	t.Run("Get Initial Client Mappings", func(t *testing.T) {
		require.NotNil(t, groupID)
		require.NotNil(t, testClientUUID)

		roles, resp, err := c.Groups.GetClientRoleMappings(ctx, realmName, *groupID, *testClientUUID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		// May be nil or empty
		if roles != nil {
			require.Equal(t, 0, len(roles), "Should have no client role mappings initially")
		}
	})

	// 6. Add client role mapping
	t.Run("Add Client Role Mappings", func(t *testing.T) {
		require.NotNil(t, groupID)
		require.NotNil(t, testClientUUID)
		require.NotNil(t, testClientRole)

		roles := []keycloakv2.RoleRepresentation{*testClientRole}
		resp, err := c.Groups.AddClientRoleMappings(ctx, realmName, *groupID, *testClientUUID, roles)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	// 7. Get client role mappings and verify
	t.Run("Get Client Role Mappings", func(t *testing.T) {
		require.NotNil(t, groupID)
		require.NotNil(t, testClientUUID)

		roles, resp, err := c.Groups.GetClientRoleMappings(ctx, realmName, *groupID, *testClientUUID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, roles)
		require.Equal(t, 1, len(roles), "Should have exactly one client role mapping")
		require.Equal(t, *testClientRole.Name, *roles[0].Name)
	})

	// 8. Delete client role mapping
	t.Run("Delete Client Role Mappings", func(t *testing.T) {
		require.NotNil(t, groupID)
		require.NotNil(t, testClientUUID)
		require.NotNil(t, testClientRole)

		roles := []keycloakv2.RoleRepresentation{*testClientRole}
		resp, err := c.Groups.DeleteClientRoleMappings(ctx, realmName, *groupID, *testClientUUID, roles)
		require.NoError(t, err)
		require.NotNil(t, resp)
	})

	// 9. Verify deletion
	t.Run("Verify Deletion", func(t *testing.T) {
		require.NotNil(t, groupID)
		require.NotNil(t, testClientUUID)

		roles, resp, err := c.Groups.GetClientRoleMappings(ctx, realmName, *groupID, *testClientUUID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		// Should be nil or empty
		if roles != nil {
			require.Equal(t, 0, len(roles), "Should have no client role mappings after deletion")
		}
	})
}
