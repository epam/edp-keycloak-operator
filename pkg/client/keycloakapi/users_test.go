package keycloakapi_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

func TestUsersClient_UserProfile_CRUD(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-user-profile-%d", time.Now().UnixNano())
	enabled := true

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm first
	realm := keycloakapi.RealmRepresentation{
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
	permissions := keycloakapi.UserProfileAttributePermissions{
		Edit: &editPermissions,
		View: &viewPermissions,
	}

	// Create the custom attribute
	customAttribute := keycloakapi.UserProfileAttribute{
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

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Test getting user profile for a non-existent realm
	profile, resp, err := c.Users.GetUsersProfile(ctx, "nonexistent-realm-12345")
	require.Error(t, err)
	require.True(
		t,
		keycloakapi.IsNotFound(err),
		fmt.Sprintf("Should return %d error for non-existent realm", http.StatusNotFound),
	)
	require.Nil(t, profile, "Profile should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestUsersClient_FindUserByUsername(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-find-user-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	realm := keycloakapi.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	username := "test-find-user"
	email := "test-find-user@example.com"
	user := keycloakapi.UserRepresentation{
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

	// Partial match should return ErrNotFound (exact=true)
	_, resp, err = c.Users.FindUserByUsername(ctx, realmName, "test-find")
	require.True(t, keycloakapi.IsNotFound(err), "partial match should return ErrNotFound")
	require.NotNil(t, resp)
}

func TestUsersClient_FindUserByUsername_NotFound(t *testing.T) {
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

	// Non-existent realm
	user, resp, err := c.Users.FindUserByUsername(ctx, "nonexistent-realm-12345", "anyuser")
	require.Error(t, err)
	require.True(
		t,
		keycloakapi.IsNotFound(err),
		fmt.Sprintf("Should return %d error for non-existent realm", http.StatusNotFound),
	)
	require.Nil(t, user)
	require.NotNil(t, resp)
}

func TestUsersClient_CreateUser(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-create-user-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	realm := keycloakapi.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	username := "new-test-user"
	email := "new-test-user@example.com"
	user := keycloakapi.UserRepresentation{
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

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-create-user-conflict-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	realm := keycloakapi.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	_, err = c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)

	username := "duplicate-user"
	user := keycloakapi.UserRepresentation{
		Username: &username,
		Enabled:  &enabled,
	}

	resp, err := c.Users.CreateUser(ctx, realmName, user)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Create the same user again — should conflict
	resp, err = c.Users.CreateUser(ctx, realmName, user)
	require.Error(t, err)
	require.True(t, keycloakapi.IsConflict(err), "Should return 409 Conflict for duplicate user")
	require.NotNil(t, resp)
}

func TestUsersClient_GetUserRealmRoleMappings(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-user-role-mappings-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create a role
	roleName := "mapping-test-role"
	_, err = c.Roles.CreateRealmRole(ctx, realmName, keycloakapi.RoleRepresentation{Name: &roleName})
	require.NoError(t, err)

	// Create a user
	username := "role-mapping-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{Username: &username, Enabled: &enabled})
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

	addResp, err := c.Users.AddUserRealmRoles(ctx, realmName, *user.Id, []keycloakapi.RoleRepresentation{*role})
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

	// Delete the role mapping
	deleteResp, err := c.Users.DeleteUserRealmRoles(ctx, realmName, *user.Id, []keycloakapi.RoleRepresentation{*role})
	require.NoError(t, err)
	require.NotNil(t, deleteResp)

	// Verify role is no longer mapped
	roles, resp, err = c.Users.GetUserRealmRoleMappings(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)

	for _, r := range roles {
		if r.Name != nil && *r.Name == roleName {
			t.Fatal("Role should no longer be mapped after deletion")
		}
	}
}

func TestUsersClient_AddUserRealmRoles_UserNotFound(t *testing.T) {
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

	roleName := "uma_authorization"
	role, _, err := c.Roles.GetRealmRole(ctx, keycloakapi.MasterRealm, roleName)
	require.NoError(t, err)
	require.NotNil(t, role)

	resp, err := c.Users.AddUserRealmRoles(
		ctx,
		keycloakapi.MasterRealm,
		"nonexistent-user-id-12345",
		[]keycloakapi.RoleRepresentation{*role},
	)
	require.Error(t, err)
	require.NotNil(t, resp)
}

func TestUsersClient_GetUserGroups(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-user-groups-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create a user
	username := "user-groups-test"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{Username: &username, Enabled: &enabled})
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
	groupResp, err := c.Groups.CreateGroup(ctx, realmName, keycloakapi.GroupRepresentation{Name: &groupName})
	require.NoError(t, err)
	require.NotNil(t, groupResp)

	groupID := keycloakapi.GetResourceIDFromResponse(groupResp)
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

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	_, resp, err := c.Users.GetUserGroups(context.Background(), keycloakapi.MasterRealm, "nonexistent-user-id-12345")
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_AddUserToGroup(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-add-user-group-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create a user
	username := "add-group-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	// Create two groups
	group1Name := "add-user-group-1"
	group2Name := "add-user-group-2"

	resp, err := c.Groups.CreateGroup(ctx, realmName, keycloakapi.GroupRepresentation{Name: &group1Name})
	require.NoError(t, err)

	group1ID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, group1ID)

	resp, err = c.Groups.CreateGroup(ctx, realmName, keycloakapi.GroupRepresentation{Name: &group2Name})
	require.NoError(t, err)

	group2ID := keycloakapi.GetResourceIDFromResponse(resp)
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

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Users.AddUserToGroup(
		context.Background(), keycloakapi.MasterRealm, "nonexistent-user-id", "nonexistent-group-id",
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_RemoveUserFromGroup(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-remove-user-group-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create a user
	username := "remove-group-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	// Create a group
	groupName := "remove-user-group"
	resp, err := c.Groups.CreateGroup(ctx, realmName, keycloakapi.GroupRepresentation{Name: &groupName})
	require.NoError(t, err)

	groupID := keycloakapi.GetResourceIDFromResponse(resp)
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

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Users.RemoveUserFromGroup(
		context.Background(),
		keycloakapi.MasterRealm,
		"nonexistent-user-id",
		"nonexistent-group-id",
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_UpdateAndDeleteUser(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-upd-del-user-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	username := "update-delete-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	// UpdateUser — change email
	newEmail := "updated@example.com"
	updatedUser := *user
	updatedUser.Email = &newEmail

	resp, err := c.Users.UpdateUser(ctx, realmName, *user.Id, updatedUser)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify update
	found, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.NotNil(t, found.Email)
	require.Equal(t, newEmail, *found.Email)

	// DeleteUser
	resp, err = c.Users.DeleteUser(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify deletion
	_, _, err = c.Users.FindUserByUsername(ctx, realmName, username)
	require.True(t, keycloakapi.IsNotFound(err), "user should be gone after deletion")
}

func TestUsersClient_UpdateUser_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Users.UpdateUser(
		context.Background(),
		keycloakapi.MasterRealm,
		"nonexistent-user-id-12345",
		keycloakapi.UserRepresentation{},
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_SetUserPassword(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-set-pwd-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	username := "password-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	credType := "password"
	credValue := "s3cret!"
	temporary := false
	resp, err := c.Users.SetUserPassword(ctx, realmName, *user.Id, keycloakapi.CredentialRepresentation{
		Type:      &credType,
		Value:     &credValue,
		Temporary: &temporary,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestUsersClient_UserClientRoleMappings(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-user-cli-roles-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create client
	clientID := fmt.Sprintf("test-client-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	clientResp, err := c.Clients.CreateClient(ctx, realmName, keycloakapi.ClientRepresentation{
		ClientId: &clientID,
		Protocol: &protocol,
	})
	require.NoError(t, err)

	clientUUID := keycloakapi.GetResourceIDFromResponse(clientResp)
	require.NotEmpty(t, clientUUID)

	// Create client role
	roleName := "user-client-role"
	_, err = c.Clients.CreateClientRole(ctx, realmName, clientUUID, keycloakapi.RoleRepresentation{Name: &roleName})
	require.NoError(t, err)

	role, _, err := c.Clients.GetClientRole(ctx, realmName, clientUUID, roleName)
	require.NoError(t, err)
	require.NotNil(t, role)

	// Create user
	username := "client-role-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	// Initially no client role mappings
	roles, resp, err := c.Users.GetUserClientRoleMappings(ctx, realmName, *user.Id, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, roles)

	// Add client role
	addResp, err := c.Users.AddUserClientRoles(
		ctx, realmName, *user.Id, clientUUID, []keycloakapi.RoleRepresentation{*role},
	)
	require.NoError(t, err)
	require.NotNil(t, addResp)

	// Verify role is mapped
	roles, resp, err = c.Users.GetUserClientRoleMappings(ctx, realmName, *user.Id, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	found := false

	for _, r := range roles {
		if r.Name != nil && *r.Name == roleName {
			found = true
			break
		}
	}

	require.True(t, found, "Client role should be mapped to user")

	// Delete client role mapping
	delResp, err := c.Users.DeleteUserClientRoles(
		ctx, realmName, *user.Id, clientUUID, []keycloakapi.RoleRepresentation{*role},
	)
	require.NoError(t, err)
	require.NotNil(t, delResp)

	// Verify role is no longer mapped
	roles, resp, err = c.Users.GetUserClientRoleMappings(ctx, realmName, *user.Id, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, roles)
}

func TestUsersClient_UserFederatedIdentities(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-fed-id-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{Realm: &realmName, Enabled: &enabled})
	require.NoError(t, err)

	// Create an identity provider
	alias := fmt.Sprintf("test-idp-%d", time.Now().UnixNano())
	providerID := "github"

	_, err = c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakapi.IdentityProviderRepresentation{
		Alias:      &alias,
		ProviderId: &providerID,
		Enabled:    &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	})
	require.NoError(t, err)

	// Create user
	username := "federated-user"
	_, err = c.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{Username: &username, Enabled: &enabled})
	require.NoError(t, err)

	user, _, err := c.Users.FindUserByUsername(ctx, realmName, username)
	require.NoError(t, err)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)

	// Initially no federated identities
	identities, resp, err := c.Users.GetUserFederatedIdentities(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, identities)

	// Link federated identity
	externalUserID := "external-user-123"
	externalUsername := "external-user"
	linkResp, err := c.Users.CreateUserFederatedIdentity(
		ctx, realmName, *user.Id, alias, keycloakapi.FederatedIdentityRepresentation{
			IdentityProvider: &alias,
			UserId:           &externalUserID,
			UserName:         &externalUsername,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, linkResp)

	// Verify federated identity is linked
	identities, resp, err = c.Users.GetUserFederatedIdentities(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, identities, 1)
	require.NotNil(t, identities[0].IdentityProvider)
	require.Equal(t, alias, *identities[0].IdentityProvider)

	// Unlink federated identity
	delResp, err := c.Users.DeleteUserFederatedIdentity(ctx, realmName, *user.Id, alias)
	require.NoError(t, err)
	require.NotNil(t, delResp)

	// Verify unlinked
	identities, resp, err = c.Users.GetUserFederatedIdentities(ctx, realmName, *user.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Empty(t, identities)
}

func TestUsersClient_SetUserPassword_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	credType := "password"
	credValue := "s3cret!"
	temporary := false
	resp, err := c.Users.SetUserPassword(
		context.Background(),
		keycloakapi.MasterRealm,
		"nonexistent-user-id-12345",
		keycloakapi.CredentialRepresentation{
			Type:      &credType,
			Value:     &credValue,
			Temporary: &temporary,
		},
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_DeleteUser_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Users.DeleteUser(context.Background(), keycloakapi.MasterRealm, "nonexistent-user-id-12345")
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_DeleteUserRealmRoles_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	roleName := "uma_authorization"
	role, _, err := c.Roles.GetRealmRole(context.Background(), keycloakapi.MasterRealm, roleName)
	require.NoError(t, err)
	require.NotNil(t, role)

	resp, err := c.Users.DeleteUserRealmRoles(
		context.Background(),
		keycloakapi.MasterRealm,
		"nonexistent-user-id-12345",
		[]keycloakapi.RoleRepresentation{*role},
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_GetUserFederatedIdentities_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	_, resp, err := c.Users.GetUserFederatedIdentities(
		context.Background(), keycloakapi.MasterRealm, "nonexistent-user-id-12345",
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_CreateUserFederatedIdentity_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	externalUserID := "ext-123"
	externalUsername := "ext-user"
	alias := "nonexistent-provider"

	resp, err := c.Users.CreateUserFederatedIdentity(
		context.Background(),
		keycloakapi.MasterRealm,
		"nonexistent-user-id-12345",
		alias,
		keycloakapi.FederatedIdentityRepresentation{
			IdentityProvider: &alias,
			UserId:           &externalUserID,
			UserName:         &externalUsername,
		},
	)
	require.Error(t, err)
	require.NotNil(t, resp)
}

func TestUsersClient_DeleteUserFederatedIdentity_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Users.DeleteUserFederatedIdentity(
		context.Background(),
		keycloakapi.MasterRealm,
		"nonexistent-user-id-12345",
		"nonexistent-provider",
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_AddUserClientRoles_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Users.AddUserClientRoles(
		context.Background(),
		keycloakapi.MasterRealm,
		"nonexistent-user-id-12345",
		"nonexistent-client-id",
		[]keycloakapi.RoleRepresentation{},
	)
	require.Error(t, err)
	require.NotNil(t, resp)
}

func TestUsersClient_DeleteUserClientRoles_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Users.DeleteUserClientRoles(
		context.Background(),
		keycloakapi.MasterRealm,
		"nonexistent-user-id-12345",
		"nonexistent-client-id",
		[]keycloakapi.RoleRepresentation{},
	)
	require.Error(t, err)
	require.NotNil(t, resp)
}

func TestUsersClient_GetUserClientRoleMappings_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	_, resp, err := c.Users.GetUserClientRoleMappings(
		context.Background(),
		keycloakapi.MasterRealm,
		"nonexistent-user-id-12345",
		"nonexistent-client-id",
	)
	require.Error(t, err)
	require.NotNil(t, resp)
	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 for non-existent user")
}

func TestUsersClient_UpdateUsersProfile_NotFound(t *testing.T) {
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

	// Create a minimal user profile config for testing
	customAttrName := "testAttribute"
	customDisplayName := "Test Attribute"
	editPermissions := []string{"admin"}
	viewPermissions := []string{"admin"}

	permissions := keycloakapi.UserProfileAttributePermissions{
		Edit: &editPermissions,
		View: &viewPermissions,
	}

	customAttribute := keycloakapi.UserProfileAttribute{
		Name:        &customAttrName,
		DisplayName: &customDisplayName,
		Permissions: &permissions,
	}

	attributes := []keycloakapi.UserProfileAttribute{customAttribute}
	userProfile := keycloakapi.UserProfileConfig{
		Attributes: &attributes,
	}

	// Test updating user profile for a non-existent realm
	profile, resp, err := c.Users.UpdateUsersProfile(ctx, "nonexistent-realm-12345", userProfile)
	require.Error(t, err)
	require.True(
		t,
		keycloakapi.IsNotFound(err),
		fmt.Sprintf("Should return %d error for non-existent realm", http.StatusNotFound),
	)
	require.Nil(t, profile, "Profile should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

// newUsersTestRealm creates a Keycloak client and a fresh realm.
// The realm is automatically deleted in t.Cleanup.
func newUsersTestRealm(t *testing.T) (*keycloakapi.KeycloakClient, string) {
	t.Helper()

	keycloakURL := testutils.GetKeycloakURLOrSkip(t)

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	realmName := fmt.Sprintf("test-realm-users-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(context.Background(), keycloakapi.RealmRepresentation{
		Realm:   &realmName,
		Enabled: ptr.To(true),
	})
	require.NoError(t, err)

	return c, realmName
}

// createTestUser creates a user in the given realm and returns its UUID.
func createTestUser(
	t *testing.T, c *keycloakapi.KeycloakClient, ctx context.Context, realmName string,
) (string, string) {
	t.Helper()

	username := fmt.Sprintf("test-user-%d", time.Now().UnixNano())

	resp, err := c.Users.CreateUser(ctx, realmName, keycloakapi.UserRepresentation{
		Username: &username,
		Enabled:  ptr.To(true),
		Email:    ptr.To(fmt.Sprintf("%s@example.com", username)),
	})
	require.NoError(t, err)

	userID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, userID)

	return userID, username
}

func TestUsersClient_GetUsers(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	userID, username := createTestUser(t, c, ctx, realmName)
	_ = userID

	users, resp, err := c.Users.GetUsers(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Greater(t, len(users), 0, "should find at least one user")

	found := false

	for _, u := range users {
		if u.Username != nil && *u.Username == username {
			found = true

			break
		}
	}

	require.True(t, found, "created user should be in list")
}

func TestUsersClient_GetUser(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	userID, username := createTestUser(t, c, ctx, realmName)

	user, resp, err := c.Users.GetUser(ctx, realmName, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, user)
	require.Equal(t, username, *user.Username)
}

func TestUsersClient_GetUser_NotFound(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	_, _, err := c.Users.GetUser(ctx, realmName, "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
	require.True(t, keycloakapi.IsNotFound(err))
}

func TestUsersClient_GetUserSessions(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	userID, _ := createTestUser(t, c, ctx, realmName)

	sessions, resp, err := c.Users.GetUserSessions(ctx, realmName, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	// A freshly created user has no sessions.
	require.Empty(t, sessions)
}

func TestUsersClient_LogoutUser(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	userID, _ := createTestUser(t, c, ctx, realmName)

	resp, err := c.Users.LogoutUser(ctx, realmName, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestUsersClient_GetUserCredentials(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	userID, _ := createTestUser(t, c, ctx, realmName)

	// Set a password so that we have a credential.
	_, err := c.Users.SetUserPassword(ctx, realmName, userID, keycloakapi.CredentialRepresentation{
		Type:      ptr.To("password"),
		Value:     ptr.To("testPassword123!"),
		Temporary: ptr.To(false),
	})
	require.NoError(t, err)

	creds, resp, err := c.Users.GetUserCredentials(ctx, realmName, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Greater(t, len(creds), 0, "should have at least one credential after setting password")
}

func TestUsersClient_DeleteUserCredential(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	userID, _ := createTestUser(t, c, ctx, realmName)

	// Set a password.
	_, err := c.Users.SetUserPassword(ctx, realmName, userID, keycloakapi.CredentialRepresentation{
		Type:      ptr.To("password"),
		Value:     ptr.To("testPassword123!"),
		Temporary: ptr.To(false),
	})
	require.NoError(t, err)

	// List credentials.
	creds, _, err := c.Users.GetUserCredentials(ctx, realmName, userID)
	require.NoError(t, err)
	require.Greater(t, len(creds), 0)

	credID := *creds[0].Id

	// Delete the credential.
	resp, err := c.Users.DeleteUserCredential(ctx, realmName, userID, credID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify deletion.
	creds, _, err = c.Users.GetUserCredentials(ctx, realmName, userID)
	require.NoError(t, err)
	require.Empty(t, creds)
}

func TestUsersClient_ExecuteActionsEmail(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	userID, _ := createTestUser(t, c, ctx, realmName)

	// ExecuteActionsEmail requires SMTP to be configured. Without it Keycloak returns an error.
	// We verify the API call is accepted; a 500 due to missing SMTP config is expected.
	_, err := c.Users.ExecuteActionsEmail(ctx, realmName, userID, []string{"UPDATE_PASSWORD"})
	// Either no error (SMTP configured) or a server error (SMTP not configured) is acceptable.
	if err != nil {
		require.True(t, keycloakapi.IsServerError(err), "expected server error when SMTP is not configured, got: %v", err)
	}
}

func TestUsersClient_SendVerifyEmail(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	userID, _ := createTestUser(t, c, ctx, realmName)

	// SendVerifyEmail requires SMTP. Without it, Keycloak returns an error.
	_, err := c.Users.SendVerifyEmail(ctx, realmName, userID)
	if err != nil {
		require.True(t, keycloakapi.IsServerError(err), "expected server error when SMTP is not configured, got: %v", err)
	}
}

func TestUsersClient_ImpersonateUser(t *testing.T) {
	t.Parallel()

	c, realmName := newUsersTestRealm(t)
	ctx := context.Background()

	userID, _ := createTestUser(t, c, ctx, realmName)

	result, resp, err := c.Users.ImpersonateUser(ctx, realmName, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, result)
}
