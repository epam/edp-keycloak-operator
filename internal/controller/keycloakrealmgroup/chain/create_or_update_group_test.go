package chain

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const (
	testGroupName      = "test-group"
	testGroupPath      = "/test-group"
	testChildGroupName = "child-group"
	testUpdatedPath    = "/updated-path"
	testNamespace      = "ns1"
	testCRNameA        = "cr-a"
	testExistingGroup  = "existing-group"
	testCRNameB        = "cr-b"
	testRealmRefName   = "test"
)

// newFakeK8sClient builds a fake controller-runtime client with the keycloak
// schemes registered, pre-populated with the given objects.
func newFakeK8sClient(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, keycloakApi.AddToScheme(scheme))
	require.NoError(t, keycloakAlpha.AddToScheme(scheme))

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func namespacedKeycloakStack(ns, realmName, url string) []client.Object {
	return []client.Object{
		&keycloakApi.Keycloak{
			ObjectMeta: metav1.ObjectMeta{Name: "keycloak", Namespace: ns},
			Spec:       keycloakApi.KeycloakSpec{Url: url},
		},
		&keycloakApi.KeycloakRealm{
			ObjectMeta: metav1.ObjectMeta{Name: testRealmRefName, Namespace: ns},
			Spec: keycloakApi.KeycloakRealmSpec{
				RealmName: realmName,
				KeycloakRef: common.KeycloakRef{
					Kind: keycloakApi.KeycloakKind,
					Name: "keycloak",
				},
			},
		},
	}
}

func TestCreateOrUpdateGroup_Serve_CreateTopLevel(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = testGroupPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), testRealmName,
		keycloakapi.GroupRepresentation{
			Name:       ptr.To(testGroupName),
			Path:       ptr.To(testGroupPath),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(&keycloakapi.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{"Location": []string{"http://localhost/admin/realms/test-realm/groups/group-id-123"}},
		},
	}, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "group-id-123", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_CreateChildGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName, ParentGroupID: "parent-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testChildGroupName
	group.Spec.Description = "Child group description"
	group.Spec.Path = "/child-group"
	group.Spec.Attributes = map[string][]string{"a": {"b"}}

	mockGroups.EXPECT().FindChildGroupByName(
		context.Background(), testRealmName, "parent-id", testChildGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().CreateChildGroup(
		context.Background(), testRealmName, "parent-id",
		keycloakapi.GroupRepresentation{
			Name:        ptr.To(testChildGroupName),
			Description: ptr.To("Child group description"),
			Path:        ptr.To("/child-group"),
			Attributes:  &map[string][]string{"a": {"b"}},
		},
	).Return(&keycloakapi.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{"Location": []string{"http://localhost/admin/realms/test-realm/groups/child-id-456"}},
		},
	}, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "child-id-456", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_UpdateExisting(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testExistingGroup
	group.Spec.Description = "Updated description"
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"new-key": {"new-val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:            ptr.To("existing-id"),
		Name:          ptr.To(testExistingGroup),
		Path:          ptr.To("/old-path"),
		SubGroupCount: ptr.To(int64(2)),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), testRealmName, "existing-id",
		keycloakapi.GroupRepresentation{
			Id:            ptr.To("existing-id"),
			Name:          ptr.To(testExistingGroup),
			Description:   ptr.To("Updated description"),
			Path:          ptr.To(testUpdatedPath),
			Attributes:    &map[string][]string{"new-key": {"new-val"}},
			SubGroupCount: ptr.To(int64(2)),
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "existing-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_FindGroupError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testGroupName,
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to search for group")
}

func TestCreateOrUpdateGroup_Serve_CreateGroupError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = testGroupPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), testRealmName,
		keycloakapi.GroupRepresentation{
			Name:       ptr.To(testGroupName),
			Path:       ptr.To(testGroupPath),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(nil, errors.New("create failed"))

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to create group")
}

func assertNoDescriptionOnWire(t *testing.T, rep keycloakapi.GroupRepresentation) {
	t.Helper()

	body, err := json.Marshal(rep)
	require.NoError(t, err)
	assert.NotContains(t, string(body), `"description"`)
}

