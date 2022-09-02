package chain

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

func TestRealmSettings_ServeRequest(t *testing.T) {
	rs := RealmSettings{}
	kClient := new(adapter.Mock)
	realm := keycloakApi.KeycloakRealm{}
	ctx := context.Background()

	err := rs.ServeRequest(ctx, &realm, kClient)
	require.NoError(t, err)

	theme := "LoginTheme test"

	realm = keycloakApi.KeycloakRealm{
		Spec: keycloakApi.KeycloakRealmSpec{
			RealmName: "realm1",
			Themes: &keycloakApi.RealmThemes{
				LoginTheme: &theme,
			},
			BrowserSecurityHeaders: &map[string]string{
				"foo": "bar",
			},
			RealmEventConfig: &keycloakApi.RealmEventConfig{
				EventsListeners: []string{"foo", "bar"},
			},
			PasswordPolicies: []keycloakApi.PasswordPolicy{
				{Type: "foo", Value: "bar"},
			},
		},
	}

	kClient.On("UpdateRealmSettings", realm.Spec.RealmName, &adapter.RealmSettings{
		Themes: &adapter.RealmThemes{
			LoginTheme: &theme,
		},
		BrowserSecurityHeaders: &map[string]string{
			"foo": "bar",
		},
		PasswordPolicies: []adapter.PasswordPolicy{
			{Type: "foo", Value: "bar"},
		},
	}).Return(nil)

	kClient.On("SetRealmEventConfig", realm.Spec.RealmName, &adapter.RealmEventConfig{
		EventsListeners: []string{"foo", "bar"},
	}).Return(nil).Once()

	err = rs.ServeRequest(ctx, &realm, kClient)
	require.NoError(t, err)

	kClient.On("SetRealmEventConfig", realm.Spec.RealmName, &adapter.RealmEventConfig{
		EventsListeners: []string{"foo", "bar"},
	}).Return(errors.New("event config fatal")).Once()

	err = rs.ServeRequest(ctx, &realm, kClient)
	require.Error(t, err)

	if !strings.Contains(err.Error(), "unable to set realm event config") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	kClient.AssertExpectations(t)
}
