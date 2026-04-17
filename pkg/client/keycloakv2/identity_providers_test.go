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
const testMapperName = "test-mapper"

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

func TestIdentityProvidersClient_GetAndDeleteIdentityProvider(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)

	ctx := context.Background()

	alias := fmt.Sprintf("test-idp-get-del-%d", time.Now().UnixNano())
	providerID := testGithubProviderID
	enabled := true

	_, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakv2.IdentityProviderRepresentation{
		Alias:      &alias,
		ProviderId: &providerID,
		Enabled:    &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	})
	require.NoError(t, err)

	// Get the identity provider
	idp, resp, err := c.IdentityProviders.GetIdentityProvider(ctx, realmName, alias)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, idp)
	require.NotNil(t, idp.Alias)
	require.Equal(t, alias, *idp.Alias)

	// Delete the identity provider
	delResp, err := c.IdentityProviders.DeleteIdentityProvider(ctx, realmName, alias)
	require.NoError(t, err)
	require.NotNil(t, delResp)

	// Verify deletion
	_, _, err = c.IdentityProviders.GetIdentityProvider(ctx, realmName, alias)
	require.True(t, keycloakv2.IsNotFound(err), "identity provider should be gone after deletion")
}

func TestIdentityProvidersClient_GetIdentityProvider_NotFound(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)

	_, resp, err := c.IdentityProviders.GetIdentityProvider(context.Background(), realmName, "nonexistent-alias-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 for non-existent identity provider")
	require.NotNil(t, resp)
}

func TestIdentityProvidersClient_UpdateIdentityProvider(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)

	ctx := context.Background()

	alias := fmt.Sprintf("test-idp-update-%d", time.Now().UnixNano())
	providerID := testGithubProviderID
	enabled := true

	_, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakv2.IdentityProviderRepresentation{
		Alias:      &alias,
		ProviderId: &providerID,
		Enabled:    &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	})
	require.NoError(t, err)

	displayName := "Updated IDP"
	_, err = c.IdentityProviders.UpdateIdentityProvider(ctx, realmName, alias, keycloakv2.IdentityProviderRepresentation{
		Alias:       &alias,
		ProviderId:  &providerID,
		Enabled:     &enabled,
		DisplayName: &displayName,
		Config: &map[string]string{
			"clientId":     "updated-client-id",
			"clientSecret": "updated-client-secret",
		},
	})
	require.NoError(t, err)

	idp, _, err := c.IdentityProviders.GetIdentityProvider(ctx, realmName, alias)
	require.NoError(t, err)
	require.NotNil(t, idp.DisplayName)
	require.Equal(t, "Updated IDP", *idp.DisplayName)
}

func TestIdentityProvidersClient_Mappers(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)

	ctx := context.Background()

	alias := fmt.Sprintf("test-idp-mappers-%d", time.Now().UnixNano())
	providerID := testGithubProviderID
	enabled := true

	_, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakv2.IdentityProviderRepresentation{
		Alias:      &alias,
		ProviderId: &providerID,
		Enabled:    &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	})
	require.NoError(t, err)

	// GetIDPMappers — empty initially
	mappers, _, err := c.IdentityProviders.GetIDPMappers(ctx, realmName, alias)
	require.NoError(t, err)
	require.Empty(t, mappers)

	// CreateIDPMapper
	mapperName := "test-mapper"
	mapperType := "hardcoded-attribute-idp-mapper"
	_, err = c.IdentityProviders.CreateIDPMapper(ctx, realmName, alias, keycloakv2.IdentityProviderMapperRepresentation{
		Name:                   &mapperName,
		IdentityProviderAlias:  &alias,
		IdentityProviderMapper: &mapperType,
		Config: &map[string]string{
			"attribute":       "test-attr",
			"attribute.value": "test-value",
		},
	})
	require.NoError(t, err)

	// GetIDPMappers — should have one
	mappers, _, err = c.IdentityProviders.GetIDPMappers(ctx, realmName, alias)
	require.NoError(t, err)
	require.Len(t, mappers, 1)
	require.NotNil(t, mappers[0].Name)
	require.Equal(t, testMapperName, *mappers[0].Name)
	require.NotNil(t, mappers[0].Id)

	// DeleteIDPMapper
	_, err = c.IdentityProviders.DeleteIDPMapper(ctx, realmName, alias, *mappers[0].Id)
	require.NoError(t, err)

	// Verify deletion
	mappers, _, err = c.IdentityProviders.GetIDPMappers(ctx, realmName, alias)
	require.NoError(t, err)
	require.Empty(t, mappers)
}

