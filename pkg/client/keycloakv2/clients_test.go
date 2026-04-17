package keycloakv2_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
	"github.com/stretchr/testify/require"
)

const (
	protocolOpenIDConnect = "openid-connect"
	testClientRoleName    = "test-client-role"
)

func TestClientsClient_ClientCRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name to avoid conflicts
	realmName := fmt.Sprintf("test-realm-cli-crud-%d", time.Now().UnixNano())
	enabled := true

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm first
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	clientId := fmt.Sprintf("test-client-%d", time.Now().UnixNano())
	clientName := "Test Client for CRUD"
	protocol := protocolOpenIDConnect
	publicClient := true

	var clientUUID string

	// 1. Create a client
	t.Run("Create", func(t *testing.T) {
		client := keycloakv2.ClientRepresentation{
			ClientId:     &clientId,
			Name:         &clientName,
			Protocol:     &protocol,
			PublicClient: &publicClient,
		}

		resp, err := c.Clients.CreateClient(ctx, realmName, client)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.HTTPResponse)

		// Extract client UUID from Location header
		clientUUID = keycloakv2.GetResourceIDFromResponse(resp)
		require.NotEmpty(t, clientUUID)
	})

	// 2. Get the created client by UUID
	t.Run("GetByUUID", func(t *testing.T) {
		client, resp, err := c.Clients.GetClient(ctx, realmName, clientUUID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.HTTPResponse)
		require.NotNil(t, client)
		require.NotNil(t, client.ClientId)
		require.Equal(t, clientId, *client.ClientId)
		require.NotNil(t, client.Name)
		require.Equal(t, clientName, *client.Name)
		require.NotNil(t, client.Protocol)
		require.Equal(t, protocol, *client.Protocol)
		require.NotNil(t, client.PublicClient)
		require.Equal(t, publicClient, *client.PublicClient)
	})

	// 3. List all clients and verify our client is present
	t.Run("GetAll", func(t *testing.T) {
		clients, resp, err := c.Clients.GetClients(ctx, realmName, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.HTTPResponse)
		require.NotNil(t, clients)
		require.Greater(t, len(clients), 0, "Should have at least one client")

		// Verify our client is in the list
		found := false

		for _, cl := range clients {
			if cl.ClientId != nil && *cl.ClientId == clientId {
				found = true

				require.NotNil(t, cl.Name)
				require.Equal(t, clientName, *cl.Name)

				break
			}
		}

		require.True(t, found, "Created client should be in the clients list")
	})

	// 4. List clients with search parameter
	t.Run("GetAll with search", func(t *testing.T) {
		params := &keycloakv2.GetClientsParams{
			ClientId: &clientId,
		}

		clients, resp, err := c.Clients.GetClients(ctx, realmName, params)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, clients)
		require.Greater(t, len(clients), 0, "Should find clients matching clientId")

		found := false

		for _, cl := range clients {
			if cl.ClientId != nil && *cl.ClientId == clientId {
				found = true
				break
			}
		}

		require.True(t, found, "Should find test client in search results")
	})

	// 5. Delete the client
	t.Run("Delete", func(t *testing.T) {
		resp, err := c.Clients.DeleteClient(ctx, realmName, clientUUID)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.HTTPResponse)
	})

	// 6. Verify deletion
	t.Run("Verify Delete", func(t *testing.T) {
		_, _, err := c.Clients.GetClient(ctx, realmName, clientUUID)
		require.Error(t, err)
		require.True(t, keycloakv2.IsNotFound(err), "Expected 404 Not Found error after deletion")
	})
}

