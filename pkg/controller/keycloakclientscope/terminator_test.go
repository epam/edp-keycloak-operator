package keycloakclientscope

import (
	"context"
	"strings"
	"testing"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/pkg/errors"
)

func TestTerminator_DeleteResource(t *testing.T) {
	logger := mock.Logger{}
	kClient := new(adapter.Mock)
	kClient.On("DeleteClientScope", "foo", "bar").Return(nil).Once()
	term := makeTerminator(kClient, "foo", "bar", &logger)
	if err := term.DeleteResource(context.Background()); err != nil {
		t.Fatal(err)
	}

	kClient.On("DeleteClientScope", "foo", "bar").Return(errors.New("fatal")).Once()
	err := term.DeleteResource(context.Background())
	if err == nil {
		t.Fatal("no error returned")
	}

	if !strings.Contains(err.Error(), "unable to delete client scope") {
		t.Fatalf("wrong error logged: %s", err.Error())
	}
}
