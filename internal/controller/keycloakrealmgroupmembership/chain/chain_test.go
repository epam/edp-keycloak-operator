package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type mockHandler struct {
	returnErr error
	called    bool
}

func (m *mockHandler) Serve(
	_ context.Context,
	_ *keycloakApi.KeycloakRealmGroupMembership,
	_ string,
	_ *MembershipContext,
) error {
	m.called = true

	return m.returnErr
}

func TestChain_Serve(t *testing.T) {
	h1 := &mockHandler{}
	h2 := &mockHandler{}

	ch := &Chain{}
	ch.Use(h1, h2)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmGroupMembership{}, testRealm, &MembershipContext{})
	require.NoError(t, err)
	assert.True(t, h1.called)
	assert.True(t, h2.called)
}

func TestChain_Serve_HandlerError_StopsChain(t *testing.T) {
	h1 := &mockHandler{returnErr: errors.New("handler error")}
	h2 := &mockHandler{}

	ch := &Chain{}
	ch.Use(h1, h2)

	err := ch.Serve(context.Background(), &keycloakApi.KeycloakRealmGroupMembership{}, testRealm, &MembershipContext{})
	require.ErrorContains(t, err, "handler error")
	assert.True(t, h1.called)
	assert.False(t, h2.called)
}

func TestMakeChain(t *testing.T) {
	ch := MakeChain(&keycloakapi.KeycloakClient{})
	require.Len(t, ch.handlers, 3)
}
