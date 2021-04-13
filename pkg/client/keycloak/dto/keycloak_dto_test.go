package dto

import (
	"testing"

	"github.com/epam/keycloak-operator/v2/pkg/apis/v1/v1alpha1"
)

func TestConvertSpecToClient(t *testing.T) {
	r := ConvertSpecToRealm(v1alpha1.KeycloakRealmSpec{})
	if !r.SsoRealmEnabled {
		t.Fatal("sso realm enabled must be true when in spec is unset")
	}

	r = ConvertSpecToRealm(v1alpha1.KeycloakRealmSpec{
		SsoRealmEnabled: nil,
	})
	if !r.SsoRealmEnabled {
		t.Fatal("sso realm enabled must be true when in spec is nil")
	}

	b := true
	r = ConvertSpecToRealm(v1alpha1.KeycloakRealmSpec{
		SsoRealmEnabled: &b,
	})
	if !r.SsoRealmEnabled {
		t.Fatal("sso realm enabled must be true when in spec is true")
	}

	b = false
	r = ConvertSpecToRealm(v1alpha1.KeycloakRealmSpec{
		SsoRealmEnabled: &b,
	})
	if r.SsoRealmEnabled {
		t.Fatal("sso realm enabled must be false when in spec is false")
	}
}
