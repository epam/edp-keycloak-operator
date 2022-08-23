package chain

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v10"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestAuthFlow_ServeRequest(t *testing.T) {
	kc := adapter.Mock{}
	af := AuthFlow{}

	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm1",
		},
	}

	ctx := context.Background()

	err := af.ServeRequest(ctx, &realm, &kc)
	require.NoError(t, err)

	kc.On("SetRealmBrowserFlow", "realm1", "flow-alias-1").Return(nil)
	realm.Spec.BrowserFlow = gocloak.StringP("flow-alias-1")
	err = af.ServeRequest(ctx, &realm, &kc)
	require.NoError(t, err)
}

func TestAuthFlow_ServeRequest_Failure(t *testing.T) {
	kc := adapter.Mock{}
	af := AuthFlow{}

	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm1",
		},
	}

	mockErr := errors.New("fatal")

	kc.On("SetRealmBrowserFlow", "realm1", "flow-alias-1").Return(mockErr)
	realm.Spec.BrowserFlow = gocloak.StringP("flow-alias-1")
	err := af.ServeRequest(context.Background(), &realm, &kc)
	if err == nil {
		t.Fatal("no error on mock fatal")
	}

	assert.ErrorIs(t, err, mockErr)
}
