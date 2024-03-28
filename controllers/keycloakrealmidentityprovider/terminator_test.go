package keycloakrealmidentityprovider

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestTerminator_DeleteResource(t *testing.T) {
	kClient := mocks.NewMockClient(t)

	kClient.On("DeleteIdentityProvider", testifymock.Anything, "realm", "alias1").Return(nil).Once()
	term := makeTerminator("realm", "alias1", kClient, false)
	err := term.DeleteResource(context.Background())
	require.NoError(t, err)

	kClient.On("DeleteIdentityProvider", testifymock.Anything, "realm", "alias1").
		Return(errors.New("delete res fatal")).Once()

	err = term.DeleteResource(context.Background())
	require.Error(t, err)

	assert.Contains(t, err.Error(), "delete res fatal")
}
