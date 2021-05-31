package chain

import (
	"testing"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestRealmSettings_ServeRequest(t *testing.T) {
	rs := RealmSettings{}
	kClient := new(adapter.Mock)
	realm := keycloakApi.KeycloakRealm{}

	if err := rs.ServeRequest(&realm, kClient); err != nil {
		t.Fatal(err)
	}

	realm = keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm1",
			Themes: &keycloakApi.RealmThemes{
				LoginTheme: stringP("LoginTheme test"),
			},
			BrowserSecurityHeaders: &map[string]string{
				"foo": "bar",
			},
		},
	}

	kClient.On("UpdateRealmSettings", realm.Spec.RealmName, &adapter.RealmSettings{
		Themes: &adapter.RealmThemes{
			LoginTheme: stringP("LoginTheme test"),
		},
		BrowserSecurityHeaders: &map[string]string{
			"foo": "bar",
		},
	}).Return(nil)

	if err := rs.ServeRequest(&realm, kClient); err != nil {
		t.Fatal(err)
	}
}
