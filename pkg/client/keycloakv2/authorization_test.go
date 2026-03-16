package keycloakv2_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/testutils"
	"github.com/stretchr/testify/require"
)

// createAuthzClient sets up a Keycloak client with Authorization Services enabled and returns
// the keycloak client, the realm name, and the client UUID.
func createAuthzClient(
	t *testing.T,
	kc *keycloakv2.KeycloakClient,
	ctx context.Context,
	realmName string,
) string {
	t.Helper()

	enabled := true
	authzEnabled := true
	serviceAccountsEnabled := true
	protocol := protocolOpenIDConnect
	clientID := fmt.Sprintf("authz-client-%d", time.Now().UnixNano())
	bearerOnly := false
	publicClient := false

	resp, err := kc.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId:                     &clientID,
		Protocol:                     &protocol,
		Enabled:                      &enabled,
		AuthorizationServicesEnabled: &authzEnabled,
		ServiceAccountsEnabled:       &serviceAccountsEnabled,
		BearerOnly:                   &bearerOnly,
		PublicClient:                 &publicClient,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	clientUUID := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, clientUUID)

	return clientUUID
}

func TestAuthorizationClient_ScopeCRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	kc, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-authz-scope-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = kc.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = kc.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientUUID := createAuthzClient(t, kc, ctx, realmName)

	scopeName := fmt.Sprintf("test-scope-%d", time.Now().UnixNano())

	// Create scope — ID comes from the response body, not Location header
	resp, err := kc.Authorization.CreateScope(ctx, realmName, clientUUID, keycloakv2.ScopeRepresentation{
		Name: &scopeName,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	var createdScope keycloakv2.ScopeRepresentation

	require.NoError(t, json.Unmarshal(resp.Body, &createdScope))
	require.NotNil(t, createdScope.Id)

	scopeID := *createdScope.Id
	require.NotEmpty(t, scopeID)

	// GetScopes — verify our scope is present
	scopes, resp, err := kc.Authorization.GetScopes(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	found := false

	for _, s := range scopes {
		if s.Name != nil && *s.Name == scopeName {
			found = true
			break
		}
	}

	require.True(t, found, "created scope should be in the list")

	// DeleteScope
	resp, err = kc.Authorization.DeleteScope(ctx, realmName, clientUUID, scopeID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify gone
	scopes, _, err = kc.Authorization.GetScopes(ctx, realmName, clientUUID)
	require.NoError(t, err)

	for _, s := range scopes {
		if s.Name != nil && *s.Name == scopeName {
			t.Fatal("deleted scope should not be in the list")
		}
	}
}

func TestAuthorizationClient_ResourceCRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	kc, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-authz-res-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = kc.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = kc.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientUUID := createAuthzClient(t, kc, ctx, realmName)

	resourceName := fmt.Sprintf("test-resource-%d", time.Now().UnixNano())
	resourceType := "urn:test:resource:test"

	// Create resource
	created, resp, err := kc.Authorization.CreateResource(ctx, realmName, clientUUID, keycloakv2.ResourceRepresentation{
		Name: &resourceName,
		Type: &resourceType,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, created)
	require.NotNil(t, created.UnderscoreId)

	resourceID := *created.UnderscoreId

	// GetResources — verify our resource is present
	resources, resp, err := kc.Authorization.GetResources(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	found := false

	for _, r := range resources {
		if r.Name != nil && *r.Name == resourceName {
			found = true
			break
		}
	}

	require.True(t, found, "created resource should be in the list")

	// UpdateResource
	updatedName := resourceName + "-updated"
	resp, err = kc.Authorization.UpdateResource(ctx, realmName, clientUUID, resourceID, keycloakv2.ResourceRepresentation{
		UnderscoreId: &resourceID,
		Name:         &updatedName,
		Type:         &resourceType,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify update
	resources, _, err = kc.Authorization.GetResources(ctx, realmName, clientUUID)
	require.NoError(t, err)

	foundUpdated := false

	for _, r := range resources {
		if r.Name != nil && *r.Name == updatedName {
			foundUpdated = true
			break
		}
	}

	require.True(t, foundUpdated, "resource name should be updated")

	// DeleteResource
	resp, err = kc.Authorization.DeleteResource(ctx, realmName, clientUUID, resourceID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify gone
	resources, _, err = kc.Authorization.GetResources(ctx, realmName, clientUUID)
	require.NoError(t, err)

	for _, r := range resources {
		if r.UnderscoreId != nil && *r.UnderscoreId == resourceID {
			t.Fatal("deleted resource should not be in the list")
		}
	}
}

func TestAuthorizationClient_PolicyCRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	kc, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-authz-policy-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = kc.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = kc.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientUUID := createAuthzClient(t, kc, ctx, realmName)

	// GetPolicies — baseline (Keycloak creates a Default Policy automatically)
	policies, resp, err := kc.Authorization.GetPolicies(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	baseline := len(policies)

	policyName := fmt.Sprintf("test-policy-%d", time.Now().UnixNano())
	policyType := "time"
	logic := keycloakv2.Logic("POSITIVE")
	decisionStrategy := keycloakv2.DecisionStrategy("UNANIMOUS")

	policy := keycloakv2.PolicyRepresentation{
		Name:             &policyName,
		Type:             &policyType,
		Logic:            &logic,
		DecisionStrategy: &decisionStrategy,
	}

	// CreatePolicy
	created, resp, err := kc.Authorization.CreatePolicy(ctx, realmName, clientUUID, policyType, policy)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, created)
	require.NotNil(t, created.Id)

	policyID := *created.Id

	// GetPolicies — verify count increased
	policies, _, err = kc.Authorization.GetPolicies(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.Greater(t, len(policies), baseline, "policy count should have increased")

	// UpdatePolicy
	updatedName := policyName + "-updated"
	policy.Name = &updatedName
	resp, err = kc.Authorization.UpdatePolicy(ctx, realmName, clientUUID, policyType, policyID, policy)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// DeletePolicy
	resp, err = kc.Authorization.DeletePolicy(ctx, realmName, clientUUID, policyID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify gone
	policies, _, err = kc.Authorization.GetPolicies(ctx, realmName, clientUUID)
	require.NoError(t, err)

	for _, p := range policies {
		if p.Id != nil && *p.Id == policyID {
			t.Fatal("deleted policy should not be in the list")
		}
	}
}

// createPlainClient creates a basic (non-authz) Keycloak client and returns its UUID.
func createPlainClient(
	t *testing.T,
	kc *keycloakv2.KeycloakClient,
	ctx context.Context,
	realmName string,
) string {
	t.Helper()

	enabled := true
	protocol := protocolOpenIDConnect
	clientID := fmt.Sprintf("plain-client-%d", time.Now().UnixNano())

	resp, err := kc.Clients.CreateClient(ctx, realmName, keycloakv2.ClientRepresentation{
		ClientId: &clientID,
		Protocol: &protocol,
		Enabled:  &enabled,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	uuid := keycloakv2.GetResourceIDFromResponse(resp)
	require.NotEmpty(t, uuid)

	return uuid
}

func TestAuthorizationClient_PolicyTypes(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	kc, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-policy-types-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = kc.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = kc.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientUUID := createAuthzClient(t, kc, ctx, realmName)

	// --- Prerequisites ---

	// Realm role
	roleName := fmt.Sprintf("policy-role-%d", time.Now().UnixNano())
	_, err = kc.Roles.CreateRealmRole(ctx, realmName, keycloakv2.RoleRepresentation{Name: &roleName})
	require.NoError(t, err)

	role, _, err := kc.Roles.GetRealmRole(ctx, realmName, roleName)
	require.NoError(t, err)
	require.NotNil(t, role.Id)

	// Group
	groupName := fmt.Sprintf("policy-group-%d", time.Now().UnixNano())
	groupResp, err := kc.Groups.CreateGroup(ctx, realmName, keycloakv2.GroupRepresentation{Name: &groupName})
	require.NoError(t, err)

	groupID := keycloakv2.GetResourceIDFromResponse(groupResp)
	require.NotEmpty(t, groupID)

	// Plain client (for client policy)
	plainClientUUID := createPlainClient(t, kc, ctx, realmName)

	// User (for user policy)
	username := fmt.Sprintf("policy-user-%d", time.Now().UnixNano())
	userResp, err := kc.Users.CreateUser(ctx, realmName, keycloakv2.UserRepresentation{Username: &username})
	require.NoError(t, err)

	userID := keycloakv2.GetResourceIDFromResponse(userResp)
	require.NotEmpty(t, userID)

	// Seed time policy (required by the aggregate sub-test)
	seedName := fmt.Sprintf("seed-time-policy-%d", time.Now().UnixNano())
	seedBody := keycloakv2.TimePolicyBody{
		PolicyBodyBase: keycloakv2.PolicyBodyBase{
			Name: seedName,
			Type: "time",
		},
		Hour:    "8",
		HourEnd: "18",
	}

	seedPolicy, _, err := kc.Authorization.CreatePolicy(ctx, realmName, clientUUID, "time", seedBody)
	require.NoError(t, err)
	require.NotNil(t, seedPolicy)
	require.NotNil(t, seedPolicy.Id)

	// --- Sub-tests ---

	t.Run("time", func(t *testing.T) {
		t.Parallel()

		name := fmt.Sprintf("time-policy-%d", time.Now().UnixNano())
		body := keycloakv2.TimePolicyBody{
			PolicyBodyBase: keycloakv2.PolicyBodyBase{
				Name:             name,
				Type:             "time",
				DecisionStrategy: keycloakv2.DecisionStrategy("UNANIMOUS"),
				Logic:            keycloakv2.Logic("POSITIVE"),
			},
			Hour:    "9",
			HourEnd: "17",
		}

		created, _, err := kc.Authorization.CreatePolicy(ctx, realmName, clientUUID, "time", body)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.NotNil(t, created.Id)

		// GET and verify all fields
		resp, err := kc.Authorization.GetPolicy(ctx, realmName, clientUUID, "time", *created.Id)
		require.NoError(t, err)

		var got keycloakv2.TimePolicyBody

		require.NoError(t, json.Unmarshal(resp.Body, &got))
		require.Equal(t, name, got.Name)
		require.Equal(t, "time", got.Type)
		require.Equal(t, keycloakv2.DecisionStrategy("UNANIMOUS"), got.DecisionStrategy)
		require.Equal(t, keycloakv2.Logic("POSITIVE"), got.Logic)
		require.Equal(t, "9", got.Hour)
		require.Equal(t, "17", got.HourEnd)

		body.Name = name + "-updated"
		_, err = kc.Authorization.UpdatePolicy(ctx, realmName, clientUUID, "time", *created.Id, body)
		require.NoError(t, err)

		_, err = kc.Authorization.DeletePolicy(ctx, realmName, clientUUID, *created.Id)
		require.NoError(t, err)
	})

	t.Run("aggregate", func(t *testing.T) {
		t.Parallel()

		name := fmt.Sprintf("aggregate-policy-%d", time.Now().UnixNano())
		body := keycloakv2.AggregatePolicyBody{
			PolicyBodyBase: keycloakv2.PolicyBodyBase{
				Name:             name,
				Type:             "aggregate",
				DecisionStrategy: keycloakv2.DecisionStrategy("UNANIMOUS"),
				Logic:            keycloakv2.Logic("POSITIVE"),
			},
			Policies: []string{*seedPolicy.Id},
		}

		created, _, err := kc.Authorization.CreatePolicy(ctx, realmName, clientUUID, "aggregate", body)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.NotNil(t, created.Id)

		// GET and verify base fields (Keycloak does not return "policies" in the type-specific GET)
		resp, err := kc.Authorization.GetPolicy(ctx, realmName, clientUUID, "aggregate", *created.Id)
		require.NoError(t, err)

		var got keycloakv2.AggregatePolicyBody

		require.NoError(t, json.Unmarshal(resp.Body, &got))
		require.Equal(t, name, got.Name)
		require.Equal(t, "aggregate", got.Type)
		require.Equal(t, keycloakv2.DecisionStrategy("UNANIMOUS"), got.DecisionStrategy)
		require.Equal(t, keycloakv2.Logic("POSITIVE"), got.Logic)

		body.Name = name + "-updated"
		_, err = kc.Authorization.UpdatePolicy(ctx, realmName, clientUUID, "aggregate", *created.Id, body)
		require.NoError(t, err)

		_, err = kc.Authorization.DeletePolicy(ctx, realmName, clientUUID, *created.Id)
		require.NoError(t, err)
	})

	t.Run("client", func(t *testing.T) { //nolint:dupl // each policy subtest uses a distinct body type
		t.Parallel()

		name := fmt.Sprintf("client-policy-%d", time.Now().UnixNano())
		body := keycloakv2.ClientPolicyBody{
			PolicyBodyBase: keycloakv2.PolicyBodyBase{
				Name:             name,
				Type:             "client",
				DecisionStrategy: keycloakv2.DecisionStrategy("UNANIMOUS"),
				Logic:            keycloakv2.Logic("POSITIVE"),
			},
			Clients: []string{plainClientUUID},
		}

		created, _, err := kc.Authorization.CreatePolicy(ctx, realmName, clientUUID, "client", body)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.NotNil(t, created.Id)

		// GET and verify all fields
		resp, err := kc.Authorization.GetPolicy(ctx, realmName, clientUUID, "client", *created.Id)
		require.NoError(t, err)

		var got keycloakv2.ClientPolicyBody

		require.NoError(t, json.Unmarshal(resp.Body, &got))
		require.Equal(t, name, got.Name)
		require.Equal(t, "client", got.Type)
		require.Equal(t, keycloakv2.DecisionStrategy("UNANIMOUS"), got.DecisionStrategy)
		require.Equal(t, keycloakv2.Logic("POSITIVE"), got.Logic)
		require.Equal(t, []string{plainClientUUID}, got.Clients)

		body.Name = name + "-updated"
		_, err = kc.Authorization.UpdatePolicy(ctx, realmName, clientUUID, "client", *created.Id, body)
		require.NoError(t, err)

		_, err = kc.Authorization.DeletePolicy(ctx, realmName, clientUUID, *created.Id)
		require.NoError(t, err)
	})

	t.Run("group", func(t *testing.T) {
		t.Parallel()

		name := fmt.Sprintf("group-policy-%d", time.Now().UnixNano())
		body := keycloakv2.GroupPolicyBody{
			PolicyBodyBase: keycloakv2.PolicyBodyBase{
				Name:             name,
				Type:             "group",
				DecisionStrategy: keycloakv2.DecisionStrategy("UNANIMOUS"),
				Logic:            keycloakv2.Logic("POSITIVE"),
			},
			Groups: []keycloakv2.GroupDefinition{
				{ID: groupID, ExtendChildren: false},
			},
		}

		created, _, err := kc.Authorization.CreatePolicy(ctx, realmName, clientUUID, "group", body)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.NotNil(t, created.Id)

		// GET and verify all fields
		resp, err := kc.Authorization.GetPolicy(ctx, realmName, clientUUID, "group", *created.Id)
		require.NoError(t, err)

		var got keycloakv2.GroupPolicyBody

		require.NoError(t, json.Unmarshal(resp.Body, &got))
		require.Equal(t, name, got.Name)
		require.Equal(t, "group", got.Type)
		require.Equal(t, keycloakv2.DecisionStrategy("UNANIMOUS"), got.DecisionStrategy)
		require.Equal(t, keycloakv2.Logic("POSITIVE"), got.Logic)
		require.Len(t, got.Groups, 1)
		require.Equal(t, groupID, got.Groups[0].ID)
		require.False(t, got.Groups[0].ExtendChildren)

		body.Name = name + "-updated"
		_, err = kc.Authorization.UpdatePolicy(ctx, realmName, clientUUID, "group", *created.Id, body)
		require.NoError(t, err)

		_, err = kc.Authorization.DeletePolicy(ctx, realmName, clientUUID, *created.Id)
		require.NoError(t, err)
	})

	t.Run("role", func(t *testing.T) {
		t.Parallel()

		name := fmt.Sprintf("role-policy-%d", time.Now().UnixNano())
		body := keycloakv2.RolePolicyBody{
			PolicyBodyBase: keycloakv2.PolicyBodyBase{
				Name:             name,
				Type:             "role",
				DecisionStrategy: keycloakv2.DecisionStrategy("UNANIMOUS"),
				Logic:            keycloakv2.Logic("POSITIVE"),
			},
			Roles: []keycloakv2.RoleDefinition{
				{ID: *role.Id, Required: false},
			},
		}

		created, _, err := kc.Authorization.CreatePolicy(ctx, realmName, clientUUID, "role", body)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.NotNil(t, created.Id)

		// GET and verify all fields
		resp, err := kc.Authorization.GetPolicy(ctx, realmName, clientUUID, "role", *created.Id)
		require.NoError(t, err)

		var got keycloakv2.RolePolicyBody

		require.NoError(t, json.Unmarshal(resp.Body, &got))
		require.Equal(t, name, got.Name)
		require.Equal(t, "role", got.Type)
		require.Equal(t, keycloakv2.DecisionStrategy("UNANIMOUS"), got.DecisionStrategy)
		require.Equal(t, keycloakv2.Logic("POSITIVE"), got.Logic)
		require.Len(t, got.Roles, 1)
		require.Equal(t, *role.Id, got.Roles[0].ID)
		require.False(t, got.Roles[0].Required)

		body.Name = name + "-updated"
		_, err = kc.Authorization.UpdatePolicy(ctx, realmName, clientUUID, "role", *created.Id, body)
		require.NoError(t, err)

		_, err = kc.Authorization.DeletePolicy(ctx, realmName, clientUUID, *created.Id)
		require.NoError(t, err)
	})

	t.Run("user", func(t *testing.T) { //nolint:dupl // each policy subtest uses a distinct body type
		t.Parallel()

		name := fmt.Sprintf("user-policy-%d", time.Now().UnixNano())
		body := keycloakv2.UserPolicyBody{
			PolicyBodyBase: keycloakv2.PolicyBodyBase{
				Name:             name,
				Type:             "user",
				DecisionStrategy: keycloakv2.DecisionStrategy("UNANIMOUS"),
				Logic:            keycloakv2.Logic("POSITIVE"),
			},
			Users: []string{userID},
		}

		created, _, err := kc.Authorization.CreatePolicy(ctx, realmName, clientUUID, "user", body)
		require.NoError(t, err)
		require.NotNil(t, created)
		require.NotNil(t, created.Id)

		// GET and verify all fields
		resp, err := kc.Authorization.GetPolicy(ctx, realmName, clientUUID, "user", *created.Id)
		require.NoError(t, err)

		var got keycloakv2.UserPolicyBody

		require.NoError(t, json.Unmarshal(resp.Body, &got))
		require.Equal(t, name, got.Name)
		require.Equal(t, "user", got.Type)
		require.Equal(t, keycloakv2.DecisionStrategy("UNANIMOUS"), got.DecisionStrategy)
		require.Equal(t, keycloakv2.Logic("POSITIVE"), got.Logic)
		require.Equal(t, []string{userID}, got.Users)

		body.Name = name + "-updated"
		_, err = kc.Authorization.UpdatePolicy(ctx, realmName, clientUUID, "user", *created.Id, body)
		require.NoError(t, err)

		_, err = kc.Authorization.DeletePolicy(ctx, realmName, clientUUID, *created.Id)
		require.NoError(t, err)
	})
}

func TestAuthorizationClient_PermissionCRUD(t *testing.T) {
	keycloakURL := testutils.GetKeycloakURLOrSkip(t)
	t.Parallel()

	kc, err := keycloakv2.NewKeycloakClient(
		context.Background(),
		keycloakURL,
		keycloakv2.DefaultAdminClientID,
		keycloakv2.WithPasswordGrant(keycloakv2.DefaultAdminUsername, keycloakv2.DefaultAdminPassword),
	)
	require.NoError(t, err)

	ctx := context.Background()
	realmName := fmt.Sprintf("test-realm-authz-perm-%d", time.Now().UnixNano())
	enabled := true

	t.Cleanup(func() {
		_, _ = kc.Realms.DeleteRealm(context.Background(), realmName)
	})

	_, err = kc.Realms.CreateRealm(ctx, keycloakv2.RealmRepresentation{
		Realm:   &realmName,
		Enabled: &enabled,
	})
	require.NoError(t, err)

	clientUUID := createAuthzClient(t, kc, ctx, realmName)

	// GetPermissions — baseline
	permissions, resp, err := kc.Authorization.GetPermissions(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	baseline := len(permissions)

	permName := fmt.Sprintf("test-permission-%d", time.Now().UnixNano())
	permType := "scope"
	decisionStrategy := keycloakv2.DecisionStrategy("UNANIMOUS")

	perm := keycloakv2.PolicyRepresentation{
		Name:             &permName,
		Type:             &permType,
		DecisionStrategy: &decisionStrategy,
	}

	// CreatePermission
	created, resp, err := kc.Authorization.CreatePermission(ctx, realmName, clientUUID, permType, perm)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, created)
	require.NotNil(t, created.Id)

	permID := *created.Id

	// GetPermissions — verify count increased
	permissions, _, err = kc.Authorization.GetPermissions(ctx, realmName, clientUUID)
	require.NoError(t, err)
	require.Greater(t, len(permissions), baseline, "permission count should have increased")

	// UpdatePermission
	updatedName := permName + "-updated"
	perm.Name = &updatedName
	resp, err = kc.Authorization.UpdatePermission(ctx, realmName, clientUUID, permType, permID, perm)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// DeletePermission
	resp, err = kc.Authorization.DeletePermission(ctx, realmName, clientUUID, permID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify gone
	permissions, _, err = kc.Authorization.GetPermissions(ctx, realmName, clientUUID)
	require.NoError(t, err)

	for _, p := range permissions {
		if p.Id != nil && *p.Id == permID {
			t.Fatal("deleted permission should not be in the list")
		}
	}
}