func TestCreateOrUpdateGroup_Serve_OmitsEmptyDescription(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		mockGroups := mocks.NewMockGroupsClient(t)

		kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
		groupCtx := &GroupContext{RealmName: testRealmName}

		group := &keycloakApi.KeycloakRealmGroup{}
		group.Spec.Name = testGroupName
		group.Spec.Path = testGroupPath

		mockGroups.EXPECT().FindGroupByName(
			context.Background(), testRealmName, testGroupName,
		).Return(nil, nil, keycloakapi.ErrNotFound)

		mockGroups.EXPECT().CreateGroup(
			context.Background(), testRealmName,
			keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Path:       ptr.To(testGroupPath),
				Attributes: &group.Spec.Attributes,
			},
		).RunAndReturn(func(_ context.Context, _ string, rep keycloakapi.GroupRepresentation) (*keycloakapi.Response, error) {
			assertNoDescriptionOnWire(t, rep)

			return &keycloakapi.Response{
				HTTPResponse: &http.Response{
					Header: http.Header{"Location": []string{"http://localhost/admin/realms/test-realm/groups/group-id-123"}},
				},
			}, nil
		})

		h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
		require.NoError(t, h.Serve(context.Background(), group, kClient, groupCtx))
	})

	t.Run("update over a stored description", func(t *testing.T) {
		mockGroups := mocks.NewMockGroupsClient(t)

		kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
		groupCtx := &GroupContext{RealmName: testRealmName}

		group := &keycloakApi.KeycloakRealmGroup{}
		group.Spec.Name = testExistingGroup
		group.Spec.Path = testUpdatedPath

		mockGroups.EXPECT().FindGroupByName(
			context.Background(), testRealmName, testExistingGroup,
		).Return(&keycloakapi.GroupRepresentation{
			Id:          ptr.To("existing-id"),
			Name:        ptr.To(testExistingGroup),
			Description: ptr.To("stale description"),
			Path:        ptr.To("/old-path"),
		}, nil, nil)

		mockGroups.EXPECT().GetGroup(
			context.Background(), testRealmName, "existing-id",
		).Return(&keycloakapi.GroupRepresentation{
			Id:          ptr.To("existing-id"),
			Name:        ptr.To(testExistingGroup),
			Description: ptr.To("stale description"),
			Path:        ptr.To("/old-path"),
		}, nil, nil)

		mockGroups.EXPECT().UpdateGroup(
			context.Background(), testRealmName, "existing-id",
			keycloakapi.GroupRepresentation{
				Id:         ptr.To("existing-id"),
				Name:       ptr.To(testExistingGroup),
				Path:       ptr.To(testUpdatedPath),
				Attributes: &group.Spec.Attributes,
			},
		).RunAndReturn(func(_ context.Context, _, _ string, rep keycloakapi.GroupRepresentation) (*keycloakapi.Response, error) {
			assertNoDescriptionOnWire(t, rep)

			return nil, nil
		})

		h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
		require.NoError(t, h.Serve(context.Background(), group, kClient, groupCtx))
	})
}

func TestCreateOrUpdateGroup_Serve_UpdateGroupError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testExistingGroup
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), testRealmName, "existing-id",
		keycloakapi.GroupRepresentation{
			Id:         ptr.To("existing-id"),
			Name:       ptr.To(testExistingGroup),
			Path:       ptr.To(testUpdatedPath),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(nil, errors.New("update failed"))

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to update group")
}

func TestCreateOrUpdateGroup_Serve_UpdateExistingChildGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName, ParentGroupID: "parent-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testChildGroupName
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"k": {"v"}}

	mockGroups.EXPECT().FindChildGroupByName(
		context.Background(), testRealmName, "parent-id", testChildGroupName,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("child-id"),
		Name: ptr.To(testChildGroupName),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "child-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("child-id"),
		Name: ptr.To(testChildGroupName),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), testRealmName, "child-id",
		keycloakapi.GroupRepresentation{
			Id:         ptr.To("child-id"),
			Name:       ptr.To(testChildGroupName),
			Path:       ptr.To(testUpdatedPath),
			Attributes: &map[string][]string{"k": {"v"}},
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "child-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_FindChildGroupError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName, ParentGroupID: "parent-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testChildGroupName

	mockGroups.EXPECT().FindChildGroupByName(
		context.Background(), testRealmName, "parent-id", testChildGroupName,
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to search for group")
}

func TestCreateOrUpdateGroup_Serve_RenameByID(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName, GroupID: "existing-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = "new-name"
	group.Spec.Description = "Updated desc"
	group.Spec.Path = "/new-name"
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To("old-name"),
		Path: ptr.To("/old-name"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), testRealmName, "existing-id",
		keycloakapi.GroupRepresentation{
			Id:          ptr.To("existing-id"),
			Name:        ptr.To("new-name"),
			Description: ptr.To("Updated desc"),
			Path:        ptr.To("/new-name"),
			Attributes:  &map[string][]string{"key": {"val"}},
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "existing-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_ExistingGroupWithoutID(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName, GroupID: "existing-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(&keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName)}, nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "has no id")
}

