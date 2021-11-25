package keycloakrealmidentityprovider

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator_DeleteResource(t *testing.T) {
	kClient := adapter.Mock{}
	l := mock.Logger{}

	kClient.On("DeleteIdentityProvider", "realm", "alias1").Return(nil).Once()
	term := makeTerminator("realm", "alias1", &kClient, &l)
	if err := term.DeleteResource(context.Background()); err != nil {
		t.Fatal(err)
	}

	kClient.On("DeleteIdentityProvider", "realm", "alias1").
		Return(errors.New("delete res fatal")).Once()
	err := term.DeleteResource(context.Background())
	if err == nil {
		t.Fatal("no error returned")
	}

	if err.Error() != "unable to delete realm idp: delete res fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	if term.GetLogger() != &l {
		t.Fatal("wrong logger returned")
	}
}
