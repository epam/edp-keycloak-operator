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

func TestSyncSubGroups_Serve_EmptySubGroups(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.SubGroups = []string{} // Empty - should skip

	// No calls expected

	h := NewSyncSubGroups()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncSubGroups_Serve_AddSubGroups(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "parent-group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.SubGroups = []string{"subgroup1", "subgroup2"}

	// Get current children (empty)
	mockGroups.EXPECT().GetChildGroups(
		context.Background(), "test-realm", "parent-group-123", (*keycloakv2.GetChildGroupsParams)(nil),
	).Return([]keycloakv2.GroupRepresentation{}, nil, nil)

	// Find subgroup1
	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", "subgroup1",
	).Return(&keycloakv2.GroupRepresentation{
		Id: ptr.To("subgroup1-id"), Name: ptr.To("subgroup1"),
	}, nil, nil)

	// Add subgroup1 as child
	mockGroups.EXPECT().CreateChildGroup(
		context.Background(), "test-realm", "parent-group-123",
		keycloakv2.GroupRepresentation{Id: ptr.To("subgroup1-id"), Name: ptr.To("subgroup1")},
	).Return(nil, nil)

	// Find subgroup2
	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", "subgroup2",
	).Return(&keycloakv2.GroupRepresentation{
		Id: ptr.To("subgroup2-id"), Name: ptr.To("subgroup2"),
	}, nil, nil)

	// Add subgroup2 as child
	mockGroups.EXPECT().CreateChildGroup(
		context.Background(), "test-realm", "parent-group-123",
		keycloakv2.GroupRepresentation{Id: ptr.To("subgroup2-id"), Name: ptr.To("subgroup2")},
	).Return(nil, nil)

	h := NewSyncSubGroups()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncSubGroups_Serve_DetachSubGroups(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "parent-group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.SubGroups = []string{"keep-child"} // Only keeping one, detach the other

	// Get current children: keep-child (keep), old-child (detach)
	mockGroups.EXPECT().GetChildGroups(
		context.Background(), "test-realm", "parent-group-123", (*keycloakv2.GetChildGroupsParams)(nil),
	).Return([]keycloakv2.GroupRepresentation{
		{Id: ptr.To("keep-child-id"), Name: ptr.To("keep-child")},
		{Id: ptr.To("old-child-id"), Name: ptr.To("old-child")},
	}, nil, nil)

	// Detach old-child (create at top level)
	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
		keycloakv2.GroupRepresentation{Id: ptr.To("old-child-id"), Name: ptr.To("old-child")},
	).Return(nil, nil)

	h := NewSyncSubGroups()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncSubGroups_Serve_AddAndDetach(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "parent-group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.SubGroups = []string{"keep-child", "new-child"}

	// Get current children: keep-child (keep), old-child (detach)
	mockGroups.EXPECT().GetChildGroups(
		context.Background(), "test-realm", "parent-group-123", (*keycloakv2.GetChildGroupsParams)(nil),
	).Return([]keycloakv2.GroupRepresentation{
		{Id: ptr.To("keep-child-id"), Name: ptr.To("keep-child")},
		{Id: ptr.To("old-child-id"), Name: ptr.To("old-child")},
	}, nil, nil)

	// Find new-child (keep-child already exists)
	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", "new-child",
	).Return(&keycloakv2.GroupRepresentation{
		Id: ptr.To("new-child-id"), Name: ptr.To("new-child"),
	}, nil, nil)

	// Add new-child as child
	mockGroups.EXPECT().CreateChildGroup(
		context.Background(), "test-realm", "parent-group-123",
		keycloakv2.GroupRepresentation{Id: ptr.To("new-child-id"), Name: ptr.To("new-child")},
	).Return(nil, nil)

	// Detach old-child (create at top level)
	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
		keycloakv2.GroupRepresentation{Id: ptr.To("old-child-id"), Name: ptr.To("old-child")},
	).Return(nil, nil)

	h := NewSyncSubGroups()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
}

func TestSyncSubGroups_Serve_ErrorGettingChildGroups(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "parent-group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.SubGroups = []string{"subgroup1"}

	mockGroups.EXPECT().GetChildGroups(
		context.Background(), "test-realm", "parent-group-123", (*keycloakv2.GetChildGroupsParams)(nil),
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncSubGroups()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get child groups")
}

func TestSyncSubGroups_Serve_SubGroupNotFound(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "parent-group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.SubGroups = []string{"nonexistent"}

	mockGroups.EXPECT().GetChildGroups(
		context.Background(), "test-realm", "parent-group-123", (*keycloakv2.GetChildGroupsParams)(nil),
	).Return([]keycloakv2.GroupRepresentation{}, nil, nil)

	// FindGroupByName returns nil (not found)
	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", "nonexistent",
	).Return(nil, nil, nil)

	h := NewSyncSubGroups()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "subgroup \"nonexistent\" not found")
}

func TestSyncSubGroups_Serve_ErrorFindingSubGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "parent-group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.SubGroups = []string{"subgroup1"}

	mockGroups.EXPECT().GetChildGroups(
		context.Background(), "test-realm", "parent-group-123", (*keycloakv2.GetChildGroupsParams)(nil),
	).Return([]keycloakv2.GroupRepresentation{}, nil, nil)

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", "subgroup1",
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncSubGroups()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to find subgroup")
}

func TestSyncSubGroups_Serve_ErrorCreatingChildGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "parent-group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.SubGroups = []string{"subgroup1"}

	mockGroups.EXPECT().GetChildGroups(
		context.Background(), "test-realm", "parent-group-123", (*keycloakv2.GetChildGroupsParams)(nil),
	).Return([]keycloakv2.GroupRepresentation{}, nil, nil)

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", "subgroup1",
	).Return(&keycloakv2.GroupRepresentation{
		Id: ptr.To("subgroup1-id"), Name: ptr.To("subgroup1"),
	}, nil, nil)

	mockGroups.EXPECT().CreateChildGroup(
		context.Background(), "test-realm", "parent-group-123",
		keycloakv2.GroupRepresentation{Id: ptr.To("subgroup1-id"), Name: ptr.To("subgroup1")},
	).Return(nil, errors.New("create failed"))

	h := NewSyncSubGroups()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to add subgroup")
}

func TestSyncSubGroups_Serve_ErrorDetachingSubGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}

	groupCtx := &GroupContext{
		RealmName: "test-realm",
		GroupID:   "parent-group-123",
	}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.SubGroups = []string{"keep-child"} // Detach others

	mockGroups.EXPECT().GetChildGroups(
		context.Background(), "test-realm", "parent-group-123", (*keycloakv2.GetChildGroupsParams)(nil),
	).Return([]keycloakv2.GroupRepresentation{
		{Id: ptr.To("keep-child-id"), Name: ptr.To("keep-child")},
		{Id: ptr.To("child1-id"), Name: ptr.To("child1")},
	}, nil, nil)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
		keycloakv2.GroupRepresentation{Id: ptr.To("child1-id"), Name: ptr.To("child1")},
	).Return(nil, errors.New("detach failed"))

	h := NewSyncSubGroups()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to detach subgroup")
}
