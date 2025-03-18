package keycloakrealmcomponent

import (
	"context"
	"testing"

	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestTerminator_DeleteResource(t *testing.T) {
	kcAdapter := mocks.NewMockClient(t)

	kcAdapter.On("DeleteComponent", testifymock.Anything, "foo", "bar").Return(nil)
	term := makeTerminator("foo", "bar", kcAdapter, false)
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)
}