func TestIdentityProvidersClient_ManagementPermissions(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)

	ctx := context.Background()

	alias := fmt.Sprintf("test-idp-perms-%d", time.Now().UnixNano())
	providerID := testGithubProviderID
	enabled := true

	_, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakv2.IdentityProviderRepresentation{
		Alias:      &alias,
		ProviderId: &providerID,
		Enabled:    &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	})
	require.NoError(t, err)

	// GetIDPManagementPermissions — disabled by default
	perms, _, err := c.IdentityProviders.GetIDPManagementPermissions(ctx, realmName, alias)
	require.NoError(t, err)
	require.NotNil(t, perms)
	require.NotNil(t, perms.Enabled)
	require.False(t, *perms.Enabled)

	// UpdateIDPManagementPermissions — enable
	enabledTrue := true
	updated, _, err := c.IdentityProviders.UpdateIDPManagementPermissions(ctx, realmName, alias,
		keycloakv2.ManagementPermissionReference{
			Enabled: &enabledTrue,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.NotNil(t, updated.Enabled)
	require.True(t, *updated.Enabled)

	// Verify
	perms, _, err = c.IdentityProviders.GetIDPManagementPermissions(ctx, realmName, alias)
	require.NoError(t, err)
	require.NotNil(t, perms)
	require.True(t, *perms.Enabled)
}

func TestIdentityProvidersClient_GetIdentityProviders(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)
	ctx := context.Background()

	// Create an IDP.
	alias := fmt.Sprintf("list-idp-%d", time.Now().UnixNano())
	providerID := testGithubProviderID
	enabled := true

	_, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakv2.IdentityProviderRepresentation{
		Alias:      &alias,
		ProviderId: &providerID,
		Enabled:    &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	})
	require.NoError(t, err)

	// List all IDPs.
	idps, resp, err := c.IdentityProviders.GetIdentityProviders(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Greater(t, len(idps), 0)

	found := false

	for _, idp := range idps {
		if idp.Alias != nil && *idp.Alias == alias {
			found = true

			break
		}
	}

	require.True(t, found, "created IDP should be in the list")
}

func TestIdentityProvidersClient_UpdateIDPMapper(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)
	ctx := context.Background()

	// Create an IDP.
	alias := fmt.Sprintf("mapper-idp-%d", time.Now().UnixNano())
	providerID := testGithubProviderID
	enabled := true

	_, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakv2.IdentityProviderRepresentation{
		Alias:      &alias,
		ProviderId: &providerID,
		Enabled:    &enabled,
		Config: &map[string]string{
			"clientId":     "test-client-id",
			"clientSecret": "test-client-secret",
		},
	})
	require.NoError(t, err)

	// Create an IDP mapper.
	mapperName := testMapperName
	mapperType := "hardcoded-attribute-idp-mapper"
	mapperConfig := map[string]string{
		"syncMode":        "INHERIT",
		"attribute":       "test-attr",
		"attribute.value": "original-value",
	}

	mapperResp, err := c.IdentityProviders.CreateIDPMapper(ctx, realmName, alias,
		keycloakv2.IdentityProviderMapperRepresentation{
			Name:                   &mapperName,
			IdentityProviderAlias:  &alias,
			IdentityProviderMapper: &mapperType,
			Config:                 &mapperConfig,
		})
	require.NoError(t, err)

	mapperID := keycloakv2.GetResourceIDFromResponse(mapperResp)
	require.NotEmpty(t, mapperID)

	// Update the mapper.
	updatedConfig := map[string]string{
		"syncMode":        "INHERIT",
		"attribute":       "test-attr",
		"attribute.value": "updated-value",
	}

	resp, err := c.IdentityProviders.UpdateIDPMapper(ctx, realmName, alias, mapperID,
		keycloakv2.IdentityProviderMapperRepresentation{
			Id:                     &mapperID,
			Name:                   &mapperName,
			IdentityProviderAlias:  &alias,
			IdentityProviderMapper: &mapperType,
			Config:                 &updatedConfig,
		})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify by listing mappers.
	mappers, _, err := c.IdentityProviders.GetIDPMappers(ctx, realmName, alias)
	require.NoError(t, err)
	require.Len(t, mappers, 1)
	require.Equal(t, "updated-value", (*mappers[0].Config)["attribute.value"])
}

func TestIdentityProvidersClient_ExportBrokerConfig(t *testing.T) {
	t.Parallel()

	c, realmName := newIdentityProvidersTestRealm(t)
	ctx := context.Background()

	// Create a SAML IDP for export testing.
	alias := fmt.Sprintf("export-idp-%d", time.Now().UnixNano())
	providerID := "saml"
	enabled := true

	_, err := c.IdentityProviders.CreateIdentityProvider(ctx, realmName, keycloakv2.IdentityProviderRepresentation{
		Alias:      &alias,
		ProviderId: &providerID,
		Enabled:    &enabled,
		Config: &map[string]string{
			"singleSignOnServiceUrl": "https://idp.example.com/sso",
		},
	})
	require.NoError(t, err)

	data, resp, err := c.IdentityProviders.ExportBrokerConfig(ctx, realmName, alias, "saml-sp-descriptor")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, data, "exported SAML SP descriptor should not be empty")
}
