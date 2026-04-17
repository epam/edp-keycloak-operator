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

func TestSessionsClient_GetRealmSessionStats(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-sessions-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: ptr.To(true),
	})
	require.NoError(t, err)

	stats, resp, err := c.Sessions.GetRealmSessionStats(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	// Stats may be empty for a fresh realm with no active sessions.
	require.NotNil(t, stats)
}

func TestSessionsClient_LogoutAllSessions(t *testing.T) {
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
	realmName := fmt.Sprintf("test-realm-logout-all-%d", time.Now().UnixNano())

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: ptr.To(true),
	})
	require.NoError(t, err)

	resp, err := c.Sessions.LogoutAllSessions(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
}
