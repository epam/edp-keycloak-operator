package keycloakclient

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestTerminator(t *testing.T) {
	kClient := new(adapter.Mock)

	term := makeTerminator("realm", "client", kClient)

	kClient.On("DeleteClient", "realm", "client").Return(nil).Once()

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteClient", "realm", "client").Return(errors.New("fatal")).Once()

	if err := term.DeleteResource(context.Background()); err == nil {
		t.Fatal("no error returned")
	}
}
