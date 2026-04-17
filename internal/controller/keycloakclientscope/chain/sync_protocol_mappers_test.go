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

func TestSyncProtocolMappers_Serve_Success(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID
	scope.Spec.ProtocolMappers = []keycloakApi.ProtocolMapper{
		{
			Name:           "groups",
			Protocol:       testProtocolOIDC,
			ProtocolMapper: "oidc-group-membership-mapper",
			Config:         map[string]string{"claim.name": "groups"},
		},
	}

	// Existing mapper to delete
	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return([]keycloakapi.ProtocolMapperRepresentation{
		{Id: ptr.To("old-mapper-id"), Name: ptr.To("old-mapper")},
	}, nil, nil)

	mockScopes.EXPECT().DeleteClientScopeProtocolMapper(
		context.Background(), testRealmName, testScopeID, "old-mapper-id",
	).Return(nil, nil)

	config := map[string]string{"claim.name": "groups"}

	mockScopes.EXPECT().CreateClientScopeProtocolMapper(
		context.Background(), testRealmName, testScopeID,
		keycloakapi.ProtocolMapperRepresentation{
			Name:           ptr.To("groups"),
			Protocol:       ptr.To(testProtocolOIDC),
			ProtocolMapper: ptr.To("oidc-group-membership-mapper"),
			Config:         &config,
		},
	).Return(nil, nil)

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSyncProtocolMappers_Serve_NoMappers(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	// No existing mappers
	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil, nil)

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSyncProtocolMappers_Serve_GetMappersError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil, errors.New("api error"))

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get existing protocol mappers")
}

func TestSyncProtocolMappers_Serve_DeleteMapperError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return([]keycloakapi.ProtocolMapperRepresentation{
		{Id: ptr.To("mapper-id"), Name: ptr.To("mapper")},
	}, nil, nil)

	mockScopes.EXPECT().DeleteClientScopeProtocolMapper(
		context.Background(), testRealmName, testScopeID, "mapper-id",
	).Return(nil, errors.New("delete error"))

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete protocol mapper")
}

func TestSyncProtocolMappers_Serve_NilMapperID(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	// Existing mapper with nil ID should be skipped
	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return([]keycloakapi.ProtocolMapperRepresentation{
		{Id: nil, Name: ptr.To("mapper-without-id")},
	}, nil, nil)

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSyncProtocolMappers_Serve_CreateMapperError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.APIClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID
	scope.Spec.ProtocolMappers = []keycloakApi.ProtocolMapper{
		{
			Name:           "mapper",
			Protocol:       testProtocolOIDC,
			ProtocolMapper: "oidc-audience-mapper",
			Config:         map[string]string{},
		},
	}

	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return(nil, nil, nil)

	config := map[string]string{}

	mockScopes.EXPECT().CreateClientScopeProtocolMapper(
		context.Background(), testRealmName, testScopeID,
		keycloakapi.ProtocolMapperRepresentation{
			Name:           ptr.To("mapper"),
			Protocol:       ptr.To(testProtocolOIDC),
			ProtocolMapper: ptr.To("oidc-audience-mapper"),
			Config:         &config,
		},
	).Return(nil, errors.New("create error"))

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create protocol mapper")
}
