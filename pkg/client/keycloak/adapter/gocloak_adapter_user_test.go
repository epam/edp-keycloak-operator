package adapter

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v8"
	"github.com/pkg/errors"
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
		RequiredUserActions: []string{"FOO"},
		Groups:              []string{"group1"},
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
		Groups:          &usr.Groups,
		Email:           &usr.Email,
		Attributes: &map[string][]string{
			"foo": {"bar"},
		},
	}

	mockClient.On("CreateUser", realmName, goClUser).Return(nil)

	if err := adapter.SyncRealmUser(context.Background(), realmName, &usr, false); err != nil {
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
		Username:   "vasia",
		Groups:     []string{"foo", "bar"},
		Attributes: map[string]string{"bar": "baz"},
		Roles:      []string{"r3", "r4"},
	}

	realmName := "realm1"

	mockClient.On("GetUsers", realmName, gocloak.GetUsersParams{Username: gocloak.StringP(usr.Username)}).
		Return([]*gocloak.User{
			{
				Username:   &usr.Username,
				ID:         gocloak.StringP("id1"),
				Groups:     &[]string{"g1", "g2"},
				RealmRoles: &[]string{"r1", "r2"},
				Attributes: &map[string][]string{"foo": {"baz", "zaz"}},
			},
		}, nil)

	mockClient.On("UpdateUser", realmName, gocloak.User{
		ID:         gocloak.StringP("id1"),
		Username:   gocloak.StringP("vasia"),
		Attributes: &map[string][]string{"bar": {"baz"}, "foo": {"baz", "zaz"}},
		RealmRoles: &[]string{"r1", "r2"},
		Groups:     &[]string{"foo", "bar", "g1", "g2"},
	}).Return(nil)

	mockClient.On("GetRealmRole", realmName, "r3").Return(&gocloak.Role{}, nil)
	mockClient.On("GetRealmRole", realmName, "r4").Return(&gocloak.Role{}, nil)
	mockClient.On("AddRealmRoleToUser", realmName, "id1", []gocloak.Role{{}}).Return(nil)

	if err := adapter.SyncRealmUser(context.Background(), realmName, &usr, true); err != nil {
		t.Fatal(err)
	}
}

func TestGoCloakAdapter_SyncRealmUser_UserExists_Failure(t *testing.T) {
	mockClient := new(MockGoCloakClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	usr := KeycloakUser{
		Username:   "vasia",
		Groups:     []string{"foo", "bar"},
		Attributes: map[string]string{"bar": "baz"},
		Roles:      []string{"r3", "r4"},
	}

	realmName := "realm1"

	mockClient.On("GetUsers", realmName, gocloak.GetUsersParams{Username: gocloak.StringP(usr.Username)}).
		Return([]*gocloak.User{
			{
				Username:   &usr.Username,
				ID:         gocloak.StringP("id1"),
				Groups:     &[]string{"g1", "g2"},
				RealmRoles: &[]string{"r1", "r2"},
				Attributes: &map[string][]string{"foo": {"baz", "zaz"}},
			},
		}, nil)

	mockClient.On("UpdateUser", realmName, gocloak.User{
		ID:         gocloak.StringP("id1"),
		Username:   gocloak.StringP("vasia"),
		Attributes: &map[string][]string{"bar": {"baz"}, "foo": {"baz", "zaz"}},
		RealmRoles: &[]string{"r1", "r2"},
		Groups:     &[]string{"foo", "bar", "g1", "g2"},
	}).Return(nil)

	mockClient.On("GetRealmRole", realmName, "r3").Return(&gocloak.Role{}, nil)
	mockClient.On("GetRealmRole", realmName, "r4").Return(&gocloak.Role{}, nil)
	mockClient.On("AddRealmRoleToUser", realmName, "id1", []gocloak.Role{{}}).
		Return(errors.New("add realm role fatal"))

	err := adapter.SyncRealmUser(context.Background(), realmName, &usr, true)
	if err == nil {
		t.Fatal("no error returned")
	}

	if err.Error() != "unable to add realm role to user: unable to add realm role to user: add realm role fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
