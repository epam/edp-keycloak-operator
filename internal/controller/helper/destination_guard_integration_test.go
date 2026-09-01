package helper

import (
	"context"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/destination"
)

// Enforcement against a running Keycloak, with only that Keycloak's host listed. httptest binds
// 127.0.0.1, so the unlisted server's host differs from the Keycloak host under test.
func TestDestinationGuard_RealKeycloak(t *testing.T) {
	t.Parallel()

	keycloakURL := os.Getenv("TEST_KEYCLOAK_URL")
	if keycloakURL == "" {
		t.Skip("TEST_KEYCLOAK_URL is not set")
	}

	parsed, err := url.Parse(keycloakURL)
	require.NoError(t, err)

	guard, err := destination.New([]string{parsed.Hostname()}, true)
	require.NoError(t, err)

	t.Run("listed Keycloak host authenticates", func(t *testing.T) {
		t.Parallel()

		kc := keycloakCRWithPasswordGrant(keycloakURL)

		// The fixture Secret holds the Keycloak admin password; a listed host must authenticate.
		h := guardedHelperWithPassword(t, kc, guard, "admin")

		client, err := h.CreateKeycloakClientFromKeycloak(context.Background(), kc)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("unlisted host is denied and receives nothing", func(t *testing.T) {
		t.Parallel()

		var received string

		server := recordingServer(t, &received)
		kc := keycloakCRWithPasswordGrant(server.URL)

		_, err := guardedHelper(t, kc, guard).CreateKeycloakClientFromKeycloak(context.Background(), kc)

		require.ErrorIs(t, err, destination.ErrNotAllowed)
		assert.Empty(t, received, "nothing may reach an unlisted destination")
	})
}
