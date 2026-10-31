package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/fakehttp"
)

func TestSyncClientRoles_Serve_AddRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{"role1", "role2"})},
	})

	// Empty role mappings (no clients have roles yet)
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{}, nil, nil)

	// Resolve client UUID (since client not in mappings)
	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "test-client",
	).Return("client-uuid", nil)

	// Get role1
	mockClients.EXPECT().GetClientRole(
		context.Background(), testRealmName, "client-uuid", "role1",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("role1-id"),
		Name: ptr.To("role1"),
	}, nil, nil)

	// Get role2
	mockClients.EXPECT().GetClientRole(
		context.Background(), testRealmName, "client-uuid", "role2",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("role2-id"),
		Name: ptr.To("role2"),
	}, nil, nil)

	// Add both roles
	mockGroups.EXPECT().AddClientRoleMappings(
		context.Background(), testRealmName, testGroupID, "client-uuid",
		[]keycloakapi.RoleRepresentation{
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

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{})}, // Empty - remove all
	})

	// Role mappings with existing client roles
	clientMappings := map[string]keycloakapi.ClientMappingsRepresentation{
		"test-client": {
			Id:     ptr.To("client-uuid"),
			Client: ptr.To("test-client"),
			Mappings: &[]keycloakapi.RoleRepresentation{
				{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
			},
		},
	}
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{
		ClientMappings: &clientMappings,
	}, nil, nil)

	// Remove old role
	mockGroups.EXPECT().DeleteClientRoleMappings(
		context.Background(), testRealmName, testGroupID, "client-uuid",
		[]keycloakapi.RoleRepresentation{
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

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{}) // No clients claimed

	// Existing role mappings with a client
	clientMappings := map[string]keycloakapi.ClientMappingsRepresentation{
		"old-client": {
			Id:     ptr.To("old-client-uuid"),
			Client: ptr.To("old-client"),
			Mappings: &[]keycloakapi.RoleRepresentation{
				{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
			},
		},
	}
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{
		ClientMappings: &clientMappings,
	}, nil, nil)

	// Delete roles for unclaimed client
	mockGroups.EXPECT().DeleteClientRoleMappings(
		context.Background(), testRealmName, testGroupID, "old-client-uuid",
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, nil)

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

// A client named with no role list is neither synced nor swept.
func TestSyncClientRoles_Serve_MentionedClientWithoutRolesIsLeftAlone(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "unmanaged-client"},
	})

	clientMappings := map[string]keycloakapi.ClientMappingsRepresentation{
		"unmanaged-client": {
			Id:     ptr.To("unmanaged-client-uuid"),
			Client: ptr.To("unmanaged-client"),
			Mappings: &[]keycloakapi.RoleRepresentation{
				{Id: ptr.To("kept-role-id"), Name: ptr.To("kept-role")},
			},
		},
	}
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{
		ClientMappings: &clientMappings,
	}, nil, nil)

	// No DeleteClientRoleMappings expectation: any delete fails this test.

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncClientRoles_Serve_MultipleClients(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "client1", Roles: ptr.To([]string{"role1"})},
		{ClientID: "client2", Roles: ptr.To([]string{"role2"})},
	})

	// Empty role mappings
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{}, nil, nil)

	// Client1: Resolve UUID (since not in mappings)
	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "client1",
	).Return("client1-uuid", nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), testRealmName, "client1-uuid", "role1",
	).Return(&keycloakapi.RoleRepresentation{
		Id: ptr.To("role1-id"), Name: ptr.To("role1"),
	}, nil, nil)

	mockGroups.EXPECT().AddClientRoleMappings(
		context.Background(), testRealmName, testGroupID, "client1-uuid",
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
		},
	).Return(nil, nil)

	// Client2: Resolve UUID (since not in mappings)
	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "client2",
	).Return("client2-uuid", nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), testRealmName, "client2-uuid", "role2",
	).Return(&keycloakapi.RoleRepresentation{
		Id: ptr.To("role2-id"), Name: ptr.To("role2"),
	}, nil, nil)

	mockGroups.EXPECT().AddClientRoleMappings(
		context.Background(), testRealmName, testGroupID, "client2-uuid",
		[]keycloakapi.RoleRepresentation{
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

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{"role1"})},
	})

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get role mappings")
}

