package keycloakrealmidentityprovider

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mock"
)

func TestTerminator_DeleteResource(t *testing.T) {
	kClient := adapter.Mock{}
	l := mock.NewLogr()

	kClient.On("DeleteIdentityProvider", "realm", "alias1").Return(nil).Once()
	term := makeTerminator("realm", "alias1", &kClient, l)
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteIdentityProvider", "realm", "alias1").
		Return(errors.New("delete res fatal")).Once()

	err = term.DeleteResource(context.Background())
	require.Error(t, err)

	if err.Error() != "unable to delete realm idp: delete res fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
