package testutils

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// GetKeycloakURLOrSkip returns the TEST_KEYCLOAK_URL environment variable value.
// If the environment variable is not set, it skips the test.
// This is used for integration tests that require a running Keycloak instance.
func GetKeycloakURLOrSkip(t *testing.T) string {
	t.Helper()

	url := os.Getenv("TEST_KEYCLOAK_URL")
	if url == "" {
		t.Skip("TEST_KEYCLOAK_URL not set, skipping integration test")
	}

	return url
}

// CreateRealmWithRetry creates a fresh enabled realm and registers t.Cleanup
// to delete it. Keycloak transiently returns "unknown_error" under parallel
// test load, so realm creation is retried for up to 10s.
func CreateRealmWithRetry(t *testing.T, c *keycloakapi.KeycloakClient, realmName string) {
	t.Helper()

	enabled := true

	require.Eventually(t, func() bool {
		_, err := c.Realms.CreateRealm(context.Background(), keycloakapi.RealmRepresentation{
			Realm:   &realmName,
			Enabled: &enabled,
		})

		return err == nil
	}, 10*time.Second, time.Second, "creating realm %s", realmName)

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})
}
