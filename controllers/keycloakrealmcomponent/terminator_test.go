package keycloakrealmcomponent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator_DeleteResource(t *testing.T) {
	var (
		kcAdapter adapter.Mock
	)

	kcAdapter.On("DeleteComponent", "foo", "bar").Return(nil)
	term := makeTerminator("foo", "bar", &kcAdapter, mock.NewLogr())
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)
}
