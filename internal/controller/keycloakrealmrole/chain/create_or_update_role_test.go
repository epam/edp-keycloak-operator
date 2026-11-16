package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestCreateOrUpdateRole_Serve_CreateNew(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}
	roleCtx := &RoleContext{}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName
	role.Spec.Description = "Test description"
	role.Spec.Composite = false
	role.Spec.Attributes = map[string][]string{"key": {"val"}}

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil, keycloakapi.ErrNotFound).Once()

	desc := "Test description"
	composite := false
	attrs := map[string][]string{"key": {"val"}}

	mockRoles.EXPECT().CreateRealmRole(
		context.Background(), "test-realm",
		keycloakapi.RoleRepresentation{
			Name:        ptr.To(testRoleName),
			Description: &desc,
			Composite:   &composite,
			Attributes:  &attrs,
		},
	).Return(nil, nil)

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(&keycloakapi.RoleRepresentation{
		Id:   ptr.To("role-id-123"),
		Name: ptr.To(testRoleName),
	}, nil, nil).Once()

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.NoError(t, err)
	assert.Equal(t, "role-id-123", roleCtx.RoleID)
}

func TestCreateOrUpdateRole_Serve_UpdateExisting(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}
	roleCtx := &RoleContext{}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName
	role.Spec.Description = "Updated description"
	role.Spec.Composite = true
	role.Spec.Attributes = map[string][]string{"key": {"new-val"}}

	existingRole := &keycloakapi.RoleRepresentation{
		Id:   ptr.To("role-id-123"),
		Name: ptr.To(testRoleName),
	}

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(existingRole, nil, nil)

	desc := "Updated description"
	composite := true
	attrs := map[string][]string{"key": {"new-val"}}

	mockRoles.EXPECT().UpdateRealmRole(
		context.Background(), "test-realm", testRoleName,
		keycloakapi.RoleRepresentation{
			Id:          ptr.To("role-id-123"),
			Name:        ptr.To(testRoleName),
			Description: &desc,
			Composite:   &composite,
			Attributes:  &attrs,
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.NoError(t, err)
	assert.Equal(t, "role-id-123", roleCtx.RoleID)
}

func TestCreateOrUpdateRole_Serve_GetRealmRoleError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get realm role")
}

func TestCreateOrUpdateRole_Serve_MissingAfterCreate(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}
	roleCtx := &RoleContext{}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil, keycloakapi.ErrNotFound).Once()

	mockRoles.EXPECT().CreateRealmRole(
		context.Background(), "test-realm", mock.Anything,
	).Return(nil, nil)

	// No role and no error on the re-read. Report it instead of dereferencing nil.
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil, nil).Once()

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not addressable after sync")
	assert.Empty(t, roleCtx.RoleID)
}

func TestCreateOrUpdateRole_Serve_ExistingRoleWithoutID(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}
	roleCtx := &RoleContext{}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName

	// The existing role has no id (and already matches the empty spec, so no update is sent).
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(&keycloakapi.RoleRepresentation{Name: ptr.To(testRoleName)}, nil, nil)

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `realm role "test-role" has no id`)
	assert.Empty(t, roleCtx.RoleID)
}

func TestCreateOrUpdateRole_Serve_CreateError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil, keycloakapi.ErrNotFound)

	var nilAttrs map[string][]string

	mockRoles.EXPECT().CreateRealmRole(
		context.Background(), "test-realm",
		keycloakapi.RoleRepresentation{
			Name:        ptr.To(testRoleName),
			Description: ptr.To(""),
			Composite:   ptr.To(false),
			Attributes:  &nilAttrs,
		},
	).Return(nil, errors.New("create error"))

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create realm role")
}

func TestCreateOrUpdateRole_Serve_UpdateError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName
	role.Spec.Description = "new description" // differs from Keycloak, so an update is sent

	existingRole := &keycloakapi.RoleRepresentation{
		Id:   ptr.To("role-id-123"),
		Name: ptr.To(testRoleName),
	}

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(existingRole, nil, nil)

	var nilAttrs map[string][]string

	mockRoles.EXPECT().UpdateRealmRole(
		context.Background(), "test-realm", testRoleName,
		keycloakapi.RoleRepresentation{
			Id:          ptr.To("role-id-123"),
			Name:        ptr.To(testRoleName),
			Description: ptr.To("new description"),
			Composite:   ptr.To(false),
			Attributes:  &nilAttrs,
		},
	).Return(nil, errors.New("update error"))

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update realm role")
}

func TestCreateOrUpdateRole_Serve_UpToDateSkipsUpdate(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}
	roleCtx := &RoleContext{}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName
	role.Spec.Description = "Same description"
	role.Spec.Composite = true
	role.Spec.Attributes = map[string][]string{"key": {"b", "a"}, "empty": {}}

	// Same values as the spec (value order and empty keys do not matter): no UpdateRealmRole call.
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(&keycloakapi.RoleRepresentation{
		Id:          ptr.To("role-id-123"),
		Name:        ptr.To(testRoleName),
		Description: ptr.To("Same description"),
		Composite:   ptr.To(true),
		Attributes:  &map[string][]string{"key": {"a", "b"}},
	}, nil, nil)

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.NoError(t, err)
	assert.Equal(t, "role-id-123", roleCtx.RoleID)
}

func TestCreateOrUpdateRole_Serve_NilSpecAttributesNotManaged(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}
	roleCtx := &RoleContext{}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName
	role.Spec.Description = "desc"

	// Attributes set by hand in Keycloak, none in the spec: no update.
	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(&keycloakapi.RoleRepresentation{
		Id:          ptr.To("role-id-123"),
		Name:        ptr.To(testRoleName),
		Description: ptr.To("desc"),
		Attributes:  &map[string][]string{"manual": {"x"}},
	}, nil, nil)

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.NoError(t, err)
	assert.Equal(t, "role-id-123", roleCtx.RoleID)
}

func TestCreateOrUpdateRole_Serve_AttributeDriftUpdates(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}
	roleCtx := &RoleContext{}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName
	role.Spec.Description = "desc"
	role.Spec.Attributes = map[string][]string{"key": {"want"}}

	mockRoles.EXPECT().GetRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(&keycloakapi.RoleRepresentation{
		Id:          ptr.To("role-id-123"),
		Name:        ptr.To(testRoleName),
		Description: ptr.To("desc"),
		Attributes:  &map[string][]string{"key": {"have"}},
	}, nil, nil)

	mockRoles.EXPECT().UpdateRealmRole(
		context.Background(), "test-realm", testRoleName, mock.Anything,
	).Return(nil, nil).Once()

	h := NewCreateOrUpdateRole(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.NoError(t, err)
	assert.Equal(t, "role-id-123", roleCtx.RoleID)
}
