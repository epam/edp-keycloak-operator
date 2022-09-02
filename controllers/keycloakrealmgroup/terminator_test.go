package keycloakrealmgroup

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

	term := makeTerminator(kClient, "foo", "bar", lg)

	kClient.On("DeleteGroup", "foo", "bar").Return(nil).Once()

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteGroup", "foo", "bar").Return(errors.New("fatal")).Once()

	if err := term.DeleteResource(context.Background()); err == nil {
		t.Fatal("no error returned")
	}

	loggerSink, ok := lg.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	assert.NotEmpty(t, loggerSink.InfoMessages(), "no info messages logged")
}
