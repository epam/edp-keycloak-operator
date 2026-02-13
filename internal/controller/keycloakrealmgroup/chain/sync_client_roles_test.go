package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestSyncClientRoles_Serve_AddRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: []string{"role1", "role2"}},
	}

	// Empty role mappings (no clients have roles yet)
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(&keycloakv2.MappingsRepresentation{}, nil, nil)

	// Resolve client UUID (since client not in mappings)
	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakv2.GetClientsParams{ClientId: ptr.To("test-client")},
	).Return([]keycloakv2.ClientRepresentation{
		{Id: ptr.To("client-uuid"), ClientId: ptr.To("test-client")},
	}, nil, nil)

	// Get role1
	mockClients.EXPECT().GetClientRole(
		context.Background(), "test-realm", "client-uuid", "role1",
	).Return(&keycloakv2.RoleRepresentation{
		Id:   ptr.To("role1-id"),
		Name: ptr.To("role1"),
	}, nil, nil)

	// Get role2
	mockClients.EXPECT().GetClientRole(
		context.Background(), "test-realm", "client-uuid", "role2",
	).Return(&keycloakv2.RoleRepresentation{
		Id:   ptr.To("role2-id"),
		Name: ptr.To("role2"),
	}, nil, nil)

	// Add both roles
	mockGroups.EXPECT().AddClientRoleMappings(
		context.Background(), "test-realm", "group-123", "client-uuid",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
			{Id: ptr.To("role2-id"), Name: ptr.To("role2")},
		},
	).Return(nil, nil)

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncClientRoles_Serve_RemoveRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: []string{}}, // Empty - remove all
	}

	// Role mappings with existing client roles
	clientMappings := map[string]keycloakv2.ClientMappingsRepresentation{
		"test-client": {
			Id:     ptr.To("client-uuid"),
			Client: ptr.To("test-client"),
			Mappings: &[]keycloakv2.RoleRepresentation{
				{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
			},
		},
	}
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(&keycloakv2.MappingsRepresentation{
		ClientMappings: &clientMappings,
	}, nil, nil)

	// Remove old role
	mockGroups.EXPECT().DeleteClientRoleMappings(
		context.Background(), "test-realm", "group-123", "client-uuid",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, nil)

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncClientRoles_Serve_RemoveUnclaimedClient(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{} // No clients claimed

	// Existing role mappings with a client
	clientMappings := map[string]keycloakv2.ClientMappingsRepresentation{
		"old-client": {
			Id:     ptr.To("old-client-uuid"),
			Client: ptr.To("old-client"),
			Mappings: &[]keycloakv2.RoleRepresentation{
				{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
			},
		},
	}
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(&keycloakv2.MappingsRepresentation{
		ClientMappings: &clientMappings,
	}, nil, nil)

	// Delete roles for unclaimed client
	mockGroups.EXPECT().DeleteClientRoleMappings(
		context.Background(), "test-realm", "group-123", "old-client-uuid",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, nil)

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncClientRoles_Serve_MultipleClients(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{
		{ClientID: "client1", Roles: []string{"role1"}},
		{ClientID: "client2", Roles: []string{"role2"}},
	}

	// Empty role mappings
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(&keycloakv2.MappingsRepresentation{}, nil, nil)

	// Client1: Resolve UUID (since not in mappings)
	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakv2.GetClientsParams{ClientId: ptr.To("client1")},
	).Return([]keycloakv2.ClientRepresentation{
		{Id: ptr.To("client1-uuid"), ClientId: ptr.To("client1")},
	}, nil, nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), "test-realm", "client1-uuid", "role1",
	).Return(&keycloakv2.RoleRepresentation{
		Id: ptr.To("role1-id"), Name: ptr.To("role1"),
	}, nil, nil)

	mockGroups.EXPECT().AddClientRoleMappings(
		context.Background(), "test-realm", "group-123", "client1-uuid",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
		},
	).Return(nil, nil)

	// Client2: Resolve UUID (since not in mappings)
	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakv2.GetClientsParams{ClientId: ptr.To("client2")},
	).Return([]keycloakv2.ClientRepresentation{
		{Id: ptr.To("client2-uuid"), ClientId: ptr.To("client2")},
	}, nil, nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), "test-realm", "client2-uuid", "role2",
	).Return(&keycloakv2.RoleRepresentation{
		Id: ptr.To("role2-id"), Name: ptr.To("role2"),
	}, nil, nil)

	mockGroups.EXPECT().AddClientRoleMappings(
		context.Background(), "test-realm", "group-123", "client2-uuid",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("role2-id"), Name: ptr.To("role2")},
		},
	).Return(nil, nil)

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncClientRoles_Serve_ErrorGettingRoleMappings(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: []string{"role1"}},
	}

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get role mappings")
}

