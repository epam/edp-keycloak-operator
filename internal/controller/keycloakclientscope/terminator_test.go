package keycloakclientscope

import (
	"context"
	"errors"
	"strings"
	"testing"

	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestTerminator_DeleteResource(t *testing.T) {
	logger := mock.NewLogr()
	kClient := mocks.NewMockClient(t)
	kClient.On("DeleteClientScope", testifymock.Anything, "foo", "bar").Return(nil).Once()
	term := makeTerminator(kClient, "foo", "bar", false)
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteClientScope", testifymock.Anything, "foo", "bar").Return(errors.New("fatal")).Once()

	err = term.DeleteResource(ctrl.LoggerInto(context.Background(), logger))
	require.Error(t, err)

	if !strings.Contains(err.Error(), "failed to delete client scope") {
		t.Fatalf("wrong error logged: %s", err.Error())
	}
}

func TestTerminatorSkipDeletion(t *testing.T) {
	term := makeTerminator(
		nil,
		"realm",
		"scope",
		true,
	)

	err := term.DeleteResource(context.Background())
	require.NoError(t, err)
}
