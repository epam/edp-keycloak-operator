package keycloakv2_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
)

const testGithubProviderID = "github"

// newIdentityProvidersTestRealm creates a Keycloak client and a fresh realm.
// The realm is automatically deleted in t.Cleanup.
func newIdentityProvidersTestRealm(t *testing.T) (*keycloakv2.KeycloakClient, string) {
	t.Helper()

	keycloakURL := testutils.GetKeycloakURLOrSkip(t)

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	realmName := fmt.Sprintf("test-realm-idp-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(context.Background(), keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	return c, realmName
}

func TestIdentityProvidersClient_CreateIdentityProvider(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)

	ctx := context.Background()

	alias := fmt.Sprintf("test-idp-%d", time.Now().UnixNano())
	displayName := "Test GitHub IdP"
	providerID := testGithubProviderID
	enabled := true

	resp, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakv2.IdentityProviderRepresentation{
		Alias:       &alias,
		DisplayName: &displayName,
		ProviderId:  &providerID,
		Enabled:     &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)
}

func TestIdentityProvidersClient_CreateIdentityProvider_Conflict(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)

	ctx := context.Background()

	alias := fmt.Sprintf("conflict-idp-%d", time.Now().UnixNano())
	displayName := "Conflict IdP"
	providerID := testGithubProviderID
	enabled := true

	idp := keycloakv2.IdentityProviderRepresentation{
		Alias:       &alias,
		DisplayName: &displayName,
		ProviderId:  &providerID,
		Enabled:     &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	}

	// First create should succeed
	_, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, idp)
	require.NoError(t, err)

	// Second create with same alias should conflict
	_, err = c.IdentityProviders.CreateIdentityProvider(ctx, realmName, idp)
	require.Error(t, err)
	require.True(t, keycloakv2.IsConflict(err), "expected 409 conflict for duplicate alias")
}
