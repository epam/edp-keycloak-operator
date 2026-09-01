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

func TestSyncRealmRoles_Serve_UnmanagedOmittedRolesLeftAlone(t *testing.T) {
	// No expectations: any Keycloak call fails this test.
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncRealmRoles_Serve_AddRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{"role1", "role2"})

	// No existing roles
	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	// Fetch role1
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "role1",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("role1-id"),
		Name: ptr.To("role1"),
	}, nil, nil)

	// Fetch role2
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "role2",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("role2-id"),
		Name: ptr.To("role2"),
	}, nil, nil)

	// Add both roles
	mockGroups.EXPECT().AddRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakapi.RoleRepresentation{
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

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{}) // Empty - remove all

	// Current roles
	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{
		{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
	}, nil, nil)

	// Remove old role
	mockGroups.EXPECT().DeleteRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, nil)

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncRealmRoles_Serve_EmptyListWithNoCurrentRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{}) // Managed, nothing to keep

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	// No add or delete expected: an empty list over an empty mapping set writes nothing.

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncRealmRoles_Serve_AddAndRemove(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{"role1", "role2"})

	// Current roles: role1 (keep), old-role (remove)
	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{
		{Id: ptr.To("role1-id"), Name: ptr.To("role1")},
		{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
	}, nil, nil)

	// Fetch role2 (role1 already exists)
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "role2",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("role2-id"),
		Name: ptr.To("role2"),
	}, nil, nil)

	// Add role2
	mockGroups.EXPECT().AddRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("role2-id"), Name: ptr.To("role2")},
		},
	).Return(nil, nil)

	// Remove old-role
	mockGroups.EXPECT().DeleteRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakapi.RoleRepresentation{
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

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{"role1"})

	// Current roles match spec
	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{
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

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{"role1"})

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

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{"nonexistent-role"})

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "nonexistent-role",
	).Return(nil, nil, errors.New("role not found"))

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get realm role")
}

func TestSyncRealmRoles_Serve_ErrorRoleNotFound(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{"ghost-role"})

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	// No role, no error.
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "ghost-role",
	).Return(nil, nil, nil)

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, `realm role "ghost-role" not found in realm "test-realm"`)
}

func TestSyncRealmRoles_Serve_RoleWithoutID(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{"idless-role"})

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	// Keycloak answers a name/id mismatch with 404, so no AddRealmRoleMappings may be issued.
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "idless-role",
	).Return(&keycloakapi.RoleRepresentation{Name: ptr.To("idless-role")}, nil, nil)

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, `realm role "idless-role" has no id`)
}

func TestSyncRealmRoles_Serve_ErrorAddingRoles(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)
	mockRoles := mocks.NewMockRolesClient(t)

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{"role1"})

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{}, nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", "role1",
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("role1-id"),
		Name: ptr.To("role1"),
	}, nil, nil)

	mockGroups.EXPECT().AddRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakapi.RoleRepresentation{
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

	kClient := &keycloakapi.KeycloakClient{
		Groups: mockGroups,
		Roles:  mockRoles,
	}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.RealmRoles = ptr.To([]string{}) // Remove all

	mockGroups.EXPECT().GetRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
	).Return([]keycloakapi.RoleRepresentation{
		{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
	}, nil, nil)

	mockGroups.EXPECT().DeleteRealmRoleMappings(
		context.Background(), "test-realm", "group-123",
		[]keycloakapi.RoleRepresentation{
			{Id: ptr.To("old-role-id"), Name: ptr.To("old-role")},
		},
	).Return(nil, errors.New("delete failed"))

	h := NewSyncRealmRoles()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to delete realm role mappings")
}
