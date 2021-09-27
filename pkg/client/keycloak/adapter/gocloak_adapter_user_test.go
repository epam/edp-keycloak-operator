package adapter

import (
	"testing"

	"github.com/Nerzal/gocloak/v8"
)

func TestGoCloakAdapter_SyncRealmUser(t *testing.T) {
	mockClient := new(MockGoCloakClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	usr := KeycloakUser{
		Username: "vasia",
		Attributes: map[string]string{
			"foo": "bar",
		},
	}

	realmName := "realm1"

	mockClient.On("GetUsers", realmName, gocloak.GetUsersParams{Username: gocloak.StringP(usr.Username)}).
		Return([]*gocloak.User{}, nil)

	goClUser := gocloak.User{
		Username:        &usr.Username,
		Enabled:         &usr.Enabled,
		EmailVerified:   &usr.EmailVerified,
		FirstName:       &usr.FirstName,
		LastName:        &usr.LastName,
		RequiredActions: &usr.RequiredUserActions,
		RealmRoles:      &usr.Roles,
		Groups:          &usr.Groups,
		Email:           &usr.Email,
		Attributes: &map[string][]string{
			"foo": []string{"bar"},
		},
	}

	mockClient.On("CreateUser", realmName, goClUser).Return(nil)

	if err := adapter.SyncRealmUser(realmName, &usr); err != nil {
		t.Fatal(err)
	}
}

func TestGoCloakAdapter_SyncRealmUser_UserExists(t *testing.T) {
	mockClient := new(MockGoCloakClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	usr := KeycloakUser{
		Username: "vasia",
	}

	realmName := "realm1"

	mockClient.On("GetUsers", realmName, gocloak.GetUsersParams{Username: gocloak.StringP(usr.Username)}).
		Return([]*gocloak.User{
			{
				Username: &usr.Username,
			},
		}, nil)

	err := adapter.SyncRealmUser(realmName, &usr)
	if err == nil {
		t.Fatal("no error on duplicated user")
	}

	if !IsErrDuplicated(err) {
		t.Log(err)
		t.Fatal("wrong error returned")
	}
}
