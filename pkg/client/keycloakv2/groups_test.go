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

func TestGroupsClient_FindGroupByName(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-find-grp-%d", time.Now().UnixNano())
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

	// Test: Find non-existent group
	t.Run("Find Non-Existent Group", func(t *testing.T) {
		group, resp, err := c.Groups.FindGroupByName(ctx, realmName, "non-existent-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, group, "Should return nil for non-existent group")
	})

	// Create multiple groups with similar names to test exact matching
	groupNames := []string{"test-group", "test-group-2", "test-group-prefix"}
	createdGroups := make(map[string]string) // name -> ID

	for _, name := range groupNames {
		gName := name
		group := keycloakv2.GroupRepresentation{
			Name: &gName,
		}
		resp, err := c.Groups.CreateGroup(ctx, realmName, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		extractedID := keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, extractedID)
		createdGroups[name] = extractedID
	}

	// Test: Find existing group by exact name
	t.Run("Find Existing Group By Exact Name", func(t *testing.T) {
		group, resp, err := c.Groups.FindGroupByName(ctx, realmName, "test-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group, "Should find the group")
		require.NotNil(t, group.Name)
		require.Equal(t, "test-group", *group.Name)
		require.NotNil(t, group.Id)
		require.Equal(t, createdGroups["test-group"], *group.Id)
	})

	// Test: Exact matching - should not return partial matches
	t.Run("Exact Match Verification", func(t *testing.T) {
		group, resp, err := c.Groups.FindGroupByName(ctx, realmName, "test-group-2")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group, "Should find test-group-2")
		require.Equal(t, "test-group-2", *group.Name)
		require.Equal(t, createdGroups["test-group-2"], *group.Id)
	})

	// Test: Find group with prefix name
	t.Run("Find Group With Prefix", func(t *testing.T) {
		group, resp, err := c.Groups.FindGroupByName(ctx, realmName, "test-group-prefix")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group)
		require.Equal(t, "test-group-prefix", *group.Name)
		require.Equal(t, createdGroups["test-group-prefix"], *group.Id)
	})

	// Test: Case sensitivity
	t.Run("Case Sensitivity", func(t *testing.T) {
		group, resp, err := c.Groups.FindGroupByName(ctx, realmName, "TEST-GROUP")
		require.NoError(t, err)
		require.NotNil(t, resp)
		// Keycloak search is case-insensitive, but we check exact name match in the code
		if group != nil {
			// If found, name should not match (case sensitive)
			require.NotEqual(t, "TEST-GROUP", *group.Name)
		}
	})
}

func TestGroupsClient_FindChildGroupByName(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-find-child-%d", time.Now().UnixNano())
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

	var parentGroupID string

	// Create parent group
	t.Run("Create Parent Group", func(t *testing.T) {
		groupName := "parent-group"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateGroup(ctx, realmName, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		parentGroupID = keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, parentGroupID)
	})

	// Test: Find non-existent child group
	t.Run("Find Non-Existent Child Group", func(t *testing.T) {
		group, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, parentGroupID, "non-existent-child")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, group, "Should return nil for non-existent child group")
	})

	// Create multiple child groups
	childNames := []string{"child-1", "child-2", "child-1-prefix"}
	createdChildren := make(map[string]string) // name -> ID

	for _, name := range childNames {
		cName := name
		child := keycloakv2.GroupRepresentation{
			Name: &cName,
		}
		resp, err := c.Groups.CreateChildGroup(ctx, realmName, parentGroupID, child)
		require.NoError(t, err)
		require.NotNil(t, resp)

		extractedID := keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, extractedID)
		createdChildren[name] = extractedID
	}

	// Test: Find existing child by exact name
	t.Run("Find Existing Child By Exact Name", func(t *testing.T) {
		child, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, parentGroupID, "child-1")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, child, "Should find the child group")
		require.NotNil(t, child.Name)
		require.Equal(t, "child-1", *child.Name)
		require.NotNil(t, child.Id)
		require.Equal(t, createdChildren["child-1"], *child.Id)
	})

	// Test: Exact matching for child groups
	t.Run("Exact Match Verification For Child", func(t *testing.T) {
		child, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, parentGroupID, "child-2")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, child, "Should find child-2")
		require.Equal(t, "child-2", *child.Name)
		require.Equal(t, createdChildren["child-2"], *child.Id)
	})

	// Test: Find child with prefix name
	t.Run("Find Child With Prefix", func(t *testing.T) {
		child, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, parentGroupID, "child-1-prefix")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, child)
		require.Equal(t, "child-1-prefix", *child.Name)
		require.Equal(t, createdChildren["child-1-prefix"], *child.Id)
	})

	// Test: Find child with non-existent parent
	t.Run("Find Child With Non-Existent Parent", func(t *testing.T) {
		_, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, "non-existent-parent-id", "child-1")
		require.Error(t, err)
		require.NotNil(t, resp)
		require.True(t, keycloakv2.IsNotFound(err), "Should return 404 for non-existent parent")
	})
}