func TestSyncClientRoles_Serve_ErrorResolvingClientUUID(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "nonexistent-client", Roles: ptr.To([]string{"role1"})},
	})

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{}, nil, nil)

	// Client not found
	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "nonexistent-client",
	).Return("", keycloakapi.ErrNotFound)

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "client \"nonexistent-client\" not found")
	assert.ErrorContains(t, err, "in realm \"test-realm\"")
	require.ErrorIs(t, err, keycloakapi.ErrNotFound)
}

func TestSyncClientRoles_Serve_ErrorGettingClient(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{"role1"})},
	})

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{}, nil, nil)

	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "test-client",
	).Return("", errors.New("api error"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unable to get client \"test-client\": api error")
	require.NotErrorIs(t, err, keycloakapi.ErrNotFound)
}

func TestSyncClientRoles_Serve_BlankClientIDSkipsLookup(t *testing.T) {
	const wantErr = `unable to sync client roles for client "   ": ` +
		`client "   " not found in realm "test-realm": not found`

	tests := []struct {
		name    string
		roles   *[]string
		wantErr string
	}{
		{
			name:    "roles to assign",
			roles:   ptr.To([]string{"r1"}),
			wantErr: wantErr,
		},
		{
			name:    "empty roles",
			roles:   ptr.To([]string{}),
			wantErr: wantErr,
		},
		{
			name: "unmanaged entry is skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := fakehttp.NewServerBuilder().
				AddKeycloakAuthResponders().
				AddStringResponderWithCode(200, "/admin/realms/"+testRealmName+"/groups/"+testGroupID+"/role-mappings", "{}").
				AddStringResponderWithCode(400, "/admin/realms/"+testRealmName+"/clients", "{}").
				BuildAndStart()
			t.Cleanup(server.Close)

			kClient, err := keycloakapi.NewKeycloakClient(
				context.Background(), server.GetURL(), "admin-cli", keycloakapi.WithClientSecret("secret"),
			)
			require.NoError(t, err)

			group := &keycloakApi.KeycloakRealmGroup{}
			group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{{ClientID: "   ", Roles: tt.roles}})

			groupCtx := &GroupContext{RealmName: testRealmName, GroupID: testGroupID}

			err = NewSyncClientRoles().Serve(ctrl.LoggerInto(context.Background(), logr.Discard()), group, kClient, groupCtx)

			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.EqualError(t, err, tt.wantErr)
			require.ErrorIs(t, err, keycloakapi.ErrNotFound)
		})
	}
}

func TestSyncClientRoles_Serve_ErrorGettingClientRoleMappings(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{"role1"})},
	})

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{}, nil, nil)

	// Resolve client UUID (since not in mappings)
	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "test-client",
	).Return("client-uuid", nil)

	// This test now triggers an error getting the client role (no longer gets client role mappings)
	mockClients.EXPECT().GetClientRole(
		context.Background(), testRealmName, "client-uuid", "role1",
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get client role")
}

func TestSyncClientRoles_Serve_ErrorGettingClientRole(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{"nonexistent-role"})},
	})

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{}, nil, nil)

	// Resolve client UUID (since not in mappings)
	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "test-client",
	).Return("client-uuid", nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), testRealmName, "client-uuid", "nonexistent-role",
	).Return(nil, nil, errors.New("role not found"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get client role")
}

func TestSyncClientRoles_Serve_ClientRoleMissing(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{"role1"})},
	})

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{}, nil, nil)

	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "test-client",
	).Return("client-uuid", nil)

	// No role, no error.
	mockClients.EXPECT().GetClientRole(
		context.Background(), testRealmName, "client-uuid", "role1",
	).Return(nil, nil, nil)

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, `client role "role1" not found for client "test-client"`)
}

func TestSyncClientRoles_Serve_ClientRoleWithoutID(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{"role1"})},
	})

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{}, nil, nil)

	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "test-client",
	).Return("client-uuid", nil)

	// No AddClientRoleMappings expectation: reaching it fails the test.
	mockClients.EXPECT().GetClientRole(
		context.Background(), testRealmName, "client-uuid", "role1",
	).Return(&keycloakapi.RoleRepresentation{Name: ptr.To("role1")}, nil, nil)

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, `client role "role1" has no id`)
}

