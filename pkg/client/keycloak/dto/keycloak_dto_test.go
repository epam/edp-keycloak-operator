package dto

import (
	"testing"

	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
)

func TestConvertSpecToClient(t *testing.T) {
	r := ConvertSpecToRealm(&keycloakApi.KeycloakRealmSpec{})
	require.False(t, r.SsoRealmEnabled)

	r = ConvertSpecToRealm(&keycloakApi.KeycloakRealmSpec{
		SsoRealmEnabled: nil,
	})
	require.False(t, r.SsoRealmEnabled)

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
