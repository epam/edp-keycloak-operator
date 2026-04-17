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

func TestClientScopesClient_CRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewAPIClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-cs-crud-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	scopeName := "test-scope"
	protocol := protocolOpenIDConnect
	description := "Test scope"

	// 1. Create client scope
	resp, err := c.ClientScopes.CreateClientScope(ctx, realmName, keycloakapi.ClientScopeRepresentation{
		Name:        &scopeName,
		Protocol:    &protocol,
		Description: &description,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	scopeID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, scopeID, "scope ID should be extracted from Location header")

	// 2. Get client scope by ID
	scope, _, err := c.ClientScopes.GetClientScope(ctx, realmName, scopeID)
	require.NoError(t, err)
	require.NotNil(t, scope)
	require.Equal(t, scopeName, *scope.Name)
	require.Equal(t, description, *scope.Description)

	// 3. List all client scopes and find ours
	scopes, _, err := c.ClientScopes.GetClientScopes(ctx, realmName)
	require.NoError(t, err)

	found := false

	for _, s := range scopes {
		if s.Name != nil && *s.Name == scopeName {
			found = true
			break
		}
	}

	require.True(t, found, "Created scope should be in the scopes list")

	// 4. Update client scope
	newDesc := "Updated description"
	_, err = c.ClientScopes.UpdateClientScope(ctx, realmName, scopeID, keycloakapi.ClientScopeRepresentation{
		Name:        &scopeName,
		Protocol:    &protocol,
		Description: &newDesc,
	})
	require.NoError(t, err)

	scope, _, err = c.ClientScopes.GetClientScope(ctx, realmName, scopeID)
	require.NoError(t, err)
	require.Equal(t, newDesc, *scope.Description)

	// 5. Delete client scope
	_, err = c.ClientScopes.DeleteClientScope(ctx, realmName, scopeID)
	require.NoError(t, err)

	// 6. Verify deletion
	_, _, err = c.ClientScopes.GetClientScope(ctx, realmName, scopeID)
	require.Error(t, err)
	require.True(t, keycloakapi.IsNotFound(err))
}

func TestClientScopesClient_RealmScopeType(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewAPIClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-cs-type-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	scopeName := "test-type-scope"
	protocol := protocolOpenIDConnect

	resp, err := c.ClientScopes.CreateClientScope(ctx, realmName, keycloakapi.ClientScopeRepresentation{
		Name:     &scopeName,
		Protocol: &protocol,
	})
	require.NoError(t, err)

	scopeID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, scopeID)

	// Add to default scopes
	_, err = c.ClientScopes.AddRealmDefaultClientScope(ctx, realmName, scopeID)
	require.NoError(t, err)

	// Verify in default list
	defaultScopes, _, err := c.ClientScopes.GetRealmDefaultClientScopes(ctx, realmName)
	require.NoError(t, err)
	require.True(t, containsScope(defaultScopes, scopeName))

	// Remove from default, add to optional
	_, err = c.ClientScopes.RemoveRealmDefaultClientScope(ctx, realmName, scopeID)
	require.NoError(t, err)

	_, err = c.ClientScopes.AddRealmOptionalClientScope(ctx, realmName, scopeID)
	require.NoError(t, err)

	// Verify in optional list and not in default
	optionalScopes, _, err := c.ClientScopes.GetRealmOptionalClientScopes(ctx, realmName)
	require.NoError(t, err)
	require.True(t, containsScope(optionalScopes, scopeName))

	defaultScopes, _, err = c.ClientScopes.GetRealmDefaultClientScopes(ctx, realmName)
	require.NoError(t, err)
	require.False(t, containsScope(defaultScopes, scopeName))

	// Remove from optional
	_, err = c.ClientScopes.RemoveRealmOptionalClientScope(ctx, realmName, scopeID)
	require.NoError(t, err)

	optionalScopes, _, err = c.ClientScopes.GetRealmOptionalClientScopes(ctx, realmName)
	require.NoError(t, err)
	require.False(t, containsScope(optionalScopes, scopeName))
}

