package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

func TestSetScopeType_Serve_Default(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeDefault
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().AddRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSetScopeType_Serve_Optional(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeOptional
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().AddRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSetScopeType_Serve_None(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeNone
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSetScopeType_Serve_DefaultRemoveOptionalNotFound(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeDefault
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, keycloakapi.ErrNotFound)

	mockScopes.EXPECT().AddRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSetScopeType_Serve_AddDefaultError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeDefault
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().AddRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, errors.New("api error"))

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add scope to default list")
}

func TestSetScopeType_Serve_RemoveOptionalError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeDefault
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, errors.New("api error"))

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove scope from optional list")
}

func TestSetScopeType_Serve_DeprecatedDefaultField(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	// Test backward compat: Default=true should be treated as "default" type
	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Default = true
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeNone
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().AddRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSetScopeType_Serve_OptionalRemoveDefaultNotFound(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeOptional
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, keycloakapi.ErrNotFound)

	mockScopes.EXPECT().AddRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSetScopeType_Serve_OptionalRemoveDefaultError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeOptional
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, errors.New("api error"))

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove scope from default list")
}

func TestSetScopeType_Serve_OptionalAddOptionalError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeOptional
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().AddRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, errors.New("api error"))

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add scope to optional list")
}

func TestSetScopeType_Serve_NoneRemoveDefaultNotFound(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeNone
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, keycloakapi.ErrNotFound)

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSetScopeType_Serve_NoneRemoveDefaultError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeNone
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, errors.New("api error"))

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove scope from default list")
}

func TestSetScopeType_Serve_NoneRemoveOptionalNotFound(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeNone
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, keycloakapi.ErrNotFound)

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSetScopeType_Serve_NoneRemoveOptionalError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = keycloakApi.KeycloakClientScopeTypeNone
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().RemoveRealmDefaultClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil)

	mockScopes.EXPECT().RemoveRealmOptionalClientScope(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, errors.New("api error"))

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove scope from optional list")
}

func TestSetScopeType_Serve_InvalidType(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Type = "bogus"
	scope.Status.ID = testScopeID

	h := NewSetScopeType(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid client scope type")
}
