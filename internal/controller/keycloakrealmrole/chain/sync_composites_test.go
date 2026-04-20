package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const (
	testParentRoleName = "parent-role"
	testClientName     = "my-client"
)

func TestSyncComposites_Serve_NotComposite(t *testing.T) {
	kClient := &keycloakapi.KeycloakClient{}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Composite = false

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.NoError(t, err)
}

func TestSyncComposites_Serve_AddRealmComposites(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	role.Spec.Composites = []keycloakApi.Composite{
		{Name: "child-role-1"},
		{Name: "child-role-2"},
	}

	// No existing composites
	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "child-role-1",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("child-1-id"),
		Name: ptr.To("child-role-1"),
	}, nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "child-role-2",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("child-2-id"),
		Name: ptr.To("child-role-2"),
	}, nil, nil)

	mockRoles.EXPECT().AddRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("child-1-id"), Name: ptr.To("child-role-1")},
			{Id: ptr.To("child-2-id"), Name: ptr.To("child-role-2")},
		},
	).Return(nil, nil)

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.NoError(t, err)
}

func TestSyncComposites_Serve_RemoveStaleComposites(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	// No desired composites — all existing should be removed

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{
		{Id: ptr.To("stale-id"), Name: ptr.To("stale-role")},
	}, nil, nil)

	mockRoles.EXPECT().DeleteRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("stale-id"), Name: ptr.To("stale-role")},
		},
	).Return(nil, nil)

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.NoError(t, err)
}

func TestSyncComposites_Serve_AddClientComposites(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	mockClients := mocks.NewMockClientsClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles, Clients: mockClients}

	clientName := testClientName

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	role.Spec.CompositesClientRoles = map[string][]keycloakApi.Composite{
		clientName: {{Name: "client-role-1"}},
	}

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakapi.GetClientsParams{ClientId: &clientName},
	).Return([]keycloakapi.ClientRepresentation{
		{Id: ptr.To("client-uuid-1")},
	}, nil, nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), "test-realm", "client-uuid-1", "client-role-1",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("client-role-1-id"),
		Name: ptr.To("client-role-1"),
	}, nil, nil)

	mockRoles.EXPECT().AddRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("client-role-1-id"), Name: ptr.To("client-role-1")},
		},
	).Return(nil, nil)

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.NoError(t, err)
}

func TestSyncComposites_Serve_GetCompositesError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current composites")
}

func TestSyncComposites_Serve_ClientNotFound(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	mockClients := mocks.NewMockClientsClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles, Clients: mockClients}

	clientName := "missing-client"

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	role.Spec.CompositesClientRoles = map[string][]keycloakApi.Composite{
		clientName: {{Name: "role1"}},
	}

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakapi.GetClientsParams{ClientId: &clientName},
	).Return([]keycloakapi.ClientRepresentation{}, nil, nil)

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client missing-client not found")
}

func TestSyncComposites_Serve_GetRealmRoleError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	role.Spec.Composites = []keycloakApi.Composite{
		{Name: "child-role"},
	}

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "child-role",
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get realm role")
}

func TestSyncComposites_Serve_GetClientRoleError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	mockClients := mocks.NewMockClientsClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles, Clients: mockClients}

	clientName := testClientName

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	role.Spec.CompositesClientRoles = map[string][]keycloakApi.Composite{
		clientName: {{Name: "role1"}},
	}

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakapi.GetClientsParams{ClientId: &clientName},
	).Return([]keycloakapi.ClientRepresentation{
		{Id: ptr.To("client-uuid-1")},
	}, nil, nil)

	mockClients.EXPECT().GetClientRole(
		context.Background(), "test-realm", "client-uuid-1", "role1",
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get client role")
}

func TestSyncComposites_Serve_NoChanges(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	role.Spec.Composites = []keycloakApi.Composite{
		{Name: "existing-role"},
	}

	// Current composites match desired — nothing to add or remove.
	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{
		{Id: ptr.To("existing-id"), Name: ptr.To("existing-role")},
	}, nil, nil)

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.NoError(t, err)
}

func TestSyncComposites_Serve_AddRealmCompositesError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	role.Spec.Composites = []keycloakApi.Composite{
		{Name: "new-role"},
	}

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "new-role",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("new-role-id"),
		Name: ptr.To("new-role"),
	}, nil, nil)

	mockRoles.EXPECT().AddRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("new-role-id"), Name: ptr.To("new-role")},
		},
	).Return(nil, errors.New("api error"))

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add composite roles")
}

func TestSyncComposites_Serve_DeleteStaleError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	// No desired composites — all existing are stale.

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{
		{Id: ptr.To("stale-id"), Name: ptr.To("stale-role")},
	}, nil, nil)

	mockRoles.EXPECT().DeleteRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("stale-id"), Name: ptr.To("stale-role")},
		},
	).Return(nil, errors.New("api error"))

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete stale composite roles")
}

func TestSyncComposites_Serve_GetClientsError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	mockClients := mocks.NewMockClientsClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles, Clients: mockClients}

	clientName := testClientName

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	role.Spec.CompositesClientRoles = map[string][]keycloakApi.Composite{
		clientName: {{Name: "role1"}},
	}

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakapi.GetClientsParams{ClientId: &clientName},
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get client")
}

func TestSyncComposites_Serve_ClientCompositeAlreadyPresent(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	mockClients := mocks.NewMockClientsClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles, Clients: mockClients}

	clientName := testClientName
	clientUUID := "client-uuid-1"

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	role.Spec.CompositesClientRoles = map[string][]keycloakApi.Composite{
		clientName: {{Name: "existing-client-role"}},
	}

	isClientRole := true

	// Current composites already contain the client role — key = clientUUID-roleName.
	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{
		{
			Id:          ptr.To("existing-client-role-id"),
			Name:        ptr.To("existing-client-role"),
			ClientRole:  &isClientRole,
			ContainerId: &clientUUID,
		},
	}, nil, nil)

	mockClients.EXPECT().GetClients(
		context.Background(), "test-realm",
		&keycloakapi.GetClientsParams{ClientId: &clientName},
	).Return([]keycloakapi.ClientRepresentation{
		{Id: ptr.To(clientUUID)},
	}, nil, nil)

	// GetClientRole and AddRealmRoleComposites must NOT be called.
	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.NoError(t, err)
}

func TestSyncComposites_Serve_Mixed(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testParentRoleName
	role.Spec.Composite = true
	// Desired: keep "kept-role", add "new-role"; "stale-role" should be removed.
	role.Spec.Composites = []keycloakApi.Composite{
		{Name: "kept-role"},
		{Name: "new-role"},
	}

	mockRoles.EXPECT().GetRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
	).Return([]keycloakapi.RoleRepresentation{
		{Id: ptr.To("kept-id"), Name: ptr.To("kept-role")},
		{Id: ptr.To("stale-id"), Name: ptr.To("stale-role")},
	}, nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "new-role",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("new-id"),
		Name: ptr.To("new-role"),
	}, nil, nil)

	mockRoles.EXPECT().AddRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("new-id"), Name: ptr.To("new-role")},
		},
	).Return(nil, nil)

	mockRoles.EXPECT().DeleteRealmRoleComposites(
		context.Background(), "test-realm", testParentRoleName,
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("stale-id"), Name: ptr.To("stale-role")},
		},
	).Return(nil, nil)

	h := NewSyncComposites(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.NoError(t, err)
}
