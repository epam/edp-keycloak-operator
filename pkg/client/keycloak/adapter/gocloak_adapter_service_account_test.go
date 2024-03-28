package adapter

import (
	"testing"

	"github.com/Nerzal/gocloak/v12"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter/mocks"
)

func TestGoCloakAdapter_SetServiceAccountAttributes(t *testing.T) {
	mockClient := mocks.NewMockGoCloak(t)

	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	usr1 := gocloak.User{
		Username: gocloak.StringP("user1"),
		Attributes: &map[string][]string{
			"foo1": {"bar1"},
		},
	}

	usr2 := gocloak.User{
		Username: gocloak.StringP("user1"),
		Attributes: &map[string][]string{
			"foo":  {"bar"},
			"foo1": {"bar1"},
		},
	}

	mockClient.On("GetClientServiceAccount", mock.Anything, "token", "realm1", "clientID1").Return(&usr1, nil)
	mockClient.On("UpdateUser", mock.Anything, "token", "realm1", usr2).Return(nil)

	err := adapter.SetServiceAccountAttributes("realm1", "clientID1",
		map[string]string{"foo": "bar"}, true)
	require.NoError(t, err)
}