func TestSyncClientRoles_Serve_ErrorAddingClientRoleMappings(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{"role1"})},
	})

	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{}, nil, nil)

	// Resolve client UUID (since not in mappings)
	mockClients.EXPECT().GetClientUUID(
		context.Background(), testRealmName, "test-client",
	).Return("client-uuid", nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), testRealmName, "client-uuid", "role1",
	).Return(&keycloakapi.RoleRepresentation{
		Id: ptr.To("role1-id"), Name: ptr.To("role1"),
	}, nil, nil)

	mockGroups.EXPECT().AddClientRoleMappings(
		context.Background(), testRealmName, testGroupID, "client-uuid",
		[]keycloakapi.RoleRepresentation{
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

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "test-client", Roles: ptr.To([]string{})}, // Remove all
	})

	// Role mappings with existing client roles
	clientMappings := map[string]keycloakapi.ClientMappingsRepresentation{
		"test-client": {
			Id:     ptr.To("client-uuid"),
			Client: ptr.To("test-client"),
			Mappings: &[]keycloakapi.RoleRepresentation{
				{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
			},
		},
	}
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{
		ClientMappings: &clientMappings,
	}, nil, nil)

	mockGroups.EXPECT().DeleteClientRoleMappings(
		context.Background(), testRealmName, testGroupID, "client-uuid",
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, errors.New("delete failed"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to delete client role mappings")
}

func TestSyncClientRoles_Serve_UnmanagedOmittedClientRolesLeftAlone(t *testing.T) {
	// No expectations: any Keycloak call fails this test.
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncClientRoles_Serve_ManagedListSweepsUnmentionedClient(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "app", Roles: ptr.To([]string{"r1"})},
	})

	clientMappings := map[string]keycloakapi.ClientMappingsRepresentation{
		"app": {
			Id:     ptr.To("app-uuid"),
			Client: ptr.To("app"),
			Mappings: &[]keycloakapi.RoleRepresentation{
				{Id: ptr.To("r1-id"), Name: ptr.To("r1")},
			},
		},
		"other": {
			Id:     ptr.To("other-uuid"),
			Client: ptr.To("other"),
			Mappings: &[]keycloakapi.RoleRepresentation{
				{Id: ptr.To("o1-id"), Name: ptr.To("o1")},
			},
		},
	}
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{
		ClientMappings: &clientMappings,
	}, nil, nil)

	mockGroups.EXPECT().DeleteClientRoleMappings(
		context.Background(), testRealmName, testGroupID, "other-uuid",
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("o1-id"), Name: ptr.To("o1")},
		},
	).Return(nil, nil).Once()

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncClientRoles_Serve_ErrorRemovingUnclaimedClient(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockClients := mocks.NewMockClientsClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups:  mockGroups,
		Clients: mockClients,
	}

	groupCtx := &GroupContext{
		RealmName: testRealmName,
		GroupID:   testGroupID,
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.ClientRoles = ptr.To([]keycloakApi.UserClientRole{
		{ClientID: "app"},
	})

	clientMappings := map[string]keycloakapi.ClientMappingsRepresentation{
		"other": {
			Id:     ptr.To("other-uuid"),
			Client: ptr.To("other"),
			Mappings: &[]keycloakapi.RoleRepresentation{
				{Id: ptr.To("o1-id"), Name: ptr.To("o1")},
			},
		},
	}
	mockGroups.EXPECT().GetRoleMappings(
		context.Background(), testRealmName, testGroupID,
	).Return(&keycloakapi.MappingsRepresentation{
		ClientMappings: &clientMappings,
	}, nil, nil)

	mockGroups.EXPECT().DeleteClientRoleMappings(
		context.Background(), testRealmName, testGroupID, "other-uuid",
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("o1-id"), Name: ptr.To("o1")},
		},
	).Return(nil, errors.New("delete failed"))

	h := NewSyncClientRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, `unable to remove unclaimed client roles for "other"`)
}
