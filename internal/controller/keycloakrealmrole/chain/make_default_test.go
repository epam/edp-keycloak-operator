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

func TestMakeDefault_Serve_NotDefault(t *testing.T) {
	kClient := &keycloakapi.APIClient{}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.IsDefault = false

	h := NewMakeDefault(kClient)
	err := h.Serve(context.Background(), role, "test-realm", &RoleContext{})
	require.NoError(t, err)
}

func TestMakeDefault_Serve_Success(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.APIClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName
	role.Spec.IsDefault = true

	roleCtx := &RoleContext{RoleID: "role-id-123"}

	mockRoles.EXPECT().AddRealmRoleComposites(
		context.Background(), "test-realm", "default-roles-test-realm",
		[]keycloakapi.RoleRepresentation{
			{
				Id:   ptr.To("role-id-123"),
				Name: ptr.To(testRoleName),
			},
		},
	).Return(nil, nil)

	h := NewMakeDefault(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.NoError(t, err)
}

func TestMakeDefault_Serve_Error(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.APIClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName
	role.Spec.IsDefault = true

	roleCtx := &RoleContext{RoleID: "role-id-123"}

	mockRoles.EXPECT().AddRealmRoleComposites(
		context.Background(), "test-realm", "default-roles-test-realm",
		[]keycloakapi.RoleRepresentation{
			{
				Id:   ptr.To("role-id-123"),
				Name: ptr.To(testRoleName),
			},
		},
	).Return(nil, errors.New("api error"))

	h := NewMakeDefault(kClient)
	err := h.Serve(context.Background(), role, "test-realm", roleCtx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add role to default-roles")
}
