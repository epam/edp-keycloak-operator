package keycloakrealm

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator(t *testing.T) {
	lg := mock.Logger{}
	kClient := new(adapter.Mock)

	term := makeTerminator("realm", kClient, &lg)

	if term.GetLogger() != &lg {
		t.Fatal("wrong logger set")
	}

	kClient.On("DeleteRealm", "realm").Return(nil).Once()
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteRealm", "realm").Return(errors.New("fatal")).Once()
	err = term.DeleteResource(context.Background())
	require.Error(t, err)

	if len(lg.InfoMessages) == 0 {
		t.Fatal("no info messages logged")
	}
}
