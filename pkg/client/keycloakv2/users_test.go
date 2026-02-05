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
	realmName := fmt.Sprintf("test-realm-%d", time.Now().UnixNano())
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
	var originalProfile *keycloakv2.UserProfileConfig

	t.Run("Get", func(t *testing.T) {
		profile, resp, err := c.Users.GetUsersProfile(ctx, realmName)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.HTTPResponse)
		require.NotNil(t, profile)
		require.NotNil(t, profile.Attributes)
		require.Greater(t, len(*profile.Attributes), 0, "Default profile should have attributes")

		// Store for update test
		originalProfile = profile
	})

	// 2. Update user profile with a custom attribute
	t.Run("Update", func(t *testing.T) {
		require.NotNil(t, originalProfile, "Original profile should be fetched first")

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
	})

	// 3. Verify the update by getting the profile again
	t.Run("Verify Update", func(t *testing.T) {
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
	})
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
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent realm")
	require.Nil(t, profile, "Profile should be nil for error response")
	require.Nil(t, resp, "Response should be nil for error response")
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
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent realm")
	require.Nil(t, profile, "Profile should be nil for error response")
	require.Nil(t, resp, "Response should be nil for error response")
}
