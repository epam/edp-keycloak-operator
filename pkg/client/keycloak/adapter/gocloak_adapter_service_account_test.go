package adapter

import (
	"testing"

	"github.com/Nerzal/gocloak/v8"
)

func TestGoCloakAdapter_SetServiceAccountAttributes(t *testing.T) {
	mockClient := new(MockGoCloakClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	usr1 := gocloak.User{
		Username: gocloak.StringP("user1"),
		Attributes: &map[string][]string{
			"foo1": []string{"bar1"},
		},
	}

	usr2 := gocloak.User{
		Username: gocloak.StringP("user1"),
		Attributes: &map[string][]string{
			"foo": {"bar"},
		},
	}

	mockClient.On("GetClientServiceAccount", "realm1", "clientID1").Return(&usr1, nil)
	mockClient.On("UpdateUser", "realm1", usr2).Return(nil)

	if err := adapter.SetServiceAccountAttributes("realm1", "clientID1",
		map[string]string{"foo": "bar"}); err != nil {
		t.Fatal(err)
	}
}