func TestClientsClient_GetClient_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name
	realmName := fmt.Sprintf("test-realm-cli-get-nf-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Test getting a non-existent client
	client, resp, err := c.Clients.GetClient(ctx, realmName, "nonexistent-client-uuid-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent client")
	require.Nil(t, client, "Client should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestClientsClient_DeleteClient_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name
	realmName := fmt.Sprintf("test-realm-cli-del-nf-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Test deleting a non-existent client
	resp, err = c.Clients.DeleteClient(ctx, realmName, "nonexistent-client-uuid-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent client")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestClientsClient_CreateClient_Conflict(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name to avoid conflicts
	realmName := fmt.Sprintf("test-realm-cli-conflict-%d", time.Now().UnixNano())
	enabled := true

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	clientId := "duplicate-client"
	protocol := protocolOpenIDConnect
	client := keycloakv2.ClientRepresentation{
		ClientId: &clientId,
		Protocol: &protocol,
	}

	// Create the client
	resp, err = c.Clients.CreateClient(ctx, realmName, client)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Create the same client again - should conflict
	resp, err = c.Clients.CreateClient(ctx, realmName, client)
	require.Error(t, err)
	require.True(t, keycloakv2.IsConflict(err), "Should return 409 Conflict error for duplicate client")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestClientsClient_ClientRoleCRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name to avoid conflicts
	realmName := fmt.Sprintf("test-realm-cli-role-crud-%d", time.Now().UnixNano())
	enabled := true

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm first
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// Create test client
	clientId := fmt.Sprintf("test-client-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	client := keycloakv2.ClientRepresentation{
		ClientId: &clientId,
		Protocol: &protocol,
	}

	resp, err = c.Clients.CreateClient(ctx, realmName, client)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.HTTPResponse)

	// Extract client UUID from Location header
	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	roleName := testClientRoleName
	roleDescription := "Test client role for CRUD operations"

	// 1. Create a client role
	t.Run("Create", func(t *testing.T) {
		role := keycloakv2.RoleRepresentation{
			Name:        &roleName,
			Description: &roleDescription,
		}

		resp, err := c.Clients.CreateClientRole(ctx, realmName, clientUUID, role)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.HTTPResponse)
	})

	// 2. Get the created role by name
	t.Run("GetByName", func(t *testing.T) {
		role, resp, err := c.Clients.GetClientRole(ctx, realmName, clientUUID, roleName)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.HTTPResponse)
		require.NotNil(t, role)
		require.NotNil(t, role.Name)
		require.Equal(t, roleName, *role.Name)
		require.NotNil(t, role.Description)
		require.Equal(t, roleDescription, *role.Description)
	})

	// 3. List all client roles and verify our role is present
	t.Run("GetAll", func(t *testing.T) {
		roles, resp, err := c.Clients.GetClientRoles(ctx, realmName, clientUUID, nil)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.HTTPResponse)
		require.NotNil(t, roles)
		require.Greater(t, len(roles), 0, "Should have at least one role")

		// Verify our role is in the list
		found := false

		for _, r := range roles {
			if r.Name != nil && *r.Name == roleName {
				found = true

				require.NotNil(t, r.Description)
				require.Equal(t, roleDescription, *r.Description)

				break
			}
		}

		require.True(t, found, "Created role should be in the roles list")
	})

	// 4. List roles with search parameter
	t.Run("GetAll with search", func(t *testing.T) {
		search := testClientRoleName
		params := &keycloakv2.GetClientRolesParams{
			Search: &search,
		}

		roles, resp, err := c.Clients.GetClientRoles(ctx, realmName, clientUUID, params)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, roles)
		require.Greater(t, len(roles), 0, "Should find roles matching 'test-client-role'")

		found := false

		for _, r := range roles {
			if r.Name != nil && *r.Name == roleName {
				found = true
				break
			}
		}

		require.True(t, found, "Should find test-client-role in search results")
	})

	// 5. Delete the role
	t.Run("Delete", func(t *testing.T) {
		resp, err := c.Clients.DeleteClientRole(ctx, realmName, clientUUID, roleName)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.HTTPResponse)
	})

	// 6. Verify deletion
	t.Run("Verify Delete", func(t *testing.T) {
		_, _, err := c.Clients.GetClientRole(ctx, realmName, clientUUID, roleName)
		require.Error(t, err)
		require.True(t, keycloakv2.IsNotFound(err), "Expected 404 Not Found error after deletion")
	})
}

