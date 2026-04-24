package keycloakapi_test

import (
	"context"
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestKeycloakClient_GetServerInfo(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	err = c.Refresh(context.Background())
	require.NoError(t, err)
	info, err := c.GetServerInfo(context.Background())
	require.NoError(t, err)
	require.NotNil(t, info)
	require.NotEmpty(t, info.SystemInfo.ServerVersion)
	require.NotEmpty(t, info.ComponentTypes)
	require.NotEmpty(t, info.ProviderTypes)
	require.NotEmpty(t, info.Themes)
}

func TestKeycloakClient_FeatureFlagEnabled(t *testing.T) {
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

	// Non-existent feature flag should return false without error
	enabled, err := c.FeatureFlagEnabled(ctx, "definitely-does-not-exist-12345")
	require.NoError(t, err)
	require.False(t, enabled)

	// Known feature flag should return a value without error (exact value depends on Keycloak version/config)
	_, err = c.FeatureFlagEnabled(ctx, "ORGANIZATION")
	require.NoError(t, err)
}
