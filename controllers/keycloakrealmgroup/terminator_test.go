package keycloakrealmgroup

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator(t *testing.T) {
	lg := mock.NewLogr()
	kClient := new(adapter.Mock)

	term := makeTerminator(kClient, "foo", "bar", false)

	kClient.On("DeleteGroup", "foo", "bar").Return(nil).Once()

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteGroup", "foo", "bar").Return(errors.New("fatal")).Once()

	if err := term.DeleteResource(ctrl.LoggerInto(context.Background(), lg)); err == nil {
		t.Fatal("no error returned")
	}

	loggerSink, ok := lg.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")
	assert.NotEmpty(t, loggerSink.InfoMessages(), "no info messages logged")
}

func TestTerminatorSkipDeletion(t *testing.T) {
	term := makeTerminator(
		nil,
		"realm",
		"realmCR",
		true,
	)

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)
}
