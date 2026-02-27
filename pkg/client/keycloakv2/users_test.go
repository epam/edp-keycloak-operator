package keycloakv2_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestUsersClient_UserProfile_CRUD(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-user-profile-%d", time.Now().UnixNano())
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

	// 1. Get default user profile configuration
	originalProfile, resp, err := c.Users.GetUsersProfile(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, originalProfile)
	require.NotNil(t, originalProfile.Attributes)
	require.Greater(t, len(*originalProfile.Attributes), 0, "Default profile should have attributes")

	// 2. Update user profile with a custom attribute
	// Add a custom attribute to the profile
	customAttrName := "customTestAttribute"
	customDisplayName := "Custom Test Attribute"

	// Create permissions structure
	editPermissions := []string{"admin", "user"}
	viewPermissions := []string{"admin", "user"}
	permissions := keycloakv2.UserProfileAttributePermissions{
		Edit: &editPermissions,
		View: &viewPermissions,
	}

	// Create the custom attribute
	customAttribute := keycloakv2.UserProfileAttribute{
		Name:        &customAttrName,
		DisplayName: &customDisplayName,
		Permissions: &permissions,
	}

	// Append to existing attributes
	updatedAttributes := append(*originalProfile.Attributes, customAttribute)
	originalProfile.Attributes = &updatedAttributes

	// Update the profile
	updatedProfile, resp, err := c.Users.UpdateUsersProfile(ctx, realmName, *originalProfile)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, updatedProfile)
	require.NotNil(t, updatedProfile.Attributes)

	// 3. Verify the update by getting the profile again
	profile, resp, err := c.Users.GetUsersProfile(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, profile)
	require.NotNil(t, profile.Attributes)

	// Check that the custom attribute exists
	customAttrFound := false

	for _, attr := range *profile.Attributes {
		if attr.Name != nil && *attr.Name == "customTestAttribute" {
			customAttrFound = true

			require.NotNil(t, attr.DisplayName)
			require.Equal(t, "Custom Test Attribute", *attr.DisplayName)

			break
		}
	}

	require.True(t, customAttrFound, "Custom attribute should be present in the profile")
}

func TestUsersClient_GetUsersProfile_NotFound(t *testing.T) {
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

	// Test getting user profile for a non-existent realm
	profile, resp, err := c.Users.GetUsersProfile(ctx, "nonexistent-realm-12345")
	require.Error(t, err)
	require.True(
		t,
		keycloakv2.IsNotFound(err),
		fmt.Sprintf("Should return %d error for non-existent realm", http.StatusNotFound),
	)
	require.Nil(t, profile, "Profile should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestUsersClient_FindUserByUsername(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-find-user-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	username := "test-find-user"
	email := "test-find-user@example.com"
	user := keycloakv2.UserRepresentation{
		Username: &username,
		Email:    &email,
		Enabled:  &enabled,
	}

	// Create a user
	resp, err = c.Users.CreateUser(ctx, realmName, user)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Find the user by exact username
	found, resp, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, found)
	require.NotNil(t, found.Username)
	require.Equal(t, username, *found.Username)

	// Partial match should return nil (exact=true)
	notFound, resp, err := c.Users.FindUserByUsername(ctx, realmName, "test-find")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Nil(t, notFound)
}

func TestUsersClient_FindUserByUsername_NotFound(t *testing.T) {
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

	// Non-existent realm
	user, resp, err := c.Users.FindUserByUsername(ctx, "nonexistent-realm-12345", "anyuser")
	require.Error(t, err)
	require.True(
		t,
		keycloakv2.IsNotFound(err),
		fmt.Sprintf("Should return %d error for non-existent realm", http.StatusNotFound),
	)
	require.Nil(t, user)
	require.NotNil(t, resp)
}

func TestUsersClient_CreateUser(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-create-user-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	username := "new-test-user"
	email := "new-test-user@example.com"
	user := keycloakv2.UserRepresentation{
		Username: &username,
		Email:    &email,
		Enabled:  &enabled,
	}

	resp, err = c.Users.CreateUser(ctx, realmName, user)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// Verify user was created
	found, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, username, *found.Username)
}

