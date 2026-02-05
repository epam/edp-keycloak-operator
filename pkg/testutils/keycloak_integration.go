package testutils

import (
	"os"
	"testing"
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
