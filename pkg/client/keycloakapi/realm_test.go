package keycloakapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

func TestRealmClient_CRUD(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-realm-crud-%d", time.Now().UnixNano())
	displayName := "Test Realm for CRUD"
	enabled := true

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// 1. Create test realm
	realm := keycloakapi.RealmRepresentation{
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
	realm = keycloakapi.RealmRepresentation{
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

	require.True(t, keycloakapi.IsNotFound(err), "Expected 404 Not Found error")
}

func TestRealmClient_GetRealm(t *testing.T) {
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

	// Test getting the master realm (should always exist)
	realm, resp, err := c.Realms.GetRealm(ctx, keycloakapi.MasterRealm)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
	require.NotNil(t, realm)
	require.Equal(t, keycloakapi.MasterRealm, *realm.Realm)
	require.NotNil(t, realm.Enabled)
	require.True(t, *realm.Enabled)
}

func TestRealmClient_GetRealm_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	ctx := context.Background()

	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		ctx,
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	// Test getting a non-existent realm
	realm, resp, err := c.Realms.GetRealm(ctx, "nonexistent-realm-12345")
	require.Error(t, err)

	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 error for non-existent realm")
	require.Nil(t, realm, "Realm should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestRealmClient_DeleteRealm_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	ctx := context.Background()

	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		ctx,
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	// Test deleting a non-existent realm
	resp, err := c.Realms.DeleteRealm(ctx, "nonexistent-realm-12345")
	require.Error(t, err)

	require.True(t, keycloakapi.IsNotFound(err), "Should return 404 error for non-existent realm")
	require.NotNil(t, resp, "Response should be present even for error")
	require.NotNil(t, resp.HTTPResponse, "HTTPResponse should be present even for error")
}

func TestRealmClient_SetRealmBrowserFlow(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-browser-flow-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	// "browser" is the default flow that always exists in every Keycloak realm
	resp, err := c.Realms.SetRealmBrowserFlow(ctx, realmName, "browser")
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

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	resp, err := c.Realms.SetRealmBrowserFlow(context.Background(), "nonexistent-realm-12345", "browser")
	require.Error(t, err)
	require.True(t, keycloakapi.IsNotFound(err))
	require.NotNil(t, resp)
}

func TestRealmClient_SetRealmEventConfig(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-event-config-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	eventsEnabled := true
	adminEventsEnabled := true
	adminEventsDetailsEnabled := true

	var eventsExpiration int64 = 3600

	eventTypes := []string{"LOGIN", "LOGOUT"}

	cfg := keycloakapi.RealmEventsConfigRepresentation{
		EventsEnabled:             &eventsEnabled,
		AdminEventsEnabled:        &adminEventsEnabled,
		AdminEventsDetailsEnabled: &adminEventsDetailsEnabled,
		EventsExpiration:          &eventsExpiration,
		EnabledEventTypes:         &eventTypes,
	}

	resp, err := c.Realms.SetRealmEventConfig(ctx, realmName, cfg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
}

func TestRealmClient_GetAuthenticationFlows(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-auth-flows-%d", time.Now().UnixNano())
	testutils.CreateRealmWithRetry(t, c, realmName)

	flows, resp, err := c.Realms.GetAuthenticationFlows(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Greater(t, len(flows), 0, "realm should have built-in authentication flows")

	// "browser" flow is always present in every Keycloak realm
	found := false

	for _, f := range flows {
		if f.Alias != nil && *f.Alias == "browser" {
			found = true
			break
		}
	}

	require.True(t, found, "browser flow should exist in every realm")
}

func TestRealmClient_GetAuthenticationFlows_RealmNotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	_, resp, err := c.Realms.GetAuthenticationFlows(context.Background(), "nonexistent-realm-12345")
	require.Error(t, err)
	require.True(t, keycloakapi.IsNotFound(err))
	require.NotNil(t, resp)
}

func TestRealmClient_SetRealmEventConfig_RealmNotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	eventsEnabled := true
	cfg := keycloakapi.RealmEventsConfigRepresentation{
		EventsEnabled: &eventsEnabled,
	}

	resp, err := c.Realms.SetRealmEventConfig(context.Background(), "nonexistent-realm-12345", cfg)
	require.Error(t, err)
	require.True(t, keycloakapi.IsNotFound(err))
	require.NotNil(t, resp)
}

func TestRealmClient_GetRealms(t *testing.T) {
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

	realms, resp, err := c.Realms.GetRealms(ctx)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Greater(t, len(realms), 0, "at least the master realm should exist")

	found := false

	for _, r := range realms {
		if r.Realm != nil && *r.Realm == keycloakapi.MasterRealm {
			found = true

			break
		}
	}

	require.True(t, found, "master realm should be in the list")
}

func TestRealmClient_GetRealmKeys(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-keys-%d", time.Now().UnixNano())

	testutils.CreateRealmWithRetry(t, c, realmName)

	keys, resp, err := c.Realms.GetRealmKeys(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, keys)
	require.NotNil(t, keys.Keys)
	require.Greater(t, len(*keys.Keys), 0, "realm should have at least one key")
}

func TestRealmClient_GetRealmLocalization(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-localization-%d", time.Now().UnixNano())

	testutils.CreateRealmWithRetry(t, c, realmName)

	// A fresh realm may have no localization entries for "en" — that is valid.
	localization, resp, err := c.Realms.GetRealmLocalization(ctx, realmName, "en")
	require.NoError(t, err)
	require.NotNil(t, resp)
	// localization can be nil when no entries exist for the locale.
	_ = localization
}

func TestRealmClient_PostRealmLocalization(t *testing.T) {
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

	realmName := fmt.Sprintf("test-realm-localization-post-%d", time.Now().UnixNano())

	testutils.CreateRealmWithRetry(t, c, realmName)

	_, err = c.Realms.PostRealmLocalization(ctx, realmName, "en", map[string]string{
		"customTestKey": "customTestValue",
	})
	require.NoError(t, err)

	got, resp, err := c.Realms.GetRealmLocalization(ctx, realmName, "en")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, got)
	assert.Equal(t, "customTestValue", got["customTestKey"])
}