func TestCreateOrUpdateGroup_Serve_GroupFoundByNameWithoutID(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: testNamespace},
	}
	group.Spec.Name = testGroupName

	// The ownership check and the update both need the id, so neither may run.
	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testGroupName,
	).Return(&keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName)}, nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "has no id")
	assert.Empty(t, groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_CreateWithoutLocationHeader(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = testGroupPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	// Keycloak reports the new id in the Location header only. Without it the chain would
	// carry an empty group id into every later handler.
	mockGroups.EXPECT().CreateGroup(
		context.Background(), testRealmName,
		keycloakapi.GroupRepresentation{
			Name:       ptr.To(testGroupName),
			Path:       ptr.To(testGroupPath),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(&keycloakapi.Response{HTTPResponse: &http.Response{Header: http.Header{}}}, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "Location header missing or empty")
	assert.Empty(t, groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_ExistingIDNotFound_FallsBackToName(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName, GroupID: "deleted-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = testGroupPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "deleted-id",
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), testRealmName,
		keycloakapi.GroupRepresentation{
			Name:       ptr.To(testGroupName),
			Path:       ptr.To(testGroupPath),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(&keycloakapi.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{"Location": []string{"http://localhost/admin/realms/test-realm/groups/new-id"}},
		},
	}, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "new-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_GetGroupByIDError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName, GroupID: "existing-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(nil, nil, errors.New("connection error"))

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get group by ID")
}

func TestCreateOrUpdateGroup_Serve_AdoptUnownedGroupByName(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: testNamespace},
	}
	group.Spec.Name = testExistingGroup
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), testRealmName, "existing-id",
		keycloakapi.GroupRepresentation{
			Id:         ptr.To("existing-id"),
			Name:       ptr.To(testExistingGroup),
			Path:       ptr.To(testUpdatedPath),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(nil, nil)

	// No other KeycloakRealmGroup CR owns "existing-id" - the group is up for adoption.
	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "existing-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_RefusesToAdoptGroupOwnedByAnotherCR(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: testNamespace},
	}
	group.Spec.Name = testExistingGroup

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: testNamespace},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "existing-id"},
	}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
	}, nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t, owner))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "already managed by KeycloakRealmGroup "+testNamespace+"/"+testCRNameA)
	assert.Empty(t, groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_AllowsSameGroupIDOnNamespacedRealmInAnotherNamespace(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: testNamespace},
	}
	group.Spec.Name = testExistingGroup
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}
	group.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	group.Spec.RealmRef.Name = testRealmRefName

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "other-ns"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "existing-id"},
	}
	owner.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	owner.Spec.RealmRef.Name = testRealmRefName

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), testRealmName, "existing-id",
		keycloakapi.GroupRepresentation{
			Id:         ptr.To("existing-id"),
			Name:       ptr.To(testExistingGroup),
			Path:       ptr.To(testUpdatedPath),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(nil, nil)

	objs := make([]client.Object, 0, 5)
	objs = append(objs, owner)
	objs = append(objs, namespacedKeycloakStack(testNamespace, testRealmName, "http://kc-a.example")...)
	objs = append(objs, namespacedKeycloakStack("other-ns", testRealmName, "http://kc-b.example")...)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t, objs...))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "existing-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_RefusesSameGroupIDOnClusterRealmAcrossNamespaces(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: testNamespace},
	}
	group.Spec.Name = testExistingGroup
	group.Spec.RealmRef.Kind = keycloakAlpha.ClusterKeycloakRealmKind
	group.Spec.RealmRef.Name = "shared-realm"

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "other-ns"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "existing-id"},
	}
	owner.Spec.RealmRef.Kind = keycloakAlpha.ClusterKeycloakRealmKind
	owner.Spec.RealmRef.Name = "shared-realm"

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
	}, nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t, owner))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "already managed by KeycloakRealmGroup other-ns/"+testCRNameA)
	assert.Empty(t, groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_RefusesSameGroupIDOnSharedKeycloakURLAcrossNamespaces(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: testNamespace},
	}
	group.Spec.Name = testExistingGroup
	group.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	group.Spec.RealmRef.Name = testRealmRefName

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "other-ns"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "existing-id"},
	}
	owner.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	owner.Spec.RealmRef.Name = testRealmRefName

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
	}, nil, nil)

	objs := make([]client.Object, 0, 5)
	objs = append(objs, owner)
	objs = append(objs, namespacedKeycloakStack(testNamespace, testRealmName, "http://shared.example")...)
	objs = append(objs, namespacedKeycloakStack("other-ns", testRealmName, "http://shared.example/")...)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t, objs...))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "already managed by KeycloakRealmGroup other-ns/"+testCRNameA)
	assert.Empty(t, groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_FailsClosedWhenRealmTargetCannotBeResolved(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: testNamespace},
	}
	group.Spec.Name = testExistingGroup
	group.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	group.Spec.RealmRef.Name = testRealmRefName

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: "other-ns"},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "existing-id"},
	}
	owner.Spec.RealmRef.Kind = keycloakApi.KeycloakRealmKind
	owner.Spec.RealmRef.Name = testRealmRefName

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
	}, nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t, owner))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to resolve realm target for ownership check")
	assert.Empty(t, groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_ByIDPathSkipsOwnershipCheck(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	// This CR's own status.ID is already "existing-id" - the by-ID rename path must not
	// treat that as a conflict, since it is comparing the group against itself.
	groupCtx := &GroupContext{RealmName: testRealmName, GroupID: "existing-id"}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: testNamespace},
	}
	group.Spec.Name = "new-name"
	group.Spec.Path = "/new-name"
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To("old-name"),
		Path: ptr.To("/old-name"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), testRealmName, "existing-id",
		keycloakapi.GroupRepresentation{
			Id:         ptr.To("existing-id"),
			Name:       ptr.To("new-name"),
			Path:       ptr.To("/new-name"),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(nil, nil)

	self := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: testNamespace},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "existing-id"},
	}

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t, self))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "existing-id", groupCtx.GroupID)
}

