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
	testGroupName   = "test-group"
	testUpdatedPath = "/updated-path"
)

func TestCreateOrUpdateGroup_Serve_CreateTopLevel(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = "/test-group"
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().GetGroups(
		context.Background(), "test-realm",
		&keycloakv2.GetGroupsParams{Search: ptr.To(testGroupName)},
	).Return([]keycloakv2.GroupRepresentation{}, nil, nil)

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
	group.Spec.Name = "child-group"
	group.Spec.Path = "/child-group"
	group.Spec.Attributes = map[string][]string{"a": {"b"}}

	mockGroups.EXPECT().GetGroups(
		context.Background(), "test-realm",
		&keycloakv2.GetGroupsParams{Search: ptr.To("child-group")},
	).Return(nil, nil, nil)

	mockGroups.EXPECT().CreateChildGroup(
		context.Background(), "test-realm", "parent-id",
		keycloakv2.GroupRepresentation{
			Name:       ptr.To("child-group"),
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

	mockGroups.EXPECT().GetGroups(
		context.Background(), "test-realm",
		&keycloakv2.GetGroupsParams{Search: ptr.To("existing-group")},
	).Return([]keycloakv2.GroupRepresentation{
		{
			Id:   ptr.To("existing-id"),
			Name: ptr.To("existing-group"),
			Path: ptr.To("/old-path"),
		},
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

func TestCreateOrUpdateGroup_Serve_GetGroupsError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName

	mockGroups.EXPECT().GetGroups(
		context.Background(), "test-realm",
		&keycloakv2.GetGroupsParams{Search: ptr.To(testGroupName)},
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

	mockGroups.EXPECT().GetGroups(
		context.Background(), "test-realm",
		&keycloakv2.GetGroupsParams{Search: ptr.To(testGroupName)},
	).Return([]keycloakv2.GroupRepresentation{}, nil, nil)

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

	mockGroups.EXPECT().GetGroups(
		context.Background(), "test-realm",
		&keycloakv2.GetGroupsParams{Search: ptr.To("existing-group")},
	).Return([]keycloakv2.GroupRepresentation{
		{
			Id:   ptr.To("existing-id"),
			Name: ptr.To("existing-group"),
			Path: ptr.To("/old-path"),
		},
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

func TestCreateOrUpdateGroup_Serve_FindGroupInSubGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = "nested-group"
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"k": {"v"}}

	mockGroups.EXPECT().GetGroups(
		context.Background(), "test-realm",
		&keycloakv2.GetGroupsParams{Search: ptr.To("nested-group")},
	).Return([]keycloakv2.GroupRepresentation{
		{
			Id:   ptr.To("parent-id"),
			Name: ptr.To("parent-group"),
			SubGroups: &[]keycloakv2.GroupRepresentation{
				{
					Id:   ptr.To("nested-id"),
					Name: ptr.To("nested-group"),
					Path: ptr.To("/old-path"),
				},
			},
		},
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "nested-id",
		keycloakv2.GroupRepresentation{
			Id:         ptr.To("nested-id"),
			Name:       ptr.To("nested-group"),
			Path:       ptr.To(testUpdatedPath),
			Attributes: &map[string][]string{"k": {"v"}},
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "nested-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_SearchReturnsNonMatchingGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakv2.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = "my-group"
	group.Spec.Path = "/my-group"
	group.Spec.Attributes = map[string][]string{"k": {"v"}}

	// Keycloak search is substring-based, so it may return non-exact matches
	mockGroups.EXPECT().GetGroups(
		context.Background(), "test-realm",
		&keycloakv2.GetGroupsParams{Search: ptr.To("my-group")},
	).Return([]keycloakv2.GroupRepresentation{
		{
			Id:   ptr.To("other-id"),
			Name: ptr.To("my-group-extended"),
		},
	}, nil, nil)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
		keycloakv2.GroupRepresentation{
			Name:       ptr.To("my-group"),
			Path:       ptr.To("/my-group"),
			Attributes: &map[string][]string{"k": {"v"}},
		},
	).Return(&keycloakv2.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{"Location": []string{"http://localhost/admin/realms/test-realm/groups/new-id"}},
		},
	}, nil)

	h := NewCreateOrUpdateGroup()
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "new-id", groupCtx.GroupID)
}

func TestFindGroupByNameRecursive(t *testing.T) {
	tests := []struct {
		name      string
		group     keycloakv2.GroupRepresentation
		search    string
		wantFound bool
		wantID    string
	}{
		{
			name: "direct match",
			group: keycloakv2.GroupRepresentation{
				Id:   ptr.To("id-1"),
				Name: ptr.To("target"),
			},
			search:    "target",
			wantFound: true,
			wantID:    "id-1",
		},
		{
			name: "no match",
			group: keycloakv2.GroupRepresentation{
				Id:   ptr.To("id-1"),
				Name: ptr.To("other"),
			},
			search:    "target",
			wantFound: false,
		},
		{
			name: "nil name",
			group: keycloakv2.GroupRepresentation{
				Id: ptr.To("id-1"),
			},
			search:    "target",
			wantFound: false,
		},
		{
			name: "found in subgroup",
			group: keycloakv2.GroupRepresentation{
				Id:   ptr.To("parent-id"),
				Name: ptr.To("parent"),
				SubGroups: &[]keycloakv2.GroupRepresentation{
					{
						Id:   ptr.To("child-id"),
						Name: ptr.To("target"),
					},
				},
			},
			search:    "target",
			wantFound: true,
			wantID:    "child-id",
		},
		{
			name: "found in deeply nested subgroup",
			group: keycloakv2.GroupRepresentation{
				Id:   ptr.To("level-0"),
				Name: ptr.To("level0"),
				SubGroups: &[]keycloakv2.GroupRepresentation{
					{
						Id:   ptr.To("level-1"),
						Name: ptr.To("level1"),
						SubGroups: &[]keycloakv2.GroupRepresentation{
							{
								Id:   ptr.To("level-2"),
								Name: ptr.To("target"),
							},
						},
					},
				},
			},
			search:    "target",
			wantFound: true,
			wantID:    "level-2",
		},
		{
			name: "not found in subgroups",
			group: keycloakv2.GroupRepresentation{
				Id:   ptr.To("parent-id"),
				Name: ptr.To("parent"),
				SubGroups: &[]keycloakv2.GroupRepresentation{
					{
						Id:   ptr.To("child-id"),
						Name: ptr.To("other-child"),
					},
				},
			},
			search:    "target",
			wantFound: false,
		},
		{
			name: "empty subgroups",
			group: keycloakv2.GroupRepresentation{
				Id:        ptr.To("parent-id"),
				Name:      ptr.To("parent"),
				SubGroups: &[]keycloakv2.GroupRepresentation{},
			},
			search:    "target",
			wantFound: false,
		},
		{
			name: "prefers parent over child with same name",
			group: keycloakv2.GroupRepresentation{
				Id:   ptr.To("parent-id"),
				Name: ptr.To("target"),
				SubGroups: &[]keycloakv2.GroupRepresentation{
					{
						Id:   ptr.To("child-id"),
						Name: ptr.To("target"),
					},
				},
			},
			search:    "target",
			wantFound: true,
			wantID:    "parent-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findGroupByNameRecursive(tt.group, tt.search)

			if !tt.wantFound {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			assert.Equal(t, tt.wantID, *result.Id)
		})
	}
}