func TestUsersClient_CreateUser_Conflict(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-create-user-conflict-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	_, err = c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)

	username := "duplicate-user"
	user := keycloakv2.UserRepresentation{
		Username: &username,
		Enabled:  &enabled,
	}

	resp, err := c.Users.CreateUser(ctx, realmName, user)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Create the same user again — should conflict
	resp, err = c.Users.CreateUser(ctx, realmName, user)
	require.Error(t, err)
	require.True(t, keycloakv2.IsConflict(err), "Should return 409 Conflict for duplicate user")
	require.NotNil(t, resp)
}

func TestUsersClient_GetUserRealmRoleMappings(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-user-role-mappings-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create a role
	roleName := "mapping-test-role"
	_, err = c.Roles.CreateRealmRole(ctx, realmName, keycloakv2.RoleRepresentation{Name: &roleName})
	require.NoError(t, err)

	// Create a user
	username := "role-mapping-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakv2.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	// No roles yet
	roles, resp, err := c.Users.GetUserRealmRoleMappings(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)

	for _, r := range roles {
		if r.Name != nil && *r.Name == roleName {
			t.Fatal("Role should not be mapped yet")
		}
	}

	// Add the role
	role, _, err := c.Roles.GetRealmRole(ctx, realmName, roleName)
	require.NoError(t, err)
	require.NotNil(t, role)

	addResp, err := c.Users.AddUserRealmRoles(ctx, realmName, *user.Id, []keycloakv2.RoleRepresentation{*role})
	require.NoError(t, err)
	require.NotNil(t, addResp)

	// Verify mapping
	roles, resp, err = c.Users.GetUserRealmRoleMappings(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)

	found := false

	for _, r := range roles {
		if r.Name != nil && *r.Name == roleName {
			found = true

			break
		}
	}

	require.True(t, found, "Role should appear in user realm role mappings after adding")
}

func TestUsersClient_AddUserRealmRoles_UserNotFound(t *testing.T) {
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

	roleName := "uma_authorization"
	role, _, err := c.Roles.GetRealmRole(ctx, keycloakv2.MasterRealm, roleName)
	require.NoError(t, err)
	require.NotNil(t, role)

	resp, err := c.Users.AddUserRealmRoles(
		ctx,
		keycloakv2.MasterRealm,
		"nonexistent-user-id-12345",
		[]keycloakv2.RoleRepresentation{*role},
	)
	require.Error(t, err)
	require.NotNil(t, resp)
}

func TestUsersClient_GetUserGroups(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-user-groups-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create a user
	username := "user-groups-test"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakv2.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	// Initially user should have no groups
	groups, resp, err := c.Users.GetUserGroups(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, groups)

	// Create a group and add user to it
	groupName := "user-test-group"
	groupResp, err := c.Groups.CreateGroup(ctx, realmName, keycloakv2.GroupRepresentation{Name: &groupName})
	require.NoError(t, err)
	require.NotNil(t, groupResp)

	groupID := keycloakv2.GetResourceIDFromResponse(groupResp)
	require.NotEmpty(t, groupID)

	_, err = c.Users.AddUserToGroup(ctx, realmName, *user.Id, groupID)
	require.NoError(t, err)

	// Now user should have one group
	groups, resp, err = c.Users.GetUserGroups(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, groups, 1)
	require.NotNil(t, groups[0].Name)
	require.Equal(t, groupName, *groups[0].Name)
}

