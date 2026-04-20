package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const testRoleName = "test-role"

func TestRemoveRole_ServeRequest_Success(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName

	mockRoles.EXPECT().DeleteRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, nil)

	h := NewRemoveRole(kClient)
	err := h.ServeRequest(context.Background(), role, "test-realm")
	require.NoError(t, err)
}

func TestRemoveRole_ServeRequest_NotFound(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName

	mockRoles.EXPECT().DeleteRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, keycloakapi.ErrNotFound)

	h := NewRemoveRole(kClient)
	err := h.ServeRequest(context.Background(), role, "test-realm")
	require.NoError(t, err)
}

func TestRemoveRole_ServeRequest_PreserveOnDeletion(t *testing.T) {
	kClient := &keycloakapi.KeycloakClient{}

	role := &keycloakApi.KeycloakRealmRole{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"edp.epam.com/preserve-resources-on-deletion": "true",
			},
		},
	}
	role.Spec.Name = testRoleName

	h := NewRemoveRole(kClient)
	err := h.ServeRequest(context.Background(), role, "test-realm")
	require.NoError(t, err)
}

func TestRemoveRole_ServeRequest_DeleteError(t *testing.T) {
	mockRoles := mocks.NewMockRolesClient(t)
	kClient := &keycloakapi.KeycloakClient{Roles: mockRoles}

	role := &keycloakApi.KeycloakRealmRole{}
	role.Spec.Name = testRoleName

	mockRoles.EXPECT().DeleteRealmRole(
		context.Background(), "test-realm", testRoleName,
	).Return(nil, errors.New("api error"))

	h := NewRemoveRole(kClient)
	err := h.ServeRequest(context.Background(), role, "test-realm")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete realm role")
}
