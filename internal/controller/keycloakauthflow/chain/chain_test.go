package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/mocks"
	"github.com/epam/edp-keycloak-operator/pkg/destination"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
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
	kc := &keycloakapi.KeycloakClient{
		AuthFlows: mocks.NewMockAuthFlowsClient(t),
	}

	secretRef, err := secretref.NewSecretRef(fake.NewClientBuilder().Build(), destination.AllowAll())
	require.NoError(t, err)

	ch := MakeChain(kc, secretRef)
	require.NotNil(t, ch)
	require.Len(t, ch.handlers, 2)

	sync, ok := ch.handlers[1].(*SyncAuthFlowExecutions)
	require.True(t, ok)
	assert.Same(t, secretRef, sync.secretRef)
}
