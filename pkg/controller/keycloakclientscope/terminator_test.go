package keycloakclientscope

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator_DeleteResource(t *testing.T) {
	logger := mock.Logger{}
	kClient := new(adapter.Mock)
	kClient.On("DeleteClientScope", "foo", "bar").Return(nil).Once()
	term := makeTerminator(kClient, "foo", "bar", &logger)
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteClientScope", "foo", "bar").Return(errors.New("fatal")).Once()
	err = term.DeleteResource(context.Background())
	require.Error(t, err)

	if !strings.Contains(err.Error(), "unable to delete client scope") {
		t.Fatalf("wrong error logged: %s", err.Error())
	}
}
