package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	keycloakapimocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
)

type mockHandler struct {
	called bool
	err    error
}

func (m *mockHandler) Serve(_ context.Context, _ *keycloakApi.KeycloakRealmComponent, _ string) error {
	m.called = true
	return m.err
}

func TestChain_Serve_EmptyChain(t *testing.T) {
	ch := &Chain{}
	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmComponent{}, "realm")
	require.NoError(t, err)
}

func TestChain_Serve_SingleHandler(t *testing.T) {
	h := &mockHandler{}
	ch := &Chain{}
	ch.Use(h)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmComponent{}, "realm")
	require.NoError(t, err)
	assert.True(t, h.called)
}

func TestChain_Serve_MultipleHandlers(t *testing.T) {
	h1 := &mockHandler{}
	h2 := &mockHandler{}
	ch := &Chain{}
	ch.Use(h1, h2)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmComponent{}, "realm")
	require.NoError(t, err)
	assert.True(t, h1.called)
	assert.True(t, h2.called)
}

func TestChain_Serve_ErrorStopsChain(t *testing.T) {
	h1 := &mockHandler{err: errors.New("handler error")}
	h2 := &mockHandler{}
	ch := &Chain{}
	ch.Use(h1, h2)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmComponent{}, "realm")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "handler error")
	assert.True(t, h1.called)
	assert.False(t, h2.called)
}

func TestMakeChain_NotNil(t *testing.T) {
	mockComponents := keycloakapimocks.NewMockRealmComponentsClient(t)
	kClient := &keycloakapi.KeycloakClient{RealmComponents: mockComponents}

	ch := MakeChain(nil, kClient, &mockSecretRefClient{})
	assert.NotNil(t, ch)
	assert.Len(t, ch.handlers, 1)
}

type mockSecretRefClient struct {
	err error
}

func (m *mockSecretRefClient) MapComponentConfigSecretsRefs(_ context.Context, _ map[string][]string, _ string) error {
	return m.err
}
