package testutils

import (
	"fmt"
	"os"
	"strings"
	"testing"

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

// ServerVersionAtLeast reports whether info.SystemInfo.ServerVersion is at least major.minor.
func ServerVersionAtLeast(info *keycloakapi.ServerInfo, major, minor int) (bool, error) {
	var gotMajor, gotMinor int

	if _, err := fmt.Sscanf(strings.TrimSpace(info.SystemInfo.ServerVersion), "%d.%d", &gotMajor, &gotMinor); err != nil {
		return false, fmt.Errorf("parse server version %q: %w", info.SystemInfo.ServerVersion, err)
	}

	return gotMajor > major || (gotMajor == major && gotMinor >= minor), nil
}