func TestUsersClient_GetUserGroups_NonExistentUser(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	_, resp, err := c.Users.GetUserGroups(context.Background(), keycloakv2.MasterRealm, "nonexistent-user-id-12345")
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_AddUserToGroup(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-add-user-group-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create a user
	username := "add-group-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakv2.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	// Create two groups
	group1Name := "add-user-group-1"
	group2Name := "add-user-group-2"

	resp, err := c.Groups.CreateGroup(ctx, realmName, keycloakv2.GroupRepresentation{Name: &group1Name})
	require.NoError(t, err)

	group1ID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, group1ID)

	resp, err = c.Groups.CreateGroup(ctx, realmName, keycloakv2.GroupRepresentation{Name: &group2Name})
	require.NoError(t, err)

	group2ID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, group2ID)

	// Add user to first group
	addResp, err := c.Users.AddUserToGroup(ctx, realmName, *user.Id, group1ID)
	require.NoError(t, err)
	require.NotNil(t, addResp)

	// Add user to second group
	addResp, err = c.Users.AddUserToGroup(ctx, realmName, *user.Id, group2ID)
	require.NoError(t, err)
	require.NotNil(t, addResp)

	// Verify user is in both groups
	groups, _, err := c.Users.GetUserGroups(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.Len(t, groups, 2)

	groupNames := make(map[string]bool)

	for _, g := range groups {
		if g.Name != nil {
			groupNames[*g.Name] = true
		}
	}

	require.True(t, groupNames[group1Name], "User should be in group 1")
	require.True(t, groupNames[group2Name], "User should be in group 2")

	// Adding to the same group again should be idempotent (no error)
	addResp, err = c.Users.AddUserToGroup(ctx, realmName, *user.Id, group1ID)
	require.NoError(t, err)
	require.NotNil(t, addResp)
}

func TestUsersClient_AddUserToGroup_NonExistentUser(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Users.AddUserToGroup(
		context.Background(), keycloakv2.MasterRealm, "nonexistent-user-id", "nonexistent-group-id",
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_RemoveUserFromGroup(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-remove-user-group-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create a user
	username := "remove-group-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakv2.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	// Create a group
	groupName := "remove-user-group"
	resp, err := c.Groups.CreateGroup(ctx, realmName, keycloakv2.GroupRepresentation{Name: &groupName})
	require.NoError(t, err)

	groupID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, groupID)

	// Add user to group
	_, err = c.Users.AddUserToGroup(ctx, realmName, *user.Id, groupID)
	require.NoError(t, err)

	// Verify user is in the group
	groups, _, err := c.Users.GetUserGroups(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, groupName, *groups[0].Name)

	// Remove user from group
	removeResp, err := c.Users.RemoveUserFromGroup(ctx, realmName, *user.Id, groupID)
	require.NoError(t, err)
	require.NotNil(t, removeResp)

	// Verify user is no longer in the group
	groups, _, err = c.Users.GetUserGroups(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.Empty(t, groups)
}

func TestUsersClient_RemoveUserFromGroup_NonExistentUser(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Users.RemoveUserFromGroup(
		context.Background(),
		keycloakv2.MasterRealm,
		"nonexistent-user-id",
		"nonexistent-group-id",
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_UpdateUsersProfile_NotFound(t *testing.T) {
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

	// Create a minimal user profile config for testing
	customAttrName := "testAttribute"
	customDisplayName := "Test Attribute"
	editPermissions := []string{"admin"}
	viewPermissions := []string{"admin"}

	permissions := keycloakv2.UserProfileAttributePermissions{
		Edit: &editPermissions,
		View: &viewPermissions,
	}

	customAttribute := keycloakv2.UserProfileAttribute{
		Name:        &customAttrName,
		DisplayName: &customDisplayName,
		Permissions: &permissions,
	}

	attributes := []keycloakv2.UserProfileAttribute{customAttribute}
	userProfile := keycloakv2.UserProfileConfig{
		Attributes: &attributes,
	}

	// Test updating user profile for a non-existent realm
	profile, resp, err := c.Users.UpdateUsersProfile(ctx, "nonexistent-realm-12345", userProfile)
	require.Error(t, err)
	require.True(
		t,
		keycloakv2.IsNotFound(err),
		fmt.Sprintf("Should return %d error for non-existent realm", http.StatusNotFound),
	)
	require.Nil(t, profile, "Profile should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}
