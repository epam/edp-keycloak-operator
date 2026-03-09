package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

type mockHandler struct {
	returnErr error
	called    bool
}

func (m *mockHandler) Serve(_ context.Context, _ *keycloakApi.KeycloakRealmRole, _ string, _ *RoleContext) error {
	m.called = true

	return m.returnErr
}

func TestChain_Serve_Empty(t *testing.T) {
	ch := &Chain{}
	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmRole{}, "test-realm", &RoleContext{})
	require.NoError(t, err)
}

func TestChain_Serve_SingleHandler_Success(t *testing.T) {
	h := &mockHandler{}
	ch := &Chain{}
	ch.Use(h)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmRole{}, "test-realm", &RoleContext{})
	require.NoError(t, err)
	assert.True(t, h.called)
}

func TestChain_Serve_MultipleHandlers_Success(t *testing.T) {
	h1 := &mockHandler{}
	h2 := &mockHandler{}
	h3 := &mockHandler{}

	ch := &Chain{}
	ch.Use(h1, h2, h3)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmRole{}, "test-realm", &RoleContext{})
	require.NoError(t, err)
	assert.True(t, h1.called)
	assert.True(t, h2.called)
	assert.True(t, h3.called)
}

func TestChain_Serve_HandlerError_StopsChain(t *testing.T) {
	h1 := &mockHandler{returnErr: errors.New("handler error")}
	h2 := &mockHandler{}

	ch := &Chain{}
	ch.Use(h1, h2)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmRole{}, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "handler error")
	assert.False(t, h2.called)
}

func TestChain_Serve_MiddleHandlerError(t *testing.T) {
	h1 := &mockHandler{}
	h2 := &mockHandler{returnErr: errors.New("middle error")}
	h3 := &mockHandler{}

	ch := &Chain{}
	ch.Use(h1, h2, h3)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmRole{}, "test-realm", &RoleContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "middle error")
	assert.True(t, h1.called)
	assert.False(t, h3.called)
}
