package keycloakv2_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

// newEventsTestRealm creates a Keycloak client and a fresh realm with events enabled.
// The realm is automatically deleted in t.Cleanup.
func newEventsTestRealm(t *testing.T) (*keycloakv2.KeycloakClient, string) {
	t.Helper()

	keycloakURL := testutils.GetKeycloakURLOrSkip(t)

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	realmName := fmt.Sprintf("test-realm-events-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(context.Background(), keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: ptr.To(true),
	})
	require.NoError(t, err)

	// Enable events so that queries work.
	_, err = c.Realms.SetRealmEventConfig(context.Background(), realmName, keycloakv2.RealmEventsConfigRepresentation{
		EventsEnabled:      ptr.To(true),
		AdminEventsEnabled: ptr.To(true),
	})
	require.NoError(t, err)

	return c, realmName
}

func TestEventsClient_GetEvents(t *testing.T) {
	t.Parallel()

	c, realmName := newEventsTestRealm(t)
	ctx := context.Background()

	events, resp, err := c.Events.GetEvents(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	// A freshly-created realm may have zero events — that is valid.
	require.NotNil(t, events)
}

func TestEventsClient_GetAdminEvents(t *testing.T) {
	t.Parallel()

	c, realmName := newEventsTestRealm(t)
	ctx := context.Background()

	events, resp, err := c.Events.GetAdminEvents(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, events)
}

func TestEventsClient_DeleteEvents(t *testing.T) {
	t.Parallel()

	c, realmName := newEventsTestRealm(t)
	ctx := context.Background()

	resp, err := c.Events.DeleteEvents(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// After deletion, list should be empty.
	events, _, err := c.Events.GetEvents(ctx, realmName, nil)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestEventsClient_DeleteAdminEvents(t *testing.T) {
	t.Parallel()

	c, realmName := newEventsTestRealm(t)
	ctx := context.Background()

	resp, err := c.Events.DeleteAdminEvents(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)

	events, _, err := c.Events.GetAdminEvents(ctx, realmName, nil)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestEventsClient_GetEventsConfig(t *testing.T) {
	t.Parallel()

	c, realmName := newEventsTestRealm(t)
	ctx := context.Background()

	cfg, resp, err := c.Events.GetEventsConfig(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, cfg)
	// We enabled events in the helper, so they should be enabled.
	require.NotNil(t, cfg.EventsEnabled)
	require.True(t, *cfg.EventsEnabled)
}

func TestEventsClient_BruteForce(t *testing.T) {
	t.Parallel()

	c, realmName := newEventsTestRealm(t)
	ctx := context.Background()

	// Create a user for brute-force testing.
	username := fmt.Sprintf("bf-user-%d", time.Now().UnixNano())

	userResp, err := c.Users.CreateUser(ctx, realmName, keycloakv2.UserRepresentation{
		Username: &username,
		Enabled:  ptr.To(true),
	})
	require.NoError(t, err)

	userID := keycloakv2.GetResourceIDFromResponse(userResp)
	require.NotEmpty(t, userID)

	// GetBruteForceStatus — should return status map (no lockout on a fresh user).
	status, resp, err := c.Events.GetBruteForceStatus(ctx, realmName, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, status)

	// ClearBruteForceForUser — should succeed (no-op when no lockout).
	resp, err = c.Events.ClearBruteForceForUser(ctx, realmName, userID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// ClearAllBruteForce — should succeed.
	resp, err = c.Events.ClearAllBruteForce(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
}
