package adapter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/Nerzal/gocloak/v10"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/jarcoal/httpmock"
)

func TestGoCloakAdapter_UpdateRealmSettings(t *testing.T) {
	adapter, mockClient, _, _ := initAdapter()

	settings := RealmSettings{
		Themes: &RealmThemes{
			LoginTheme: gocloak.StringP("keycloak"),
		},
		BrowserSecurityHeaders: &map[string]string{
			"foo": "bar",
		},
		PasswordPolicies: []PasswordPolicy{
			{Type: "foo", Value: "bar"},
			{Type: "bar", Value: "baz"},
		},
	}
	realmName := "ream11"

	realm := gocloak.RealmRepresentation{
		BrowserSecurityHeaders: &map[string]string{
			"test": "dets",
		},
	}
	mockClient.On("GetRealm", adapter.token.AccessToken, realmName).Return(&realm, nil)

	updateRealm := gocloak.RealmRepresentation{
		LoginTheme: settings.Themes.LoginTheme,
		BrowserSecurityHeaders: &map[string]string{
			"test": "dets",
			"foo":  "bar",
		},
		PasswordPolicy: gocloak.StringP("foo(bar) AND bar(baz)"),
	}
	mockClient.On("UpdateRealm", updateRealm).Return(nil)

	if err := adapter.UpdateRealmSettings(realmName, &settings); err != nil {
		t.Fatal(err)
	}
}

func TestGoCloakAdapter_SyncRealmIdentityProviderMappers(t *testing.T) {
	adapter, mockClient, restyClient, _ := initAdapter()
	httpmock.ActivateNonDefault(restyClient.GetClient())

	currentMapperID := "mp1id"

	mappers := []interface{}{
		map[string]interface{}{
			"id":   currentMapperID,
			"name": "mp1name",
		},
	}

	realm := gocloak.RealmRepresentation{
		Realm:                   gocloak.StringP("sso-realm-1"),
		IdentityProviderMappers: &mappers,
	}

	idpAlias := "alias-1"
	mockClient.On("GetRealm", adapter.token.AccessToken, *realm.Realm).Return(&realm, nil)

	httpmock.RegisterResponder(
		"POST",
		fmt.Sprintf("/auth/admin/realms/%s/identity-provider/instances/%s/mappers", *realm.Realm, idpAlias),
		httpmock.NewStringResponder(http.StatusCreated, "ok"))

	httpmock.RegisterResponder(
		"PUT",
		fmt.Sprintf("/auth/admin/realms/%s/identity-provider/instances/%s/mappers/%s", *realm.Realm, idpAlias,
			currentMapperID),
		httpmock.NewStringResponder(http.StatusOK, "ok"))

	if err := adapter.SyncRealmIdentityProviderMappers(*realm.Realm,
		[]dto.IdentityProviderMapper{
			{
				Name:                   "tname1",
				Config:                 map[string]string{"foo": "bar"},
				IdentityProviderMapper: "mapper-1",
				IdentityProviderAlias:  idpAlias,
			},
			{
				Name:                   "mp1name",
				Config:                 map[string]string{"foo": "bar"},
				IdentityProviderMapper: "mapper-2",
				IdentityProviderAlias:  idpAlias,
			},
		}); err != nil {
		t.Fatalf("%+v", err)
	}
}

func TestGoCloakAdapter_CreateRealmWithDefaultConfig(t *testing.T) {
	adapter, mockClient, _, _ := initAdapter()
	r := dto.Realm{}

	mockClient.On("CreateRealm", getDefaultRealm(&r)).Return("id1", nil).Once()
	if err := adapter.CreateRealmWithDefaultConfig(&r); err != nil {
		t.Fatal(err)
	}

	mockClient.On("CreateRealm", getDefaultRealm(&r)).Return("",
		errors.New("create realm fatal")).Once()
	err := adapter.CreateRealmWithDefaultConfig(&r)
	if err == nil {
		t.Fatal("no error returned")
	}

	if err.Error() != "unable to create realm: create realm fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestGoCloakAdapter_DeleteRealm(t *testing.T) {
	adapter, mockClient, _, _ := initAdapter()

	mockClient.On("DeleteRealm", "test-realm1").Return(nil).Once()
	if err := adapter.DeleteRealm(context.Background(), "test-realm1"); err != nil {
		t.Fatal(err)
	}

	mockClient.On("DeleteRealm", "test-realm2").Return(errors.New("delete fatal")).Once()
	err := adapter.DeleteRealm(context.Background(), "test-realm2")
	if err == nil {
		t.Fatal("no error returned")
	}

	if err.Error() != "unable to delete realm: delete fatal" {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