func TestClientsClient_GetClientRole_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name
	realmName := fmt.Sprintf("test-realm-cli-role-get-nf-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Create test client
	clientId := fmt.Sprintf("test-client-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	client := keycloakv2.ClientRepresentation{
		ClientId: &clientId,
		Protocol: &protocol,
	}

	resp, err = c.Clients.CreateClient(ctx, realmName, client)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Extract client UUID from Location header
	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	// Test getting a non-existent role
	role, resp, err := c.Clients.GetClientRole(ctx, realmName, clientUUID, "nonexistent-role-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent role")
	require.Nil(t, role, "Role should be nil for error response")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestClientsClient_DeleteClientRole_NotFound(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name
	realmName := fmt.Sprintf("test-realm-cli-role-del-nf-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Create test client
	clientId := fmt.Sprintf("test-client-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	client := keycloakv2.ClientRepresentation{
		ClientId: &clientId,
		Protocol: &protocol,
	}

	resp, err = c.Clients.CreateClient(ctx, realmName, client)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Extract client UUID from Location header
	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	// Test deleting a non-existent role
	resp, err = c.Clients.DeleteClientRole(ctx, realmName, clientUUID, "nonexistent-role-12345")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err), "Should return 404 error for non-existent role")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestClientsClient_GetClientByClientID(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-cli-by-id-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-client-by-id-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId: &clientId,
		Protocol: &protocol,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Found by clientId
	found, _, err := c.Clients.GetClientByClientID(ctx, realmName, clientId)
	require.NoError(t, err)
	require.NotNil(t, found)
	require.Equal(t, clientId, *found.ClientId)

	// Not found
	_, _, err = c.Clients.GetClientByClientID(ctx, realmName, "definitely-does-not-exist")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err))
}

func TestClientsClient_GetClientUUID(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-cli-uuid-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-client-uuid-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId: &clientId,
		Protocol: &protocol,
	})
	require.NoError(t, err)

	expectedUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, expectedUUID)

	// Found — UUID matches the one returned at creation
	uuid, err := c.Clients.GetClientUUID(ctx, realmName, clientId)
	require.NoError(t, err)
	require.Equal(t, expectedUUID, uuid)

	// Not found
	_, err = c.Clients.GetClientUUID(ctx, realmName, "definitely-does-not-exist")
	require.Error(t, err)
	require.True(t, keycloakv2.IsNotFound(err))
}

