package chain

import (
	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/pkg/errors"
)

type RealmSettings struct {
	next handler.RealmHandler
}

func (h RealmSettings) ServeRequest(realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
	rLog := log.WithValues("realm name", realm.Spec.RealmName)
	rLog.Info("Start updating of Keycloak realm settings")

	if realm.Spec.RealmEventConfig != nil {
		if err := kClient.SetRealmEventConfig(realm.Spec.RealmName, &adapter.RealmEventConfig{
			AdminEventsDetailsEnabled: realm.Spec.RealmEventConfig.AdminEventsDetailsEnabled,
			AdminEventsEnabled:        realm.Spec.RealmEventConfig.AdminEventsEnabled,
			EnabledEventTypes:         realm.Spec.RealmEventConfig.EnabledEventTypes,
			EventsEnabled:             realm.Spec.RealmEventConfig.EventsEnabled,
			EventsExpiration:          realm.Spec.RealmEventConfig.EventsExpiration,
			EventsListeners:           realm.Spec.RealmEventConfig.EventsListeners,
		}); err != nil {
			return errors.Wrap(err, "unable to set realm event config")
		}
	}

	if realm.Spec.BrowserSecurityHeaders == nil && realm.Spec.Themes == nil {
		rLog.Info("Realm settings is not set, exit.")
		return nextServeOrNil(h.next, realm, kClient)
	}

	settings := adapter.RealmSettings{}
	if realm.Spec.Themes != nil {
		settings.Themes = &adapter.RealmThemes{
			InternationalizationEnabled: realm.Spec.Themes.InternationalizationEnabled,
			EmailTheme:                  realm.Spec.Themes.EmailTheme,
			AdminConsoleTheme:           realm.Spec.Themes.AdminConsoleTheme,
			AccountTheme:                realm.Spec.Themes.AccountTheme,
			LoginTheme:                  realm.Spec.Themes.LoginTheme,
		}
	}

	if realm.Spec.BrowserSecurityHeaders != nil {
		settings.BrowserSecurityHeaders = realm.Spec.BrowserSecurityHeaders
	}

	if err := kClient.UpdateRealmSettings(realm.Spec.RealmName, &settings); err != nil {
		return errors.Wrap(err, "unable to update realm settings")
	}

	rLog.Info("Realm settings is updating done.")
	return nextServeOrNil(h.next, realm, kClient)
}
