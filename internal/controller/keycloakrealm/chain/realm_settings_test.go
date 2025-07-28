package chain

import (
	"context"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/mocks"
)

func TestRealmSettings_ServeRequest(t *testing.T) {
	rs := RealmSettings{}
	kClient := mocks.NewMockClient(t)
	realm := keycloakApi.KeycloakRealm{}
	ctx := context.Background()

	kClient.On("UpdateRealmSettings", mock.Anything, mock.Anything).Return(nil)
	kClient.On("SetRealmOrganizationsEnabled", mock.Anything, mock.Anything, mock.Anything).Return(nil)

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
				EventsListeners:       []string{"foo", "bar"},
				AdminEventsEnabled:    true,
				AdminEventsExpiration: 100,
			},
			PasswordPolicies: []keycloakApi.PasswordPolicy{
				{Type: "foo", Value: "bar"},
			},
			DisplayHTMLName: "<div class=\"kc-logo-text\"><span>Example</span></div>",
			FrontendURL:     "http://example.com",
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
		DisplayHTMLName:       realm.Spec.DisplayHTMLName,
		FrontendURL:           realm.Spec.FrontendURL,
		DisplayName:           realm.Spec.DisplayName,
		AdminEventsExpiration: ptr.To(100),
	}).Return(nil)

	kClient.On("SetRealmEventConfig", realm.Spec.RealmName, &adapter.RealmEventConfig{
		EventsListeners:    []string{"foo", "bar"},
		AdminEventsEnabled: true,
	}).Return(nil).Once()

	err = rs.ServeRequest(ctx, &realm, kClient)
	require.NoError(t, err)

	kClient.On("SetRealmEventConfig", realm.Spec.RealmName, &adapter.RealmEventConfig{
		EventsListeners:    []string{"foo", "bar"},
		AdminEventsEnabled: true,
	}).Return(errors.New("event config fatal")).Once()
	kClient.On("UpdateRealmSettings", mock.Anything, mock.Anything).Return(nil)
	kClient.On("SetRealmOrganizationsEnabled", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err = rs.ServeRequest(ctx, &realm, kClient)
	require.Error(t, err)

	if !strings.Contains(err.Error(), "unable to set realm event config") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}

	kClient.AssertExpectations(t)
}
