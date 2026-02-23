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

func TestRealmClient_CRUD(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-realm-crud-%d", time.Now().UnixNano())
	displayName := "Test Realm for CRUD"
	enabled := true

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// 1. Create test realm
	realm := keycloakv2.RealmRepresentation{
		Realm:       &realmName,
		DisplayName: &displayName,
		Enabled:     &enabled,
	}

	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// 2. Get realm and verify
	retrievedRealm, resp, err := c.Realms.GetRealm(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, retrievedRealm)
	require.Equal(t, realmName, *retrievedRealm.Realm)
	require.Equal(t, displayName, *retrievedRealm.DisplayName)
	require.Equal(t, enabled, *retrievedRealm.Enabled)

	// 3. Update realm settings
	updatedDisplayName := "Updated Test Realm"
	realm = keycloakv2.RealmRepresentation{
		Realm:       &realmName,
		DisplayName: &updatedDisplayName,
		Enabled:     &enabled,
	}

	resp, err = c.Realms.UpdateRealm(ctx, realmName, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// 4. Get realm and verify update
	updatedRealm, resp, err := c.Realms.GetRealm(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, updatedRealm)
	require.Equal(t, updatedDisplayName, *updatedRealm.DisplayName, "DisplayName should be updated")

	// 5. Delete realm
	resp, err = c.Realms.DeleteRealm(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// 6. Verify deletion (Get should fail with 404)
	_, _, err = c.Realms.GetRealm(ctx, realmName)
	require.Error(t, err)

	require.True(t, keycloakv2.IsNotFound(err), "Expected 404 Not Found error")
}

func TestRealmClient_GetRealm(t *testing.T) {
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

	// Test getting the master realm (should always exist)
	realm, resp, err := c.Realms.GetRealm(ctx, keycloakv2.MasterRealm)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, realm)
	require.Equal(t, keycloakv2.MasterRealm, *realm.Realm)
	require.NotNil(t, realm.Enabled)
	require.True(t, *realm.Enabled)
}

func TestRealmClient_GetRealm_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	ctx := context.Background()

	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		ctx,
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	// Test getting a non-existent realm
	realm, resp, err := c.Realms.GetRealm(ctx, "nonexistent-realm-12345")
	require.Error(t, err)

	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent realm")
	require.Nil(t, realm, "Realm should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestRealmClient_DeleteRealm_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	ctx := context.Background()

	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		ctx,
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	// Test deleting a non-existent realm
	resp, err := c.Realms.DeleteRealm(ctx, "nonexistent-realm-12345")
	require.Error(t, err)

	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent realm")
	require.NotNil(t, resp, "Response should be present even for error")
	require.NotNil(t, resp.HTTPResponse, "HTTPResponse should be present even for error")
}

func TestRealmClient_SetRealmBrowserFlow(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-browser-flow-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	resp, err := c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// "browser" is the default flow that always exists in every Keycloak realm
	resp, err = c.Realms.SetRealmBrowserFlow(ctx, realmName, "browser")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// Verify the browser flow was set
	realm, _, err := c.Realms.GetRealm(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, realm)
	require.NotNil(t, realm.BrowserFlow)
	require.Equal(t, "browser", *realm.BrowserFlow)
}

func TestRealmClient_SetRealmBrowserFlow_RealmNotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Realms.SetRealmBrowserFlow(context.Background(), "nonexistent-realm-12345", "browser")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err))
	require.NotNil(t, resp)
}

func TestRealmClient_SetRealmEventConfig(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-event-config-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	resp, err := c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	eventsEnabled := true
	adminEventsEnabled := true
	adminEventsDetailsEnabled := true

	var eventsExpiration int64 = 3600

	eventTypes := []string{"LOGIN", "LOGOUT"}

	cfg := keycloakv2.RealmEventsConfigRepresentation{
		EventsEnabled:             &eventsEnabled,
		AdminEventsEnabled:        &adminEventsEnabled,
		AdminEventsDetailsEnabled: &adminEventsDetailsEnabled,
		EventsExpiration:          &eventsExpiration,
		EnabledEventTypes:         &eventTypes,
	}

	resp, err = c.Realms.SetRealmEventConfig(ctx, realmName, cfg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
}

func TestRealmClient_SetRealmEventConfig_RealmNotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	eventsEnabled := true
	cfg := keycloakv2.RealmEventsConfigRepresentation{
		EventsEnabled: &eventsEnabled,
	}

	resp, err := c.Realms.SetRealmEventConfig(context.Background(), "nonexistent-realm-12345", cfg)
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err))
	require.NotNil(t, resp)
}
