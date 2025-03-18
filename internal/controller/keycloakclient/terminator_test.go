package keycloakclient

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestTerminator(t *testing.T) {
	kClient := mocks.NewMockClient(t)

	term := makeTerminator("realm", "client", kClient, false)

	kClient.On("DeleteClient", mock.Anything, "realm", "client").Return(nil).Once()

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteClient", mock.Anything, "realm", "client").Return(errors.New("fatal")).Once()

	if err := term.DeleteResource(context.Background()); err == nil {
		t.Fatal("no error returned")
	}
}

func TestTerminatorSkipDeletion(t *testing.T) {
	term := makeTerminator(
		"realm",
		"client",
		nil,
		true,
	)

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)
}
