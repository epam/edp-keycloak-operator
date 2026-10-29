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
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

const (
	testScopeName        = "test-scope"
	testScopeID          = "scope-id-123"
	testRealmName        = "test-realm"
	testProtocolOIDC     = "openid-connect"
	testScopeDescription = "Test description"
)

func TestCreateOrUpdateScope_Serve_CreateNew(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC
	scope.Spec.Description = testScopeDescription
	scope.Spec.Attributes = map[string]string{"key": "val", "gui.order": ""}

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakapi.ClientScopeRepresentation{}, nil, nil)

	desc := testScopeDescription
	protocol := testProtocolOIDC
	attrs := map[string]string{"key": "val", "gui.order": ""}

	mockScopes.EXPECT().CreateClientScope(
		context.Background(), testRealmName,
		keycloakapi.ClientScopeRepresentation{
			Name:        ptr.To(testScopeName),
			Protocol:    &protocol,
			Description: &desc,
			Attributes:  &attrs,
		},
	).Return(&keycloakapi.Response{
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
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC
	scope.Spec.Description = "Updated description"

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakapi.ClientScopeRepresentation{
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
		keycloakapi.ClientScopeRepresentation{
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

func TestCreateOrUpdateScope_Serve_SkipUpdateWhenInSync(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC
	scope.Spec.Description = testScopeDescription
	scope.Spec.Attributes = map[string]string{"key": "val"}

	// Server-added default attributes must not trigger an update.
	existingAttrs := map[string]string{"key": "val", "display.on.consent.screen": "true"}

	// Existing state matches spec: no UpdateClientScope call expected.
	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakapi.ClientScopeRepresentation{
		{
			Id:          ptr.To(testScopeID),
			Name:        ptr.To(testScopeName),
			Protocol:    ptr.To(testProtocolOIDC),
			Description: ptr.To(testScopeDescription),
			Attributes:  &existingAttrs,
		},
	}, nil, nil)

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testScopeID, scope.Status.ID)
}

func TestCreateOrUpdateScope_Serve_ForceUpdateOnSpecChange(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	// Generation ahead of ObservedGeneration: PUT must fire even though the
	// declared attributes match, so keys removed from the spec are dropped.
	scope := &keycloakApi.KeycloakClientScope{}
	scope.Generation = 2
	scope.Status.ObservedGeneration = 1
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC
	scope.Spec.Description = testScopeDescription
	scope.Spec.Attributes = map[string]string{"key": "val", "gui.order": ""}

	existingAttrs := map[string]string{"key": "val", "removed.key": "stale"}

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakapi.ClientScopeRepresentation{
		{
			Id:          ptr.To(testScopeID),
			Name:        ptr.To(testScopeName),
			Protocol:    ptr.To(testProtocolOIDC),
			Description: ptr.To(testScopeDescription),
			Attributes:  &existingAttrs,
		},
	}, nil, nil)

	desc := testScopeDescription
	protocol := testProtocolOIDC
	attrs := map[string]string{"key": "val", "gui.order": ""}

	mockScopes.EXPECT().UpdateClientScope(
		context.Background(), testRealmName, testScopeID,
		keycloakapi.ClientScopeRepresentation{
			Name:        ptr.To(testScopeName),
			Protocol:    &protocol,
			Description: &desc,
			Attributes:  &attrs,
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testScopeID, scope.Status.ID)
}

func TestCreateOrUpdateScope_Serve_GetScopesError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

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
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakapi.ClientScopeRepresentation{}, nil, nil)

	var nilAttrs map[string]string

	protocol := testProtocolOIDC

	mockScopes.EXPECT().CreateClientScope(
		context.Background(), testRealmName,
		keycloakapi.ClientScopeRepresentation{
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

func TestCreateOrUpdateScope_Serve_CreateWithoutLocationHeader(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakapi.ClientScopeRepresentation{}, nil, nil)

	var nilAttrs map[string]string

	protocol := testProtocolOIDC

	// Keycloak reports the new id in the Location header only. Without it status.ID would
	// stay empty and every later update would address an empty scope id.
	mockScopes.EXPECT().CreateClientScope(
		context.Background(), testRealmName,
		keycloakapi.ClientScopeRepresentation{
			Name:        ptr.To(testScopeName),
			Protocol:    &protocol,
			Description: ptr.To(""),
			Attributes:  &nilAttrs,
		},
	).Return(&keycloakapi.Response{HTTPResponse: &http.Response{Header: http.Header{}}}, nil)

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Location header missing or empty")
	assert.Empty(t, scope.Status.ID)
}

func TestCreateOrUpdateScope_Serve_UpdateError(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakapi.ClientScopeRepresentation{
		{
			Id:   ptr.To(testScopeID),
			Name: ptr.To(testScopeName),
		},
	}, nil, nil)

	var nilAttrs map[string]string

	protocol := testProtocolOIDC

	mockScopes.EXPECT().UpdateClientScope(
		context.Background(), testRealmName, testScopeID,
		keycloakapi.ClientScopeRepresentation{
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

func TestCreateOrUpdateScope_Serve_SkipUpdateWhenBlankAttributeAbsent(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Generation = 1
	scope.Status.ObservedGeneration = 1
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC
	scope.Spec.Description = testScopeDescription
	scope.Spec.Attributes = map[string]string{
		"include.in.token.scope": "true",
		"gui.order":              "",
		"consent.screen.text":    " ",
	}

	existingAttrs := map[string]string{"include.in.token.scope": "true"}

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakapi.ClientScopeRepresentation{
		{
			Id:          ptr.To(testScopeID),
			Name:        ptr.To(testScopeName),
			Protocol:    ptr.To(testProtocolOIDC),
			Description: ptr.To(testScopeDescription),
			Attributes:  &existingAttrs,
		},
	}, nil, nil)

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
	assert.Equal(t, testScopeID, scope.Status.ID)
}

func TestCreateOrUpdateScope_Serve_UpdateWhenBlankAttributeStale(t *testing.T) {
	mockScopes := mocks.NewMockClientScopesClient(t)
	kClient := &keycloakapi.KeycloakClient{ClientScopes: mockScopes}

	scope := &keycloakApi.KeycloakClientScope{}
	scope.Generation = 1
	scope.Status.ObservedGeneration = 1
	scope.Spec.Name = testScopeName
	scope.Spec.Protocol = testProtocolOIDC
	scope.Spec.Description = testScopeDescription
	scope.Spec.Attributes = map[string]string{"gui.order": ""}

	existingAttrs := map[string]string{"gui.order": "5"}

	mockScopes.EXPECT().GetClientScopes(
		context.Background(), testRealmName,
	).Return([]keycloakapi.ClientScopeRepresentation{
		{
			Id:          ptr.To(testScopeID),
			Name:        ptr.To(testScopeName),
			Protocol:    ptr.To(testProtocolOIDC),
			Description: ptr.To(testScopeDescription),
			Attributes:  &existingAttrs,
		},
	}, nil, nil)

	desc := testScopeDescription
	protocol := testProtocolOIDC
	attrs := map[string]string{"gui.order": ""}

	mockScopes.EXPECT().UpdateClientScope(
		context.Background(), testRealmName, testScopeID,
		keycloakapi.ClientScopeRepresentation{
			Name:        ptr.To(testScopeName),
			Protocol:    &protocol,
			Description: &desc,
			Attributes:  &attrs,
		},
	).Return(nil, nil)

	h := NewCreateOrUpdateScope(kClient)
	err := h.Serve(context.Background(), scope, testRealmName)
	require.NoError(t, err)
}

func TestClientScopeAttributesMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		existing map[string]string
		spec     map[string]string
		want     bool
	}{
		{name: "empty value, key absent", existing: map[string]string{"x": "1"}, spec: map[string]string{"k": ""}, want: true},
		{name: "space value, key absent", existing: map[string]string{"x": "1"}, spec: map[string]string{"k": " "}, want: true},
		{name: "control chars only, key absent", existing: map[string]string{"x": "1"}, spec: map[string]string{"k": "\x01"}, want: true},
		{name: "non-breaking space, key absent", existing: map[string]string{"x": "1"}, spec: map[string]string{"k": "\u00a0"}, want: false},
		{name: "empty value, stored empty", existing: map[string]string{"k": ""}, spec: map[string]string{"k": ""}, want: true},
		{name: "space value, stored space", existing: map[string]string{"k": " "}, spec: map[string]string{"k": " "}, want: true},
		{name: "empty value, stored non-blank", existing: map[string]string{"k": "x"}, spec: map[string]string{"k": ""}, want: false},
		{name: "padded blank value, stored empty", existing: map[string]string{"k": ""}, spec: map[string]string{"k": "  "}, want: false},
		{name: "non-blank value, key absent", existing: map[string]string{"x": "1"}, spec: map[string]string{"k": "a"}, want: false},
		{name: "non-blank value, different", existing: map[string]string{"k": "b"}, spec: map[string]string{"k": "a"}, want: false},
		{name: "non-blank value, equal", existing: map[string]string{"k": "a"}, spec: map[string]string{"k": "a"}, want: true},
		{name: "padded non-blank value, stored as is", existing: map[string]string{"k": " a "}, spec: map[string]string{"k": " a "}, want: true},
		{name: "nil spec", existing: map[string]string{"k": "a"}, spec: nil, want: true},
		{name: "empty spec", existing: map[string]string{"k": "a"}, spec: map[string]string{}, want: true},
		{name: "non-breaking space, stored as is", existing: map[string]string{"k": "\u00a0"}, spec: map[string]string{"k": "\u00a0"}, want: true},
		{name: "nil existing, blank value", existing: nil, spec: map[string]string{"k": ""}, want: true},
		{name: "nil existing, non-blank value", existing: nil, spec: map[string]string{"k": "a"}, want: false},
		{
			name:     "mix, blank absent and non-blank equal",
			existing: map[string]string{"b": "1"},
			spec:     map[string]string{"a": "", "b": "1"},
			want:     true,
		},
		{
			name:     "mix, blank absent and non-blank different",
			existing: map[string]string{"b": "2"},
			spec:     map[string]string{"a": "", "b": "1"},
			want:     false,
		},
		{
			name:     "server extra keys ignored",
			existing: map[string]string{"k": "a", "display.on.consent.screen": "true"},
			spec:     map[string]string{"k": "a"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, clientScopeAttributesMatch(tt.existing, tt.spec))
		})
	}
}

func TestIsJavaTrimBlank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "empty", input: "", want: true},
		{name: "ascii whitespace", input: " \t\n\r\f\u000B", want: true},
		{name: "control characters", input: "\u0000\u0001\u001F", want: true},
		{name: "text with padding", input: " x ", want: false},
		{name: "non-breaking space", input: "\u00A0", want: false},
		{name: "next line", input: "\u0085", want: false},
		{name: "line separator", input: "\u2028", want: false},
		{name: "ideographic space", input: "\u3000", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, isJavaTrimBlank(tt.input))
		})
	}
}
