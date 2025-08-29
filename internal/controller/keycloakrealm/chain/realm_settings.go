package chain

import (
	"context"
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/keycloakrealm/chain/handler"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type RealmSettings struct {
	next handler.RealmHandler
}

func (h RealmSettings) ServeRequest(ctx context.Context, realm *keycloakApi.KeycloakRealm, kClient keycloak.Client) error {
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
			return fmt.Errorf("unable to set realm event config: %w", err)
		}
	}

	settings := adapter.RealmSettings{
		DisplayHTMLName: realm.Spec.DisplayHTMLName,
		FrontendURL:     realm.Spec.FrontendURL,
		DisplayName:     realm.Spec.DisplayName,
	}

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

	if len(realm.Spec.PasswordPolicies) > 0 {
		settings.PasswordPolicies = h.makePasswordPolicies(realm.Spec.PasswordPolicies)
	}

	settings.TokenSettings = adapter.ToRealmTokenSettings(realm.Spec.TokenSettings)

	if realm.Spec.RealmEventConfig != nil && realm.Spec.RealmEventConfig.AdminEventsEnabled {
		eventCfCopy := realm.Spec.RealmEventConfig.DeepCopy()

		settings.AdminEventsExpiration = &eventCfCopy.AdminEventsExpiration
	}

	if err := kClient.UpdateRealmSettings(realm.Spec.RealmName, &settings); err != nil {
		return fmt.Errorf("unable to update realm settings: %w", err)
	}

	if err := kClient.SetRealmOrganizationsEnabled(ctx, realm.Spec.RealmName, realm.Spec.OrganizationsEnabled); err != nil {
		return fmt.Errorf("unable to set realm organizations enabled: %w", err)
	}

	rLog.Info("Realm settings is updating done.")

	return nextServeOrNil(ctx, h.next, realm, kClient)
}

func (h RealmSettings) makePasswordPolicies(policiesSpec []keycloakApi.PasswordPolicy) []adapter.PasswordPolicy {
	policies := make([]adapter.PasswordPolicy, len(policiesSpec))
	for i, v := range policiesSpec {
		policies[i] = adapter.PasswordPolicy{Type: v.Type, Value: v.Value}
	}

	return policies
}
