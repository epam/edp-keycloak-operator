package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestAuthFlow_ServeRequest(t *testing.T) {
	kc := mocks.NewMockClient(t)
	af := AuthFlow{}

	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm1",
		},
	}

	ctx := context.Background()

	err := af.ServeRequest(ctx, &realm, kc)
	require.NoError(t, err)

	kc.On("SetRealmBrowserFlow", mock.Anything, "realm1", "flow-alias-1").Return(nil)

	realm.Spec.BrowserFlow = gocloak.StringP("flow-alias-1")

	err = af.ServeRequest(ctx, &realm, kc)
	require.NoError(t, err)
}

func TestAuthFlow_ServeRequest_Failure(t *testing.T) {
	kc := mocks.NewMockClient(t)
	af := AuthFlow{}

	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm1",
		},
	}

	mockErr := errors.New("fatal")

	kc.On("SetRealmBrowserFlow", mock.Anything, "realm1", "flow-alias-1").Return(mockErr)

	realm.Spec.BrowserFlow = gocloak.StringP("flow-alias-1")

	err := af.ServeRequest(context.Background(), &realm, kc)
	if err == nil {
		t.Fatal("no error on mock fatal")
	}

	assert.ErrorIs(t, err, mockErr)
}
