package keycloakrealmrole

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator(t *testing.T) {
	lg := mock.NewLogr()
	kClient := new(adapter.Mock)

	term := makeTerminator("foo", "bar", kClient, lg)
	kClient.On("DeleteRealmRole", "foo", "bar").Return(nil).Once()

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteRealmRole", "foo", "bar").Return(errors.New("fatal")).Once()

	err = term.DeleteResource(context.Background())
	require.Error(t, err)

	loggerSink, ok := lg.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")

	assert.NotEmpty(t, loggerSink.InfoMessages(), "no info messages logged")
}