func TestGroupsClient_FindGroupByName_MultiLevel(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-multilevel-%d", time.Now().UnixNano())
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

	// Create a hierarchy: root-group -> level1-group -> level2-group -> level3-group
	var rootGroupID, level1GroupID, level2GroupID, level3GroupID string

	// Level 0: Create root-level group
	t.Run("Create Root Level Group", func(t *testing.T) {
		groupName := "root-group"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateGroup(ctx, realmName, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		rootGroupID = keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, rootGroupID)
	})

	// Level 1: Create first level child
	t.Run("Create Level 1 Child Group", func(t *testing.T) {
		groupName := "level1-group"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateChildGroup(ctx, realmName, rootGroupID, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		level1GroupID = keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, level1GroupID)
	})

	// Level 2: Create second level child
	t.Run("Create Level 2 Child Group", func(t *testing.T) {
		groupName := "level2-group"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateChildGroup(ctx, realmName, level1GroupID, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		level2GroupID = keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, level2GroupID)
	})

	// Level 3: Create third level child
	t.Run("Create Level 3 Child Group", func(t *testing.T) {
		groupName := "level3-group"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateChildGroup(ctx, realmName, level2GroupID, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		level3GroupID = keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, level3GroupID)
	})

	// Test: Find root-level group (Level 0)
	t.Run("Find Root Level Group", func(t *testing.T) {
		group, resp, err := c.Groups.FindGroupByName(ctx, realmName, "root-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group, "Should find root-level group")
		require.Equal(t, "root-group", *group.Name)
		require.Equal(t, rootGroupID, *group.Id)
	})

	// Test: Find level 1 child within root group
	t.Run("Find Level 1 Child Group", func(t *testing.T) {
		group, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, rootGroupID, "level1-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group, "Should find level 1 child group")
		require.Equal(t, "level1-group", *group.Name)
		require.Equal(t, level1GroupID, *group.Id)
	})

	// Test: Find level 2 child within level 1 group
	t.Run("Find Level 2 Child Group", func(t *testing.T) {
		group, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, level1GroupID, "level2-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group, "Should find level 2 child group")
		require.Equal(t, "level2-group", *group.Name)
		require.Equal(t, level2GroupID, *group.Id)
	})

	// Test: Find level 3 child within level 2 group
	t.Run("Find Level 3 Child Group", func(t *testing.T) {
		group, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, level2GroupID, "level3-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group, "Should find level 3 child group")
		require.Equal(t, "level3-group", *group.Name)
		require.Equal(t, level3GroupID, *group.Id)
	})

	// Test: Cannot find level 2 group directly in root group (not a direct child)
	t.Run("Cannot Find Non-Direct Child", func(t *testing.T) {
		group, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, rootGroupID, "level2-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, group, "Should not find level2-group in root group (it's a grandchild)")
	})

	// Test: Cannot find level 3 group directly in root group
	t.Run("Cannot Find Grandchild In Root", func(t *testing.T) {
		group, resp, err := c.Groups.FindChildGroupByName(ctx, realmName, rootGroupID, "level3-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, group, "Should not find level3-group in root group")
	})

	// Test: Verify all groups can be retrieved at their correct levels
	t.Run("Verify Group Hierarchy Integrity", func(t *testing.T) {
		// Get root group's children - should only have level1
		children, resp, err := c.Groups.GetChildGroups(ctx, realmName, rootGroupID, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 1, len(children), "Root should have exactly 1 direct child")
		require.Equal(t, "level1-group", *children[0].Name)

		// Get level1 group's children - should only have level2
		children, resp, err = c.Groups.GetChildGroups(ctx, realmName, level1GroupID, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 1, len(children), "Level1 should have exactly 1 direct child")
		require.Equal(t, "level2-group", *children[0].Name)

		// Get level2 group's children - should only have level3
		children, resp, err = c.Groups.GetChildGroups(ctx, realmName, level2GroupID, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 1, len(children), "Level2 should have exactly 1 direct child")
		require.Equal(t, "level3-group", *children[0].Name)

		// Get level3 group's children - should have no children
		children, resp, err = c.Groups.GetChildGroups(ctx, realmName, level3GroupID, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Equal(t, 0, len(children), "Level3 should have no children")
	})
}

func TestGroupsClient_GetGroupByPath(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-grp-path-%d", time.Now().UnixNano())
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

	// Build hierarchy: top-group -> mid-group -> bottom-group
	var topGroupID, midGroupID, bottomGroupID string

	t.Run("Create Top-Level Group", func(t *testing.T) {
		groupName := "top-group"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateGroup(ctx, realmName, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		topGroupID = keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, topGroupID)
	})

	t.Run("Create Mid-Level Group", func(t *testing.T) {
		groupName := "mid-group"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateChildGroup(ctx, realmName, topGroupID, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		midGroupID = keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, midGroupID)
	})

	t.Run("Create Bottom-Level Group", func(t *testing.T) {
		groupName := "bottom-group"
		group := keycloakv2.GroupRepresentation{
			Name: &groupName,
		}

		resp, err := c.Groups.CreateChildGroup(ctx, realmName, midGroupID, group)
		require.NoError(t, err)
		require.NotNil(t, resp)

		bottomGroupID = keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, bottomGroupID)
	})

	// Test: Get top-level group by path
	t.Run("Get Top-Level Group By Path", func(t *testing.T) {
		group, resp, err := c.Groups.GetGroupByPath(ctx, realmName, "/top-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group)
		require.NotNil(t, group.Name)
		require.Equal(t, "top-group", *group.Name)
		require.NotNil(t, group.Id)
		require.Equal(t, topGroupID, *group.Id)
	})

	// Test: Get nested group by path
	t.Run("Get Nested Group By Path", func(t *testing.T) {
		group, resp, err := c.Groups.GetGroupByPath(ctx, realmName, "/top-group/mid-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group)
		require.NotNil(t, group.Name)
		require.Equal(t, "mid-group", *group.Name)
		require.NotNil(t, group.Id)
		require.Equal(t, midGroupID, *group.Id)
	})

	// Test: Get deeply nested group by path
	t.Run("Get Deeply Nested Group By Path", func(t *testing.T) {
		group, resp, err := c.Groups.GetGroupByPath(ctx, realmName, "/top-group/mid-group/bottom-group")
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, group)
		require.NotNil(t, group.Name)
		require.Equal(t, "bottom-group", *group.Name)
		require.NotNil(t, group.Id)
		require.Equal(t, bottomGroupID, *group.Id)
	})

	// Test: Non-existent path returns error
	t.Run("Non-Existent Path", func(t *testing.T) {
		group, resp, err := c.Groups.GetGroupByPath(ctx, realmName, "/non-existent-group")
		require.Error(t, err)
		require.True(t, keycloakv2.IsNotFound(err), "Should return 404 for non-existent path")
		require.Nil(t, group)
		require.NotNil(t, resp)
	})

	// Test: Non-existent nested path returns error
	t.Run("Non-Existent Nested Path", func(t *testing.T) {
		group, resp, err := c.Groups.GetGroupByPath(ctx, realmName, "/top-group/non-existent")
		require.Error(t, err)
		require.True(t, keycloakv2.IsNotFound(err), "Should return 404 for non-existent nested path")
		require.Nil(t, group)
		require.NotNil(t, resp)
	})

	// Test: Partial valid path with invalid leaf returns error
	t.Run("Partial Valid Path With Invalid Leaf", func(t *testing.T) {
		group, resp, err := c.Groups.GetGroupByPath(ctx, realmName, "/top-group/mid-group/non-existent")
		require.Error(t, err)
		require.True(t, keycloakv2.IsNotFound(err), "Should return 404 for invalid leaf in path")
		require.Nil(t, group)
		require.NotNil(t, resp)
	})
}
