package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
	v2mocks "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/mocks"
)

func TestAuthFlow_ServeRequest(t *testing.T) {
	af := AuthFlow{}

	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm1",
		},
	}

	ctx := context.Background()

	err := af.ServeRequest(ctx, &realm, nil)
	require.NoError(t, err)

	mockRealm := v2mocks.NewMockRealmClient(t)
	mockRealm.On("SetRealmBrowserFlow", mock.Anything, "realm1", "flow-alias-1").Return(nil, nil)

	realm.Spec.BrowserFlow = ptr.To("flow-alias-1")

	kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}
	err = af.ServeRequest(ctx, &realm, kClientV2)
	require.NoError(t, err)
}

func TestAuthFlow_ServeRequest_Failure(t *testing.T) {
	mockRealm := v2mocks.NewMockRealmClient(t)
	af := AuthFlow{}

	realm := keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm1",
		},
	}

	mockErr := errors.New("fatal")

	mockRealm.On("SetRealmBrowserFlow", mock.Anything, "realm1", "flow-alias-1").Return(nil, mockErr)

	realm.Spec.BrowserFlow = ptr.To("flow-alias-1")

	kClientV2 := &keycloakv2.KeycloakClient{Realms: mockRealm}

	err := af.ServeRequest(context.Background(), &realm, kClientV2)
	if err == nil {
		t.Fatal("no error on mock fatal")
	}

	assert.ErrorIs(t, err, mockErr)
}
