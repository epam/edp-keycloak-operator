package keycloakapi_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

func newClientPoliciesTestRealm(t *testing.T, suffix string) (*keycloakapi.KeycloakClient, string) {
	t.Helper()

	keycloakURL := testutils.GetKeycloakURLOrSkip(t)

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	realmName := fmt.Sprintf("test-realm-cp-%s-%d", suffix, time.Now().UnixNano())

	testutils.CreateRealmWithRetry(t, c, realmName)

	return c, realmName
}

func TestClientPoliciesClient_GetAndUpdatePolicies(t *testing.T) {
	t.Parallel()

	c, realmName := newClientPoliciesTestRealm(t, "policies")
	ctx := context.Background()

	// Get current policies.
	policies, resp, err := c.ClientPolicies.GetClientPolicies(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, policies)

	// Update policies (round-trip: send back what we got).
	resp, err = c.ClientPolicies.UpdateClientPolicies(ctx, realmName, *policies)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify round-trip.
	updatedPolicies, _, err := c.ClientPolicies.GetClientPolicies(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, updatedPolicies)
}

func TestClientPoliciesClient_GetAndUpdateProfiles(t *testing.T) {
	t.Parallel()

	c, realmName := newClientPoliciesTestRealm(t, "profiles")
	ctx := context.Background()

	// Get current profiles.
	profiles, resp, err := c.ClientPolicies.GetClientProfiles(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, profiles)

	// Update profiles (round-trip).
	resp, err = c.ClientPolicies.UpdateClientProfiles(ctx, realmName, *profiles)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify.
	updatedProfiles, _, err := c.ClientPolicies.GetClientProfiles(ctx, realmName, nil)
	require.NoError(t, err)
	require.NotNil(t, updatedProfiles)
}
