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

const (
	testMapperID   = "mapper-id"
	testMapperName = "groups"
	testMapperType = "oidc-group-membership-mapper"
)

func groupsMapperScope(config map[string]string) *keycloakApi.KeycloakClientScope {
	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID
	scope.Spec.ProtocolMappers = []keycloakApi.ProtocolMapper{
		{
			Name:           testMapperName,
			Protocol:       testProtocolOIDC,
			ProtocolMapper: testMapperType,
			Config:         config,
		},
	}

	return scope
}

func groupsMapperRepr(config map[string]string) keycloakapi.ProtocolMapperRepresentation {
	return keycloakapi.ProtocolMapperRepresentation{
		Id:             ptr.To(testMapperID),
		Name:           ptr.To(testMapperName),
		Protocol:       ptr.To(testProtocolOIDC),
		ProtocolMapper: ptr.To(testMapperType),
		Config:         &config,
	}
}

func TestSyncProtocolMappers_Serve_Success(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := groupsMapperScope(map[string]string{"claim.name": "groups"})

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
			Name:           ptr.To(testMapperName),
			Protocol:       ptr.To(testProtocolOIDC),
			ProtocolMapper: ptr.To(testMapperType),
			Config:         &config,
		},
	).Return(nil, nil)

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSyncProtocolMappers_Serve_NoMappers(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

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
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

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
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Status.ID = testScopeID

	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return([]keycloakapi.ProtocolMapperRepresentation{
		{Id: ptr.To(testMapperID), Name: ptr.To("mapper")},
	}, nil, nil)

	mockScopes.EXPECT().DeleteClientScopeProtocolMapper(
		context.Background(), testRealmName, testScopeID, testMapperID,
	).Return(nil, errors.New("delete error"))

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete protocol mapper")
}

func TestSyncProtocolMappers_Serve_NilMapperID(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

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

func TestSyncProtocolMappers_Serve_SkipWhenInSync(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := groupsMapperScope(map[string]string{"claim.name": "groups"})

	// Server-added default keys must not trigger an update.
	// Existing state matches spec: no write calls expected.
	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return([]keycloakapi.ProtocolMapperRepresentation{
		groupsMapperRepr(map[string]string{"claim.name": "groups", "introspection.token.claim": "true"}),
	}, nil, nil)

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSyncProtocolMappers_Serve_ForceUpdateOnSpecChange(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	// Generation ahead of ObservedGeneration: PUT must fire even though the
	// declared keys match, so keys removed from the spec are dropped in Keycloak.
	scope := groupsMapperScope(map[string]string{"claim.name": "groups"})
	scope.Generation = 2
	scope.Status.ObservedGeneration = 1

	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return([]keycloakapi.ProtocolMapperRepresentation{
		groupsMapperRepr(map[string]string{"claim.name": "groups", "removed.key": "stale"}),
	}, nil, nil)

	mockScopes.EXPECT().UpdateClientScopeProtocolMapper(
		context.Background(), testRealmName, testScopeID, testMapperID,
		groupsMapperRepr(map[string]string{"claim.name": "groups"}),
	).Return(nil, nil)

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSyncProtocolMappers_Serve_EmptyValueKeyDetected(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := groupsMapperScope(map[string]string{"empty.key": ""})

	// Key absent server-side vs declared with "" value: must trigger an update.
	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return([]keycloakapi.ProtocolMapperRepresentation{
		groupsMapperRepr(map[string]string{}),
	}, nil, nil)

	mockScopes.EXPECT().UpdateClientScopeProtocolMapper(
		context.Background(), testRealmName, testScopeID, testMapperID,
		mock.Anything,
	).Return(nil, nil)

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSyncProtocolMappers_Serve_UpdateChangedMapper(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := groupsMapperScope(map[string]string{"claim.name": "groups", "full.path": "true"})

	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return([]keycloakapi.ProtocolMapperRepresentation{
		groupsMapperRepr(map[string]string{"claim.name": "groups", "full.path": "false"}),
	}, nil, nil)

	mockScopes.EXPECT().UpdateClientScopeProtocolMapper(
		context.Background(), testRealmName, testScopeID, testMapperID,
		groupsMapperRepr(map[string]string{"claim.name": "groups", "full.path": "true"}),
	).Return(nil, nil)

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestSyncProtocolMappers_Serve_UpdateMapperError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := groupsMapperScope(map[string]string{"claim.name": "changed"})

	mockScopes.EXPECT().GetClientScopeProtocolMappers(
		context.Background(), testRealmName, testScopeID,
	).Return([]keycloakapi.ProtocolMapperRepresentation{
		groupsMapperRepr(map[string]string{"claim.name": "groups"}),
	}, nil, nil)

	mockScopes.EXPECT().UpdateClientScopeProtocolMapper(
		context.Background(), testRealmName, testScopeID, testMapperID,
		mock.Anything,
	).Return(nil, errors.New("update error"))

	h := NewSyncProtocolMappers(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update protocol mapper")
}

func TestSyncProtocolMappers_Serve_CreateMapperError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

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
