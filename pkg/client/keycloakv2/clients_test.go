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