func TestClientScopesClient_ProtocolMappers(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewAPIClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-cs-mappers-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	scopeName := "test-mapper-scope"
	protocol := protocolOpenIDConnect

	resp, err := c.ClientScopes.CreateClientScope(ctx, realmName, keycloakapi.ClientScopeRepresentation{
		Name:     &scopeName,
		Protocol: &protocol,
	})
	require.NoError(t, err)

	scopeID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, scopeID)

	// Create a protocol mapper
	mapperName := "groups-mapper"
	mapperProtocol := protocolOpenIDConnect
	mapperType := "oidc-group-membership-mapper"
	config := map[string]string{
		"access.token.claim":   "true",
		"claim.name":           "groups",
		"full.path":            "false",
		"id.token.claim":       "true",
		"userinfo.token.claim": "true",
	}

	_, err = c.ClientScopes.CreateClientScopeProtocolMapper(
		ctx, realmName, scopeID, keycloakapi.ProtocolMapperRepresentation{
			Name:           &mapperName,
			Protocol:       &mapperProtocol,
			ProtocolMapper: &mapperType,
			Config:         &config,
		})
	require.NoError(t, err)

	// Get protocol mappers
	mappers, _, err := c.ClientScopes.GetClientScopeProtocolMappers(ctx, realmName, scopeID)
	require.NoError(t, err)
	require.Len(t, mappers, 1)
	require.Equal(t, mapperName, *mappers[0].Name)

	// Delete protocol mapper
	_, err = c.ClientScopes.DeleteClientScopeProtocolMapper(ctx, realmName, scopeID, *mappers[0].Id)
	require.NoError(t, err)

	// Verify deletion
	mappers, _, err = c.ClientScopes.GetClientScopeProtocolMappers(ctx, realmName, scopeID)
	require.NoError(t, err)
	require.Empty(t, mappers)
}

func containsScope(scopes []keycloakapi.ClientScopeRepresentation, name string) bool {
	for _, s := range scopes {
		if s.Name != nil && *s.Name == name {
			return true
		}
	}

	return false
}

func TestClientScopesClient_UpdateClientScopeProtocolMapper(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakapi.NewAPIClient(
		context.Background(),
		keycloakURL,
		keycloakapi.DefaultAdminClientID,
		keycloakapi.WithPasswordGrant(keycloakapi.DefaultAdminUsername, keycloakapi.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-cs-updmapper-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakapi.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	// Create a client scope.
	scopeName := fmt.Sprintf("test-scope-updmapper-%d", time.Now().UnixNano())
	protocol := "openid-connect"

	resp, err := c.ClientScopes.CreateClientScope(ctx, realmName, keycloakapi.ClientScopeRepresentation{
		Name:     &scopeName,
		Protocol: &protocol,
	})
	require.NoError(t, err)

	scopeID := keycloakapi.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, scopeID)

	// Create a protocol mapper.
	mapperName := "test-mapper"
	mapperProtocol := "openid-connect"
	mapperType := "oidc-hardcoded-claim-mapper"
	config := map[string]string{
		"claim.name":         "test-claim",
		"claim.value":        "original-value",
		"jsonType.label":     "String",
		"id.token.claim":     "true",
		"access.token.claim": "true",
	}

	_, err = c.ClientScopes.CreateClientScopeProtocolMapper(ctx, realmName, scopeID,
		keycloakapi.ProtocolMapperRepresentation{
			Name:           &mapperName,
			Protocol:       &mapperProtocol,
			ProtocolMapper: &mapperType,
			Config:         &config,
		})
	require.NoError(t, err)

	// Get the mapper to obtain its ID.
	mappers, _, err := c.ClientScopes.GetClientScopeProtocolMappers(ctx, realmName, scopeID)
	require.NoError(t, err)
	require.Len(t, mappers, 1)

	mapperID := *mappers[0].Id

	// Update the mapper.
	updatedConfig := map[string]string{
		"claim.name":         "test-claim",
		"claim.value":        "updated-value",
		"jsonType.label":     "String",
		"id.token.claim":     "true",
		"access.token.claim": "true",
	}

	resp, err = c.ClientScopes.UpdateClientScopeProtocolMapper(ctx, realmName, scopeID, mapperID,
		keycloakapi.ProtocolMapperRepresentation{
			Id:             &mapperID,
			Name:           &mapperName,
			Protocol:       &mapperProtocol,
			ProtocolMapper: &mapperType,
			Config:         &updatedConfig,
		})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify the update.
	mappers, _, err = c.ClientScopes.GetClientScopeProtocolMappers(ctx, realmName, scopeID)
	require.NoError(t, err)
	require.Len(t, mappers, 1)
	require.Equal(t, "updated-value", (*mappers[0].Config)["claim.value"])
}
