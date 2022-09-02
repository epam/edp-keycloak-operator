package keycloakrealm

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator(t *testing.T) {
	lg := mock.NewLogr()
	kClient := new(adapter.Mock)

	term := makeTerminator("realm", kClient, lg)

	kClient.On("DeleteRealm", "realm").Return(nil).Once()

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteRealm", "realm").Return(errors.New("fatal")).Once()

	err = term.DeleteResource(context.Background())
	require.Error(t, err)

	loggerSink, ok := lg.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	assert.NotEmpty(t, loggerSink.InfoMessages(), "no info messages logged")
}
