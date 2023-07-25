package keycloakclientscope

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator_DeleteResource(t *testing.T) {
	logger := mock.NewLogr()
	kClient := new(adapter.Mock)
	kClient.On("DeleteClientScope", "foo", "bar").Return(nil).Once()
	term := makeTerminator(kClient, "foo", "bar")
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteClientScope", "foo", "bar").Return(errors.New("fatal")).Once()

	err = term.DeleteResource(ctrl.LoggerInto(context.Background(), logger))
	require.Error(t, err)

	if !strings.Contains(err.Error(), "failed to delete client scope") {
		t.Fatalf("wrong error logged: %s", err.Error())
	}
}
