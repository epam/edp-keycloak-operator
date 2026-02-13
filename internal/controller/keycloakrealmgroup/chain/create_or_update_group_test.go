package chain

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

const (
	testGroupName      = "test-group"
	testChildGroupName = "child-group"
	testUpdatedPath    = "/updated-path"
)

func TestCreateOrUpdateGroup_Serve_CreateTopLevel(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = "/test-group"
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testGroupName,
	).Return(nil, nil, nil)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
		keycloakv2.GroupRepresentation{
			Name:       ptr.To(testGroupName),
			Path:       ptr.To("/test-group"),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(&keycloakv2.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{"Location": []string{"http://localhost/admin/realms/test-realm/groups/group-id-123"}},
		},
	}, nil)

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "group-id-123", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_CreateChildGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm", ParentGroupID: "parent-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testChildGroupName
	group.Spec.Path = "/child-group"
	group.Spec.Attributes = map[string][]string{"a": {"b"}}

	mockGroups.EXPECT().FindChildGroupByName(
		context.Background(), "test-realm", "parent-id", testChildGroupName,
	).Return(nil, nil, nil)

	mockGroups.EXPECT().CreateChildGroup(
		context.Background(), "test-realm", "parent-id",
		keycloakv2.GroupRepresentation{
			Name:       ptr.To(testChildGroupName),
			Path:       ptr.To("/child-group"),
			Attributes: &map[string][]string{"a": {"b"}},
		},
	).Return(&keycloakv2.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{"Location": []string{"http://localhost/admin/realms/test-realm/groups/child-id-456"}},
		},
	}, nil)

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "child-id-456", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_UpdateExisting(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = "existing-group"
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"new-key": {"new-val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", "existing-group",
	).Return(&keycloakv2.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To("existing-group"),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "existing-id",
		keycloakv2.GroupRepresentation{
			Id:         ptr.To("existing-id"),
			Name:       ptr.To("existing-group"),
			Path:       ptr.To(testUpdatedPath),
			Attributes: &map[string][]string{"new-key": {"new-val"}},
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "existing-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_FindGroupError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testGroupName,
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to search for group")
}

func TestCreateOrUpdateGroup_Serve_CreateGroupError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = "/test-group"
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testGroupName,
	).Return(nil, nil, nil)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
		keycloakv2.GroupRepresentation{
			Name:       ptr.To(testGroupName),
			Path:       ptr.To("/test-group"),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(nil, errors.New("create failed"))

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to create group")
}

func TestCreateOrUpdateGroup_Serve_UpdateGroupError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = "existing-group"
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", "existing-group",
	).Return(&keycloakv2.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To("existing-group"),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "existing-id",
		keycloakv2.GroupRepresentation{
			Id:         ptr.To("existing-id"),
			Name:       ptr.To("existing-group"),
			Path:       ptr.To(testUpdatedPath),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(nil, errors.New("update failed"))

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to update group")
}

func TestCreateOrUpdateGroup_Serve_UpdateExistingChildGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm", ParentGroupID: "parent-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testChildGroupName
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"k": {"v"}}

	mockGroups.EXPECT().FindChildGroupByName(
		context.Background(), "test-realm", "parent-id", testChildGroupName,
	).Return(&keycloakv2.GroupRepresentation{
		Id:   ptr.To("child-id"),
		Name: ptr.To(testChildGroupName),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "child-id",
		keycloakv2.GroupRepresentation{
			Id:         ptr.To("child-id"),
			Name:       ptr.To(testChildGroupName),
			Path:       ptr.To(testUpdatedPath),
			Attributes: &map[string][]string{"k": {"v"}},
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "child-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_FindChildGroupError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm", ParentGroupID: "parent-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testChildGroupName

	mockGroups.EXPECT().FindChildGroupByName(
		context.Background(), "test-realm", "parent-id", testChildGroupName,
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to search for group")
}
