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
	testScopeName    = "test-scope"
	testScopeID      = "scope-id-123"
	testRealmName    = "test-realm"
	testProtocolOIDC = "openid-connect"
)

func TestCreateOrUpdateScope_Serve_CreateNew(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakv2.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC
	scope.Spec.Description = "Test description"
	scope.Spec.Attributes = map[string]string{"key": "val"}

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakv2.ClientScopeRepresentation{}, nil, nil)

	desc := "Test description"
	protocol := testProtocolOIDC
	attrs := map[string]string{"key": "val"}

	mockScopes.EXPECT().CreateClientScope(
		context.Background(), testRealmName,
		keycloakv2.ClientScopeRepresentation{
			Name:        ptr.To(testScopeName),
			Protocol:    &protocol,
			Description: &desc,
			Attributes:  &attrs,
		},
	).Return(&keycloakv2.Response{
		HTTPResponse: &http.Response{
			Header: http.Header{
				"Location": []string{"http://localhost/admin/realms/test-realm/client-scopes/scope-id-123"},
			},
		},
	}, nil)

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testScopeID, scope.Status.ID)
}

func TestCreateOrUpdateScope_Serve_UpdateExisting(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakv2.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC
	scope.Spec.Description = "Updated description"

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakv2.ClientScopeRepresentation{
		{
			Id:   ptr.To(testScopeID),
			Name: ptr.To(testScopeName),
		},
	}, nil, nil)

	desc := "Updated description"
	protocol := testProtocolOIDC

	var nilAttrs map[string]string

	mockScopes.EXPECT().UpdateClientScope(
		context.Background(), testRealmName, testScopeID,
		keycloakv2.ClientScopeRepresentation{
			Name:        ptr.To(testScopeName),
			Protocol:    &protocol,
			Description: &desc,
			Attributes:  &nilAttrs,
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testScopeID, scope.Status.ID)
}

func TestCreateOrUpdateScope_Serve_GetScopesError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakv2.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return(nil, nil, errors.New("api error"))

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find client scope by name")
}

func TestCreateOrUpdateScope_Serve_CreateError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakv2.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakv2.ClientScopeRepresentation{}, nil, nil)

	var nilAttrs map[string]string

	protocol := testProtocolOIDC

	mockScopes.EXPECT().CreateClientScope(
		context.Background(), testRealmName,
		keycloakv2.ClientScopeRepresentation{
			Name:        ptr.To(testScopeName),
			Protocol:    &protocol,
			Description: ptr.To(""),
			Attributes:  &nilAttrs,
		},
	).Return(nil, errors.New("create error"))

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create client scope")
}

func TestCreateOrUpdateScope_Serve_UpdateError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakv2.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakv2.ClientScopeRepresentation{
		{
			Id:   ptr.To(testScopeID),
			Name: ptr.To(testScopeName),
		},
	}, nil, nil)

	var nilAttrs map[string]string

	protocol := testProtocolOIDC

	mockScopes.EXPECT().UpdateClientScope(
		context.Background(), testRealmName, testScopeID,
		keycloakv2.ClientScopeRepresentation{
			Name:        ptr.To(testScopeName),
			Protocol:    &protocol,
			Description: ptr.To(""),
			Attributes:  &nilAttrs,
		},
	).Return(nil, errors.New("update error"))

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update client scope")
}
