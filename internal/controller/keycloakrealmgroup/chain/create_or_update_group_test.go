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
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = testGroupPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
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
	groupCtx := &GroupContext{RealmName: "test-realm", ParentGroupID: "parent-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testChildGroupName
	group.Spec.Description = "Child group description"
	group.Spec.Path = "/child-group"
	group.Spec.Attributes = map[string][]string{"a": {"b"}}

	mockGroups.EXPECT().FindChildGroupByName(
		context.Background(), "test-realm", "parent-id", testChildGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().CreateChildGroup(
		context.Background(), "test-realm", "parent-id",
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
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testExistingGroup
	group.Spec.Description = "Updated description"
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"new-key": {"new-val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "existing-id",
		keycloakapi.GroupRepresentation{
			Id:          ptr.To("existing-id"),
			Name:        ptr.To(testExistingGroup),
			Description: ptr.To("Updated description"),
			Path:        ptr.To(testUpdatedPath),
			Attributes:  &map[string][]string{"new-key": {"new-val"}},
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
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testGroupName,
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to search for group")
}

func TestCreateOrUpdateGroup_Serve_CreateGroupError(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = testGroupPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
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
		groupCtx := &GroupContext{RealmName: "test-realm"}

		group := &keycloakApi.KeycloakRealmGroup{}
		group.Spec.Name = testGroupName
		group.Spec.Path = testGroupPath

		mockGroups.EXPECT().FindGroupByName(
			context.Background(), "test-realm", testGroupName,
		).Return(nil, nil, keycloakapi.ErrNotFound)

		mockGroups.EXPECT().CreateGroup(
			context.Background(), "test-realm",
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
		groupCtx := &GroupContext{RealmName: "test-realm"}

		group := &keycloakApi.KeycloakRealmGroup{}
		group.Spec.Name = testExistingGroup
		group.Spec.Path = testUpdatedPath

		mockGroups.EXPECT().FindGroupByName(
			context.Background(), "test-realm", testExistingGroup,
		).Return(&keycloakapi.GroupRepresentation{
			Id:          ptr.To("existing-id"),
			Name:        ptr.To(testExistingGroup),
			Description: ptr.To("stale description"),
			Path:        ptr.To("/old-path"),
		}, nil, nil)

		mockGroups.EXPECT().UpdateGroup(
			context.Background(), "test-realm", "existing-id",
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
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testExistingGroup
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "existing-id",
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
	groupCtx := &GroupContext{RealmName: "test-realm", ParentGroupID: "parent-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testChildGroupName
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"k": {"v"}}

	mockGroups.EXPECT().FindChildGroupByName(
		context.Background(), "test-realm", "parent-id", testChildGroupName,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("child-id"),
		Name: ptr.To(testChildGroupName),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "child-id",
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
	groupCtx := &GroupContext{RealmName: "test-realm", ParentGroupID: "parent-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testChildGroupName

	mockGroups.EXPECT().FindChildGroupByName(
		context.Background(), "test-realm", "parent-id", testChildGroupName,
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to search for group")
}

func TestCreateOrUpdateGroup_Serve_RenameByID(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm", GroupID: "existing-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = "new-name"
	group.Spec.Description = "Updated desc"
	group.Spec.Path = "/new-name"
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().GetGroup(
		context.Background(), "test-realm", "existing-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To("old-name"),
		Path: ptr.To("/old-name"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "existing-id",
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
	groupCtx := &GroupContext{RealmName: "test-realm", GroupID: "existing-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName

	mockGroups.EXPECT().GetGroup(
		context.Background(), "test-realm", "existing-id",
	).Return(&keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName)}, nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "has no id")
}

func TestCreateOrUpdateGroup_Serve_GroupFoundByNameWithoutID(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: testNamespace},
	}
	group.Spec.Name = testGroupName

	// The ownership check and the update both need the id, so neither may run.
	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testGroupName,
	).Return(&keycloakapi.GroupRepresentation{Name: ptr.To(testGroupName)}, nil, nil)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "has no id")
	assert.Empty(t, groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_CreateWithoutLocationHeader(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = testGroupPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	// Keycloak reports the new id in the Location header only. Without it the chain would
	// carry an empty group id into every later handler.
	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
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
	groupCtx := &GroupContext{RealmName: "test-realm", GroupID: "deleted-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName
	group.Spec.Path = testGroupPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().GetGroup(
		context.Background(), "test-realm", "deleted-id",
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testGroupName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	mockGroups.EXPECT().CreateGroup(
		context.Background(), "test-realm",
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
	groupCtx := &GroupContext{RealmName: "test-realm", GroupID: "existing-id"}

	group := &keycloakApi.KeycloakRealmGroup{}
	group.Spec.Name = testGroupName

	mockGroups.EXPECT().GetGroup(
		context.Background(), "test-realm", "existing-id",
	).Return(nil, nil, errors.New("connection error"))

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "unable to get group by ID")
}

func TestCreateOrUpdateGroup_Serve_AdoptUnownedGroupByName(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: testNamespace},
	}
	group.Spec.Name = testExistingGroup
	group.Spec.Path = testUpdatedPath
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "existing-id",
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
	groupCtx := &GroupContext{RealmName: "test-realm"}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameB, Namespace: testNamespace},
	}
	group.Spec.Name = testExistingGroup

	owner := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: testNamespace},
		Status:     keycloakApi.KeycloakRealmGroupStatus{ID: "existing-id"},
	}

	mockGroups.EXPECT().FindGroupByName(
		context.Background(), "test-realm", testExistingGroup,
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
	groupCtx := &GroupContext{RealmName: "test-realm"}

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
		context.Background(), "test-realm", testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
		Path: ptr.To("/old-path"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "existing-id",
		keycloakapi.GroupRepresentation{
			Id:         ptr.To("existing-id"),
			Name:       ptr.To(testExistingGroup),
			Path:       ptr.To(testUpdatedPath),
			Attributes: &map[string][]string{"key": {"val"}},
		},
	).Return(nil, nil)

	objs := make([]client.Object, 0, 5)
	objs = append(objs, owner)
	objs = append(objs, namespacedKeycloakStack(testNamespace, "test-realm", "http://kc-a.example")...)
	objs = append(objs, namespacedKeycloakStack("other-ns", "test-realm", "http://kc-b.example")...)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t, objs...))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	require.NoError(t, err)
	assert.Equal(t, "existing-id", groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_RefusesSameGroupIDOnClusterRealmAcrossNamespaces(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

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
		context.Background(), "test-realm", testExistingGroup,
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
	groupCtx := &GroupContext{RealmName: "test-realm"}

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
		context.Background(), "test-realm", testExistingGroup,
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To(testExistingGroup),
	}, nil, nil)

	objs := make([]client.Object, 0, 5)
	objs = append(objs, owner)
	objs = append(objs, namespacedKeycloakStack(testNamespace, "test-realm", "http://shared.example")...)
	objs = append(objs, namespacedKeycloakStack("other-ns", "test-realm", "http://shared.example/")...)

	h := NewCreateOrUpdateGroup(newFakeK8sClient(t, objs...))
	err := h.Serve(context.Background(), group, kClient, groupCtx)
	assert.ErrorContains(t, err, "already managed by KeycloakRealmGroup other-ns/"+testCRNameA)
	assert.Empty(t, groupCtx.GroupID)
}

func TestCreateOrUpdateGroup_Serve_FailsClosedWhenRealmTargetCannotBeResolved(t *testing.T) {
	mockGroups := mocks.NewMockGroupsClient(t)

	kClient := &keycloakapi.KeycloakClient{Groups: mockGroups}
	groupCtx := &GroupContext{RealmName: "test-realm"}

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
		context.Background(), "test-realm", testExistingGroup,
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
	groupCtx := &GroupContext{RealmName: "test-realm", GroupID: "existing-id"}

	group := &keycloakApi.KeycloakRealmGroup{
		ObjectMeta: metav1.ObjectMeta{Name: testCRNameA, Namespace: testNamespace},
	}
	group.Spec.Name = "new-name"
	group.Spec.Path = "/new-name"
	group.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockGroups.EXPECT().GetGroup(
		context.Background(), "test-realm", "existing-id",
	).Return(&keycloakapi.GroupRepresentation{
		Id:   ptr.To("existing-id"),
		Name: ptr.To("old-name"),
		Path: ptr.To("/old-name"),
	}, nil, nil)

	mockGroups.EXPECT().UpdateGroup(
		context.Background(), "test-realm", "existing-id",
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