func TestGroupMatchesSpec(t *testing.T) {
	tests := []struct {
		name     string
		existing keycloakapi.GroupRepresentation
		spec     keycloakApi.KeycloakRealmGroupSpec
		want     bool
	}{
		{
			name: "name, description and attributes equal",
			existing: keycloakapi.GroupRepresentation{
				Name:        ptr.To(testGroupName),
				Description: ptr.To("desc"),
				Attributes:  &map[string][]string{"k": {"a", "b"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:        testGroupName,
				Description: "desc",
				Attributes:  map[string][]string{"k": {"a", "b"}},
			},
			want: true,
		},
		{
			name:     "empty spec description equals absent description",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName)},
			spec:     keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName},
			want:     true,
		},
		{
			name:     "empty spec description equals stored empty description",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName), Description: ptr.To("")},
			spec:     keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName},
			want:     true,
		},
		{
			name:     "path is not compared",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName), Path: ptr.To(testGroupPath)},
			spec:     keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName, Path: "/stale"},
			want:     true,
		},
		{
			name: "values in another order",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"z", "a", "m"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {"a", "m", "z"}},
			},
			want: true,
		},
		{
			name:     "spec key without values equals absent key",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName)},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"empty": {}},
			},
			want: true,
		},
		{
			name: "nil spec attributes leave stored attributes unmanaged",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"a"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName},
			want: true,
		},
		{
			name:     "empty spec attributes equal absent stored attributes",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName)},
			spec:     keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName, Attributes: map[string][]string{}},
			want:     true,
		},
		{
			name: "duplicate values on both sides",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"a", "a"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {"a", "a"}},
			},
			want: true,
		},
		{
			name: "blank value on both sides",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {""}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {""}},
			},
			want: true,
		},
		{
			name: "blank and set value in another order",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"x", ""}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {"", "x"}},
			},
			want: true,
		},
		{
			name:     "name differs",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To("other")},
			spec:     keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName},
		},
		{
			name:     "description differs",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName), Description: ptr.To("old")},
			spec:     keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName, Description: "new"},
		},
		{
			name:     "spec description set, stored description absent",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName)},
			spec:     keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName, Description: "new"},
		},
		{
			name:     "spec description empty, stored description set",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName), Description: ptr.To("old")},
			spec:     keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName},
		},
		{
			name: "stored attribute key is not in the spec",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"a"}, "extra": {"x"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {"a"}},
			},
		},
		{
			name: "spec attribute key is not stored",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"a"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {"a"}, "missing": {"x"}},
			},
		},
		{
			name: "attribute value differs",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"a"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {"b"}},
			},
		},
		{
			name: "duplicate value stored once",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"a"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {"a", "a"}},
			},
		},
		{
			name: "empty spec attributes, stored attribute present",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"a"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName, Attributes: map[string][]string{}},
		},
		{
			name: "spec key without values, stored key has a value",
			existing: keycloakapi.GroupRepresentation{
				Name:       ptr.To(testGroupName),
				Attributes: &map[string][]string{"k": {"x"}},
			},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {}},
			},
		},
		{
			name:     "spec key with a blank value, stored key absent",
			existing: keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName)},
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {""}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, groupMatchesSpec(&tt.existing, &tt.spec))
		})
	}
}

