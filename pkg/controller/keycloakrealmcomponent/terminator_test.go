package keycloakrealmcomponent

import (
	"context"
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator_DeleteResource(t *testing.T) {
	var (
		kcAdapter adapter.Mock
		logger    mock.Logger
	)

	kcAdapter.On("DeleteComponent", "foo", "bar").Return(nil)
	term := makeTerminator("foo", "bar", &kcAdapter, &logger)
	if err := term.DeleteResource(context.Background()); err != nil {
		t.Fatal(err)
	}

	if term.GetLogger() != &logger {
		t.Fatal("wrong logger returned")
	}
}
