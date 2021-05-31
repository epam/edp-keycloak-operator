package adapter

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
)

func TestGoCloakAdapter_UpdateRealmSettings(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

	settings := RealmSettings{
		Themes: &RealmThemes{
			LoginTheme: gocloak.StringP("keycloak"),
		},
		BrowserSecurityHeaders: &map[string]string{
			"foo": "bar",
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
	}
	mockClient.On("UpdateRealm", updateRealm).Return(nil)

	if err := adapter.UpdateRealmSettings(realmName, &settings); err != nil {
		t.Fatal(err)
	}
}

func TestGoCloakAdapter_SyncRealmIdentityProviderMappers(t *testing.T) {
	mockClient := new(MockGoCloakClient)
	restyClient := resty.New()
	httpmock.ActivateNonDefault(restyClient.GetClient())
	mockClient.On("RestyClient").Return(restyClient)

	adapter := GoCloakAdapter{
		client:   mockClient,
		token:    &gocloak.JWT{AccessToken: "token"},
		basePath: "",
	}

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
