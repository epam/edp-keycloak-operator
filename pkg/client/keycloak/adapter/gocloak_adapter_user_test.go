package adapter

import (
	"context"
	"testing"

	"github.com/Nerzal/gocloak/v10"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGoCloakAdapter_SyncRealmUser(t *testing.T) {
	mockClient := new(MockGoCloakClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		basePath: "",
		token:    &gocloak.JWT{AccessToken: "token"},
	}

	restyClient := resty.New()
	httpmock.Reset()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	httpmock.RegisterResponder("PUT", "/auth/admin/realms/realm1/users/user-id1/reset-password",
		httpmock.NewStringResponder(200, ""))

	usr := KeycloakUser{
		Username: "vasia",
		Attributes: map[string]string{
			"foo": "bar",
		},
		RequiredUserActions: []string{"FOO"},
		Groups:              []string{"group1"},
		Password:            "123",
	}

	realmName := "realm1"

	mockClient.On("GetUsers", realmName, gocloak.GetUsersParams{Username: gocloak.StringP(usr.Username)}).
		Return([]*gocloak.User{}, nil)

	httpmock.RegisterResponder("GET", "/auth/admin/realms/realm1/users/user-id1/role-mappings/realm",
		httpmock.NewJsonResponderOrPanic(200, []UserRealmRoleMapping{
			{
				ID:   "role-id-1",
				Name: "role-name-1",
			},
		}))
	mockClient.On("DeleteRealmRoleFromUser", realmName, "user-id1", mock.Anything).Return(nil)
	httpmock.RegisterResponder("GET", "/auth/admin/realms/realm1/users/user-id1/groups",
		httpmock.NewJsonResponderOrPanic(200, []UserGroupMapping{
			{
				ID:   "group-id-1",
				Name: "group-name-1",
			},
		}))
	httpmock.RegisterResponder("DELETE", "/auth/admin/realms/realm1/users/user-id1/groups/group-id-1",
		httpmock.NewStringResponder(200, ""))

	goClUser := gocloak.User{
		Username:        &usr.Username,
		Enabled:         &usr.Enabled,
		EmailVerified:   &usr.EmailVerified,
		FirstName:       &usr.FirstName,
		LastName:        &usr.LastName,
		RequiredActions: &usr.RequiredUserActions,
		//Groups:          &usr.Groups,
		Email: &usr.Email,
		Attributes: &map[string][]string{
			"foo": {"bar"},
		},
	}

	mockClient.On("CreateUser", realmName, goClUser).Return("user-id1", nil)
	mockClient.On("GetGroups", realmName, mock.Anything).Return([]*gocloak.Group{
		{
			Name: gocloak.StringP("group1"),
			ID:   gocloak.StringP("group-id-2"),
		},
	}, nil)
	httpmock.RegisterResponder("PUT", "/auth/admin/realms/realm1/users/user-id1/groups/group-id-2",
		httpmock.NewStringResponder(200, ""))

	err := adapter.SyncRealmUser(context.Background(), realmName, &usr, false)
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
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
		Groups:     []string{"foo"},
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
		Groups:     &[]string{"g1", "g2"},
	}).Return(nil)

	mockClient.On("GetRealmRole", realmName, "r3").Return(&gocloak.Role{}, nil)
	mockClient.On("GetRealmRole", realmName, "r4").Return(&gocloak.Role{}, nil)
	mockClient.On("AddRealmRoleToUser", realmName, "id1", []gocloak.Role{{}}).Return(nil)
	mockClient.On("GetGroups", realmName, mock.Anything).Return([]*gocloak.Group{
		{
			ID:   gocloak.StringP("foo1"),
			Name: gocloak.StringP("foo"),
		},
	}, nil)

	restyClient := resty.New()
	httpmock.Reset()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)
	httpmock.RegisterResponder("PUT", "/auth/admin/realms/realm1/users/id1/groups/foo1",
		httpmock.NewStringResponder(200, ""))

	err := adapter.SyncRealmUser(context.Background(), realmName, &usr, true)
	require.NoError(t, err)

	mockClient.AssertExpectations(t)
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
		Groups:     &[]string{"g1", "g2"},
	}).Return(nil)

	mockClient.On("GetRealmRole", realmName, "r3").Return(&gocloak.Role{}, nil)
	mockClient.On("GetRealmRole", realmName, "r4").Return(&gocloak.Role{}, nil)
	mockClient.On("AddRealmRoleToUser", realmName, "id1", []gocloak.Role{{}}).
		Return(errors.New("add realm role fatal"))

	err := adapter.SyncRealmUser(context.Background(), realmName, &usr, true)
	require.Error(t, err)

	if err.Error() != "unable to sync user roles: unable to add realm role to user: unable to add realm role to user: add realm role fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
