package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

const testRealmName = "test-realm"

type mockHandler struct {
	returnErr error
	called    bool
}

func (m *mockHandler) Serve(_ context.Context, _ *keycloakApi.KeycloakAuthFlow, _ string) error {
	m.called = true

	return m.returnErr
}

func TestChain_Serve_EmptyChain(t *testing.T) {
	ch := &Chain{}
	err := ch.Serve(context.Background(), &keycloakApi.KeycloakAuthFlow{}, testRealmName)
	require.NoError(t, err)
}

func TestChain_Serve_SingleHandler(t *testing.T) {
	h := &mockHandler{}
	ch := &Chain{}
	ch.Use(h)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakAuthFlow{}, testRealmName)
	require.NoError(t, err)
	assert.True(t, h.called)
}

func TestChain_Serve_MultipleHandlers(t *testing.T) {
	h1 := &mockHandler{}
	h2 := &mockHandler{}
	ch := &Chain{}
	ch.Use(h1, h2)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakAuthFlow{}, testRealmName)
	require.NoError(t, err)
	assert.True(t, h1.called)
	assert.True(t, h2.called)
}

func TestChain_Serve_ErrorStopsChain(t *testing.T) {
	h1 := &mockHandler{returnErr: errors.New("handler error")}
	h2 := &mockHandler{}
	ch := &Chain{}
	ch.Use(h1, h2)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakAuthFlow{}, testRealmName)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to serve handler")
	assert.True(t, h1.called)
	assert.False(t, h2.called)
}

func TestMakeChain_Creates2Handlers(t *testing.T) {
	kc := &keycloakv2.KeycloakClient{
		AuthFlows: mocks.NewMockAuthFlowsClient(t),
	}

	ch := MakeChain(kc)
	require.NotNil(t, ch)
	require.Len(t, ch.handlers, 2)
}
