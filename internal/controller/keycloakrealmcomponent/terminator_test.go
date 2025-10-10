package keycloakrealmcomponent

import (
	"context"
	"testing"

	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestTerminator_DeleteResource(t *testing.T) {
	kcAdapter := mocks.NewMockClient(t)

	kcAdapter.On("DeleteComponent", testifymock.Anything, "foo", "bar").Return(nil)
	term := makeTerminator("foo", "bar", kcAdapter, false)
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)
}

func TestTerminatorDeleteResourceNotFound(t *testing.T) {
	kClient := mocks.NewMockClient(t)
	kClient.On("DeleteComponent", testifymock.Anything, "realm", "component").Return(adapter.NotFoundError("not found")).Once()

	term := makeTerminator("realm", "component", kClient, false)

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)
}
