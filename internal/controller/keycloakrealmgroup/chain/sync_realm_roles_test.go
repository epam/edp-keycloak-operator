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

func TestSyncRealmRoles_Serve_AddRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = []string{"role1", "role2"}

	// No existing roles
	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakv2.RoleRepresentation{}, nil, nil)

	// Fetch role1
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "role1",
	).Return(&keycloakv2.RoleRepresentation{
		Id:   ptr.To("role1-id"),
		Name: ptr.To("role1"),
	}, nil, nil)

	// Fetch role2
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "role2",
	).Return(&keycloakv2.RoleRepresentation{
		Id:   ptr.To("role2-id"),
		Name: ptr.To("role2"),
	}, nil, nil)

	// Add both roles
	mockGroups.EXPECT().AddRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
			{Id: ptr.To("role2-id"), Name: ptr.To("role2")},
		},
	).Return(nil, nil)

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncRealmRoles_Serve_RemoveRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = []string{} // Empty - remove all

	// Current roles
	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakv2.RoleRepresentation{
		{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
	}, nil, nil)

	// Remove old role
	mockGroups.EXPECT().DeleteRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, nil)

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncRealmRoles_Serve_AddAndRemove(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = []string{"role1", "role2"}

	// Current roles: role1 (keep), old-role (remove)
	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakv2.RoleRepresentation{
		{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
		{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
	}, nil, nil)

	// Fetch role2 (role1 already exists)
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "role2",
	).Return(&keycloakv2.RoleRepresentation{
		Id:   ptr.To("role2-id"),
		Name: ptr.To("role2"),
	}, nil, nil)

	// Add role2
	mockGroups.EXPECT().AddRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("role2-id"), Name: ptr.To("role2")},
		},
	).Return(nil, nil)

	// Remove old-role
	mockGroups.EXPECT().DeleteRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, nil)

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncRealmRoles_Serve_NoChanges(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = []string{"role1"}

	// Current roles match spec
	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakv2.RoleRepresentation{
		{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
	}, nil, nil)

	// No Add or Delete calls expected

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncRealmRoles_Serve_ErrorGettingRoleMappings(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = []string{"role1"}

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get realm role mappings")
}

func TestSyncRealmRoles_Serve_ErrorGettingRole(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = []string{"nonexistent-role"}

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakv2.RoleRepresentation{}, nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "nonexistent-role",
	).Return(nil, nil, errors.New("role not found"))

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get realm role")
}

func TestSyncRealmRoles_Serve_ErrorAddingRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = []string{"role1"}

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakv2.RoleRepresentation{}, nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "role1",
	).Return(&keycloakv2.RoleRepresentation{
		Id:   ptr.To("role1-id"),
		Name: ptr.To("role1"),
	}, nil, nil)

	mockGroups.EXPECT().AddRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
		},
	).Return(nil, errors.New("add failed"))

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to add realm role mappings")
}

func TestSyncRealmRoles_Serve_ErrorDeletingRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakv2.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = []string{} // Remove all

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakv2.RoleRepresentation{
		{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
	}, nil, nil)

	mockGroups.EXPECT().DeleteRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakv2.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, errors.New("delete failed"))

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to delete realm role mappings")
}