func TestSyncClientRoles_Serve_ErrorResolvingClientUUID(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{
		{ClientID: "nonexistent-client", Roles: []string{"role1"}},
	}

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(&keycloakv2.MappingsRepresentation{}, nil, nil)

	// Client not found
	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakv2.GetClientsParams{ClientId: ptr.To("nonexistent-client")},
	).Return([]keycloakv2.ClientRepresentation{}, nil, nil)

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "client \"nonexistent-client\" not found")
}

func TestSyncClientRoles_Serve_ErrorGettingClientRoleMappings(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: []string{"role1"}},
	}

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(&keycloakv2.MappingsRepresentation{}, nil, nil)

	// Resolve client UUID (since not in mappings)
	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakv2.GetClientsParams{ClientId: ptr.To("test-client")},
	).Return([]keycloakv2.ClientRepresentation{
		{Id: ptr.To("client-uuid"), ClientId: ptr.To("test-client")},
	}, nil, nil)

	// This test now triggers an error getting the client role (no longer gets client role mappings)
	mockClients.EXPECT().GetClientRole(
		context.Background(), "test-realm", "client-uuid", "role1",
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get client role")
}

func TestSyncClientRoles_Serve_ErrorGettingClientRole(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: []string{"nonexistent-role"}},
	}

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(&keycloakv2.MappingsRepresentation{}, nil, nil)

	// Resolve client UUID (since not in mappings)
	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakv2.GetClientsParams{ClientId: ptr.To("test-client")},
	).Return([]keycloakv2.ClientRepresentation{
		{Id: ptr.To("client-uuid"), ClientId: ptr.To("test-client")},
	}, nil, nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), "test-realm", "client-uuid", "nonexistent-role",
	).Return(nil, nil, errors.New("role not found"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get client role")
}

func TestSyncClientRoles_Serve_ErrorAddingClientRoleMappings(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: []string{"role1"}},
	}

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(&keycloakv2.MappingsRepresentation{}, nil, nil)

	// Resolve client UUID (since not in mappings)
	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakv2.GetClientsParams{ClientId: ptr.To("test-client")},
	).Return([]keycloakv2.ClientRepresentation{
		{Id: ptr.To("client-uuid"), ClientId: ptr.To("test-client")},
	}, nil, nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), "test-realm", "client-uuid", "role1",
	).Return(&keycloakv2.RoleRepresentation{
		Id: ptr.To("role1-id"), Name: ptr.To("role1"),
	}, nil, nil)

	mockGroups.EXPECT().AddClientRoleMappings(
		context.Background(), "test-realm", "group-123", "client-uuid",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
		},
	).Return(nil, errors.New("add failed"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to add client role mappings")
}

func TestSyncClientRoles_Serve_ErrorDeletingClientRoleMappings(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = []keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: []string{}}, // Remove all
	}

	// Role mappings with existing client roles
	clientMappings := map[string]keycloakv2.ClientMappingsRepresentation{
		"test-client": {
			Id:     ptr.To("client-uuid"),
			Client: ptr.To("test-client"),
			Mappings: &[]keycloakv2.RoleRepresentation{
				{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
			},
		},
	}
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(&keycloakv2.MappingsRepresentation{
		ClientMappings: &clientMappings,
	}, nil, nil)

	mockGroups.EXPECT().DeleteClientRoleMappings(
		context.Background(), "test-realm", "group-123", "client-uuid",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, errors.New("delete failed"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to delete client role mappings")
}
