package dto

import (
	"testing"

	keycloakApi "github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1"
)

func TestConvertSpecToClient(t *testing.T) {
	r := ConvertSpecToRealm(&keycloakApi.KeycloakRealmSpec{})
	if !r.SsoRealmEnabled {
		t.Fatal("sso realm enabled must be true when in spec is unset")
	}

	r = ConvertSpecToRealm(&keycloakApi.KeycloakRealmSpec{
		SsoRealmEnabled: nil,
	})
	if !r.SsoRealmEnabled {
		t.Fatal("sso realm enabled must be true when in spec is nil")
	}

	b := true
	r = ConvertSpecToRealm(&keycloakApi.KeycloakRealmSpec{
		SsoRealmEnabled: &b,
	})
	if !r.SsoRealmEnabled {
		t.Fatal("sso realm enabled must be true when in spec is true")
	}

	b = false
	r = ConvertSpecToRealm(&keycloakApi.KeycloakRealmSpec{
		SsoRealmEnabled: &b,
	})
	if r.SsoRealmEnabled {
		t.Fatal("sso realm enabled must be false when in spec is false")
	}
}