func TestCreateOrUpdateGroup_Serve_SkipsUpdateWhenGroupMatches(t *testing.T) {
	tests := []struct {
		name     string
		spec     keycloakApi.KeycloakRealmGroupSpec
		groupCtx GroupContext
		mocks    func(m *mocks.MockGroupsClient)
		wantID   string
	}{
		{
			name:     "by id, name only",
			spec:     keycloakApi.KeycloakRealmGroupSpec{Name: testGroupName},
			groupCtx: GroupContext{RealmName: testRealmName, GroupID: "existing-id"},
			mocks: func(m *mocks.MockGroupsClient) {
				m.EXPECT().GetGroup(
					context.Background(), testRealmName, "existing-id",
				).Return(&keycloakapi.GroupRepresentation{
					Id:   ptr.To("existing-id"),
					Name: ptr.To(testGroupName),
					Path: ptr.To(testGroupPath),
				}, nil, nil)
			},
			wantID: "existing-id",
		},
		{
			name: "top-level name hit, brief search result, full group equal",
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testGroupName,
				Attributes: map[string][]string{"k": {"v"}},
			},
			groupCtx: GroupContext{RealmName: testRealmName},
			mocks: func(m *mocks.MockGroupsClient) {
				m.EXPECT().FindGroupByName(
					context.Background(), testRealmName, testGroupName,
				).Return(&keycloakapi.GroupRepresentation{
					Id:   ptr.To("existing-id"),
					Name: ptr.To(testGroupName),
					Path: ptr.To(testGroupPath),
				}, nil, nil)

				m.EXPECT().GetGroup(
					context.Background(), testRealmName, "existing-id",
				).Return(&keycloakapi.GroupRepresentation{
					Id:         ptr.To("existing-id"),
					Name:       ptr.To(testGroupName),
					Path:       ptr.To(testGroupPath),
					Attributes: &map[string][]string{"k": {"v"}},
				}, nil, nil)
			},
			wantID: "existing-id",
		},
		{
			name: "child name hit, brief search result, full group equal",
			spec: keycloakApi.KeycloakRealmGroupSpec{
				Name:       testChildGroupName,
				Attributes: map[string][]string{"k": {"v"}},
			},
			groupCtx: GroupContext{RealmName: testRealmName, ParentGroupID: "parent-id"},
			mocks: func(m *mocks.MockGroupsClient) {
				m.EXPECT().FindChildGroupByName(
					context.Background(), testRealmName, "parent-id", testChildGroupName,
				).Return(&keycloakapi.GroupRepresentation{
					Id:   ptr.To("child-id"),
					Name: ptr.To(testChildGroupName),
				}, nil, nil)

				m.EXPECT().GetGroup(
					context.Background(), testRealmName, "child-id",
				).Return(&keycloakapi.GroupRepresentation{
					Id:         ptr.To("child-id"),
					Name:       ptr.To(testChildGroupName),
					Path:       ptr.To("/parent/child-group"),
					Attributes: &map[string][]string{"k": {"v"}},
				}, nil, nil)
			},
			wantID: "child-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGroups := mocks.NewMockGroupsClient(t)
			tt.mocks(mockGroups)

			group := &keycloakApi.KeycloakRealmGroup{Spec: tt.spec}
			groupCtx := tt.groupCtx

			h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
			err := h.Serve(context.Background(), group, &keycloakapi.KeycloakClient{Groups: mockGroups}, &groupCtx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, groupCtx.GroupID)
		})
	}
}

func TestCreateOrUpdateGroup_Serve_NameHitComparesFullGroup(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Attributes = map[string][]string{}

	// The search result omits attributes, so only the full group shows the stale key.
	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testGroupName,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testGroupName),
	}, nil, nil)

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:         ptr.To("existing-id"),
		Name:       ptr.To(testGroupName),
		Attributes: &map[string][]string{"stale": {"x"}},
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), testRealmName, "existing-id",
		keycloakapi.GroupRepresentation{
			Id:         ptr.To("existing-id"),
			Name:       ptr.To(testGroupName),
			Path:       ptr.To(""),
			Attributes: &map[string][]string{},
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	require.NoError(t, h.Serve(context.Background(), group, kClient, groupCtx))
	assert.Equal(t, "existing-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_NameHitGetByIDError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: testRealmName}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), testRealmName, testGroupName,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testGroupName),
	}, nil, nil)

	mockGroups.EXPECT().GetGroup(
		context.Background(), testRealmName, "existing-id",
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get group")
	assert.Empty(t, groupCtx.GroupID)
}
