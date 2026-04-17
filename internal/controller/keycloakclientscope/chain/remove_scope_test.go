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

func TestRemoveScope_Serve_Success(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().DeleteClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	h := NewRemoveScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestRemoveScope_Serve_NotFound(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, keycloakapi.ErrNotFound)

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, keycloakapi.ErrNotFound)

	mockScopes.EXPECT().DeleteClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, keycloakapi.ErrNotFound)

	h := NewRemoveScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestRemoveScope_Serve_PreserveOnDeletion(t *testing.T) {
	kClient := &keycloakapi.APIClient{}

	scope := &keycloakApi.KeycloakClientScope{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"edp.epam.com/preserve-resources-on-deletion": "true",
			},
		},
	}
	scope.Status.ID = testScopeID

	h := NewRemoveScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestRemoveScope_Serve_DeleteError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().DeleteClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, errors.New("api error"))

	h := NewRemoveScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete client scope")
}

func TestRemoveScope_Serve_RemoveDefaultError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, errors.New("api error"))

	h := NewRemoveScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove scope from default list")
}

func TestRemoveScope_Serve_RemoveOptionalError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, errors.New("api error"))

	h := NewRemoveScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove scope from optional list")
}
