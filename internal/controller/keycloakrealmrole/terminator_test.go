package keycloakrealmrole

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestTerminator(t *testing.T) {
	lg := mock.NewLogr()
	kClient := mocks.NewMockClient(t)

	term := makeTerminator("foo", "bar", kClient, false)
	kClient.On("DeleteRealmRole", testifymock.Anything, "foo", "bar").Return(nil).Once()

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteRealmRole", testifymock.Anything, "foo", "bar").Return(errors.New("fatal")).Once()

	err = term.DeleteResource(ctrl.LoggerInto(context.Background(), lg))
	require.Error(t, err)

	loggerSink, ok := lg.GetSink().(*mock.Logger)
	require.True(t, ok, "wrong logger type")

	assert.NotEmpty(t, loggerSink.InfoMessages(), "no info messages logged")
}

func TestTerminatorSkipDeletion(t *testing.T) {
	term := makeTerminator(
		"realm",
		"role",
		nil,
		true,
	)

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)
}

func TestTerminatorDeleteResourceNotFound(t *testing.T) {
	kClient := mocks.NewMockClient(t)
	kClient.On("DeleteRealmRole", testifymock.Anything, "realm", "role").Return(adapter.NotFoundError("not found")).Once()

	term := makeTerminator("realm", "role", kClient, false)

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)
}
