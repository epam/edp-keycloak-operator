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

func TestServerVersionAtLeast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		info    *keycloakapi.ServerInfo
		want    bool
		wantErr bool
	}{
		{name: "26.8.0", info: serverInfoWithVersion("26.8.0"), want: true},
		{name: "26.8.1", info: serverInfoWithVersion("26.8.1"), want: true},
		{name: "26.10.0 compares as integer", info: serverInfoWithVersion("26.10.0"), want: true},
		{name: "27.0.0", info: serverInfoWithVersion("27.0.0"), want: true},
		{name: "snapshot suffix", info: serverInfoWithVersion("999.0.0-SNAPSHOT"), want: true},
		{name: "26.7.5", info: serverInfoWithVersion("26.7.5"), want: false},
		{name: "26.4.0", info: serverInfoWithVersion("26.4.0"), want: false},
		{name: "25.0.6", info: serverInfoWithVersion("25.0.6"), want: false},
		{name: "downstream suffix", info: serverInfoWithVersion("26.4.10.redhat-00001"), want: false},
		{name: "surrounding whitespace", info: serverInfoWithVersion(" 26.8.0 "), want: true},
		{name: "empty", info: serverInfoWithVersion(""), wantErr: true},
		{name: "UNKNOWN", info: serverInfoWithVersion("UNKNOWN"), wantErr: true},
		{name: "major only", info: serverInfoWithVersion("26"), wantErr: true},
		{name: "nil info", info: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := keycloakapi.ServerVersionAtLeast(tt.info, 26, 8)
			if tt.wantErr {
				require.Error(t, err)
				require.False(t, got)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func serverInfoWithVersion(version string) *keycloakapi.ServerInfo {
	return &keycloakapi.ServerInfo{SystemInfo: keycloakapi.SystemInfo{ServerVersion: version}}
}
