package keycloakauthflow

import (
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
	"github.com/pkg/errors"
)

func TestTerminator(t *testing.T) {
	lg := mock.Logger{}
	kClient := new(adapter.Mock)

	term := makeTerminator("foo", "bar", kClient, &lg)

	if term.GetLogger() != &lg {
		t.Fatal("wrong logger set")
	}

	kClient.On("DeleteAuthFlow", "foo", "bar").Return(nil).Once()
	if err := term.DeleteResource(); err != nil {
		t.Fatal(err)
	}

	kClient.On("DeleteAuthFlow", "foo", "bar").Return(errors.New("fatal")).Once()
	if err := term.DeleteResource(); err == nil {
		t.Fatal("no error returned")
	}

	if len(lg.InfoMessages) == 0 {
		t.Fatal("no info messages logged")
	}
}