func TestClientsClient_UpdateAndScopeMethods(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-cli-update-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-client-update-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	originalName := "Original Name"
	publicClient := true

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:     &clientId,
		Name:         &originalName,
		Protocol:     &protocol,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	// UpdateClient
	updatedName := "Updated Name"
	resp, err = c.Clients.UpdateClient(ctx, realmName, clientUUID, keycloakv2.ClientRepresentation{
		ClientId:     &clientId,
		Name:         &updatedName,
		Protocol:     &protocol,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	updated, _, err := c.Clients.GetClient(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.Equal(t, updatedName, *updated.Name)

	// GetDefaultClientScopes
	defaultScopes, resp, err := c.Clients.GetDefaultClientScopes(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, defaultScopes)

	// GetOptionalClientScopes
	optionalScopes, resp, err := c.Clients.GetOptionalClientScopes(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, optionalScopes)

	// GetRealmClientScopes
	realmScopes, resp, err := c.Clients.GetRealmClientScopes(ctx, realmName)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Greater(t, len(realmScopes), 0, "realm should have at least one client scope")
}

func TestClientsClient_ServiceAccount(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-svc-acc-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-svc-acc-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	serviceAccounts := true
	publicClient := false

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:               &clientId,
		Protocol:               &protocol,
		ServiceAccountsEnabled: &serviceAccounts,
		PublicClient:           &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	user, resp, err := c.Clients.GetServiceAccountUser(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, user)
	require.NotNil(t, user.Id)
}

func TestClientsClient_ProtocolMapperCRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-mapper-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-client-mapper-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	publicClient := true

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:     &clientId,
		Protocol:     &protocol,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	mapperName := fmt.Sprintf("test-mapper-%d", time.Now().UnixNano())
	mapperProtocol := protocolOpenIDConnect
	mapperProtocolMapper := "oidc-hardcoded-claim-mapper"
	consentRequired := false

	mapper := keycloakv2.ProtocolMapperRepresentation{
		Name:            &mapperName,
		Protocol:        &mapperProtocol,
		ProtocolMapper:  &mapperProtocolMapper,
		ConsentRequired: &consentRequired,
		Config: &map[string]string{
			"claim.name":           "test-claim",
			"claim.value":          "test-value",
			"jsonType.label":       "String",
			"id.token.claim":       "true",
			"access.token.claim":   "true",
			"userinfo.token.claim": "true",
		},
	}

	// Create
	resp, err = c.Clients.CreateClientProtocolMapper(ctx, realmName, clientUUID, mapper)
	require.NoError(t, err)
	require.NotNil(t, resp)

	mapperID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, mapperID)

	// GetClientProtocolMappers — verify presence
	mappers, resp, err := c.Clients.GetClientProtocolMappers(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	found := false

	for _, m := range mappers {
		if m.Id != nil && *m.Id == mapperID {
			found = true
			break
		}
	}

	require.True(t, found, "created mapper should be in the list")

	// UpdateClientProtocolMapper — must include the ID in the body
	updatedName := mapperName + "-updated"
	mapper.Name = &updatedName
	mapper.Id = &mapperID

	resp, err = c.Clients.UpdateClientProtocolMapper(ctx, realmName, clientUUID, mapperID, mapper)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// DeleteClientProtocolMapper
	resp, err = c.Clients.DeleteClientProtocolMapper(ctx, realmName, clientUUID, mapperID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify gone
	mappers, _, err = c.Clients.GetClientProtocolMappers(ctx, realmName, clientUUID)
	require.NoError(t, err)

	for _, m := range mappers {
		if m.Id != nil && *m.Id == mapperID {
			t.Fatal("deleted mapper should not be in the list")
		}
	}
}

func TestClientsClient_ManagementPermissions(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-mgmt-perm-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-client-mgmt-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	publicClient := true

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:     &clientId,
		Protocol:     &protocol,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	// GetClientManagementPermissions — disabled by default
	perms, resp, err := c.Clients.GetClientManagementPermissions(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, perms)
	require.NotNil(t, perms.Enabled)
	require.False(t, *perms.Enabled)

	// UpdateClientManagementPermissions — enable
	enabledTrue := true
	updated, resp, err := c.Clients.UpdateClientManagementPermissions(
		ctx, realmName, clientUUID,
		keycloakv2.ManagementPermissionReference{
			Enabled: &enabledTrue,
		},
	)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, updated)
	require.NotNil(t, updated.Enabled)
	require.True(t, *updated.Enabled)
}

func TestClientsClient_RoleComposites(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-role-comp-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-client-comp-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	publicClient := true

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:     &clientId,
		Protocol:     &protocol,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	role1Name := "comp-role-1"
	role2Name := "comp-role-2"
	isComposite := true

	// Create composite role and sub-role
	_, err = c.Clients.CreateClientRole(ctx, realmName, clientUUID, keycloakv2.RoleRepresentation{
		Name:      &role1Name,
		Composite: &isComposite,
	})
	require.NoError(t, err)

	_, err = c.Clients.CreateClientRole(ctx, realmName, clientUUID, keycloakv2.RoleRepresentation{
		Name: &role2Name,
	})
	require.NoError(t, err)

	role2, _, err := c.Clients.GetClientRole(ctx, realmName, clientUUID, role2Name)
	require.NoError(t, err)
	require.NotNil(t, role2)

	// AddClientRoleComposites
	resp, err = c.Clients.AddClientRoleComposites(
		ctx, realmName, clientUUID, role1Name,
		[]keycloakv2.RoleRepresentation{*role2},
	)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// GetClientRoleComposites
	composites, resp, err := c.Clients.GetClientRoleComposites(ctx, realmName, clientUUID, role1Name)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, composites, 1)
	require.Equal(t, role2Name, *composites[0].Name)

	// DeleteClientRoleComposites
	resp, err = c.Clients.DeleteClientRoleComposites(
		ctx, realmName, clientUUID, role1Name,
		[]keycloakv2.RoleRepresentation{*role2},
	)
	require.NoError(t, err)
	require.NotNil(t, resp)

	composites, _, err = c.Clients.GetClientRoleComposites(ctx, realmName, clientUUID, role1Name)
	require.NoError(t, err)
	require.Empty(t, composites)
}

func TestClientsClient_AddDefaultClientScope(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-cli-scope-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-client-scope-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	publicClient := true

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:     &clientId,
		Protocol:     &protocol,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	// Get a realm scope to use
	realmScopes, _, err := c.Clients.GetRealmClientScopes(ctx, realmName)
	require.NoError(t, err)
	require.Greater(t, len(realmScopes), 0, "need at least one realm scope")

	// Find a scope not already in the default list
	defaultScopes, _, err := c.Clients.GetDefaultClientScopes(ctx, realmName, clientUUID)
	require.NoError(t, err)

	defaultScopeIDs := make(map[string]bool)

	for _, s := range defaultScopes {
		if s.Id != nil {
			defaultScopeIDs[*s.Id] = true
		}
	}

	var targetScope *keycloakv2.ClientScopeRepresentation

	for i := range realmScopes {
		if realmScopes[i].Id != nil && !defaultScopeIDs[*realmScopes[i].Id] {
			targetScope = &realmScopes[i]
			break
		}
	}

	if targetScope == nil {
		t.Skip("no realm scope available to add as default")
	}

	// AddDefaultClientScope — verify the API call succeeds
	resp, err = c.Clients.AddDefaultClientScope(ctx, realmName, clientUUID, *targetScope.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestClientsClient_AddOptionalClientScope(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-cli-optscope-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-client-optscope-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	publicClient := true

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:     &clientId,
		Protocol:     &protocol,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	// Get a realm scope not already in the optional list
	realmScopes, _, err := c.Clients.GetRealmClientScopes(ctx, realmName)
	require.NoError(t, err)
	require.Greater(t, len(realmScopes), 0, "need at least one realm scope")

	optionalScopes, _, err := c.Clients.GetOptionalClientScopes(ctx, realmName, clientUUID)
	require.NoError(t, err)

	optScopeIDs := make(map[string]bool)

	for _, s := range optionalScopes {
		if s.Id != nil {
			optScopeIDs[*s.Id] = true
		}
	}

	defaultScopes, _, err := c.Clients.GetDefaultClientScopes(ctx, realmName, clientUUID)
	require.NoError(t, err)

	defaultScopeIDs := make(map[string]bool)

	for _, s := range defaultScopes {
		if s.Id != nil {
			defaultScopeIDs[*s.Id] = true
		}
	}

	var targetScope *keycloakv2.ClientScopeRepresentation

	for i := range realmScopes {
		if realmScopes[i].Id != nil &&
			!optScopeIDs[*realmScopes[i].Id] &&
			!defaultScopeIDs[*realmScopes[i].Id] {
			targetScope = &realmScopes[i]
			break
		}
	}

	if targetScope == nil {
		t.Skip("no realm scope available to add as optional")
	}

	// AddOptionalClientScope — verify the API call succeeds
	resp, err = c.Clients.AddOptionalClientScope(ctx, realmName, clientUUID, *targetScope.Id)
	require.NoError(t, err)
	require.NotNil(t, resp)
}

func TestClientsClient_CreateClientRole_Conflict(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate unique realm name to avoid conflicts
	realmName := fmt.Sprintf("test-realm-cli-role-conflict-%d", time.Now().UnixNano())
	enabled := true

	// Ensure cleanup happens even if test fails
	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	// Create test realm
	realm := keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	}
	resp, err := c.Realms.CreateRealm(ctx, realm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Create test client
	clientId := fmt.Sprintf("test-client-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	client := keycloakv2.ClientRepresentation{
		ClientId: &clientId,
		Protocol: &protocol,
	}

	resp, err = c.Clients.CreateClient(ctx, realmName, client)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Extract client UUID from Location header
	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	roleName := "duplicate-client-role"
	role := keycloakv2.RoleRepresentation{
		Name: &roleName,
	}

	// Create the role
	resp, err = c.Clients.CreateClientRole(ctx, realmName, clientUUID, role)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Create the same role again - should conflict
	resp, err = c.Clients.CreateClientRole(ctx, realmName, clientUUID, role)
	require.Error(t, err)
	require.True(t, keycloakv2.IsConflict(err), "Should return 409 Conflict error for duplicate role")
	require.NotNil(t, resp, "Response should be present even for error")
}

func TestClientsClient_UpdateClientRole(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()

	realmName := fmt.Sprintf("test-realm-cli-role-update-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientId := fmt.Sprintf("test-client-role-update-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId: &clientId,
		Protocol: &protocol,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	roleName := "update-test-role"
	originalDesc := "original description"

	_, err = c.Clients.CreateClientRole(ctx, realmName, clientUUID, keycloakv2.RoleRepresentation{
		Name:        &roleName,
		Description: &originalDesc,
	})
	require.NoError(t, err)

	updatedDesc := "updated description"
	resp, err = c.Clients.UpdateClientRole(ctx, realmName, clientUUID, roleName, keycloakv2.RoleRepresentation{
		Name:        &roleName,
		Description: &updatedDesc,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	role, _, err := c.Clients.GetClientRole(ctx, realmName, clientUUID, roleName)
	require.NoError(t, err)
	require.Equal(t, updatedDesc, *role.Description)
}

func TestClientsClient_GetClientSecret(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-client-secret-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	// Create a confidential client.
	clientID := fmt.Sprintf("secret-client-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	publicClient := false

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:     &clientID,
		Protocol:     &protocol,
		Enabled:      &enabled,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)

	// GetClientSecret.
	cred, resp, err := c.Clients.GetClientSecret(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, cred)
	require.NotNil(t, cred.Value)
	require.NotEmpty(t, *cred.Value)
}

func TestClientsClient_RegenerateClientSecret(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-client-regen-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientID := fmt.Sprintf("regen-client-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	publicClient := false

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:     &clientID,
		Protocol:     &protocol,
		Enabled:      &enabled,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)

	// Get original secret.
	original, _, err := c.Clients.GetClientSecret(ctx, realmName, clientUUID)
	require.NoError(t, err)

	originalValue := *original.Value

	// Regenerate.
	regenerated, resp, err := c.Clients.RegenerateClientSecret(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, regenerated)
	require.NotNil(t, regenerated.Value)
	require.NotEqual(t, originalValue, *regenerated.Value, "secret should change after regeneration")
}

func TestClientsClient_GetClientSessions(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-client-sessions-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientID := fmt.Sprintf("sessions-client-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId: &clientID,
		Protocol: &protocol,
		Enabled:  &enabled,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)

	sessions, resp, err := c.Clients.GetClientSessions(ctx, realmName, clientUUID, nil)
	require.NoError(t, err)
	require.NotNil(t, resp)
	// A fresh client has no sessions.
	require.Empty(t, sessions)
}

func TestClientsClient_GetClientInstallationProvider(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	c, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-client-install-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = c.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = c.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientID := fmt.Sprintf("install-client-%d", time.Now().UnixNano())
	protocol := protocolOpenIDConnect
	publicClient := false

	resp, err := c.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:     &clientID,
		Protocol:     &protocol,
		Enabled:      &enabled,
		PublicClient: &publicClient,
	})
	require.NoError(t, err)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)

	data, resp, err := c.Clients.GetClientInstallationProvider(ctx, realmName, clientUUID, "keycloak-oidc-keycloak-json")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotEmpty(t, data, "installation JSON should not be empty")
}
