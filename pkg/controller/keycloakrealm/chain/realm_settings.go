package chain

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
	"github.com/epam/edp-keycloak-operator/pkg/controller/keycloakrealm/chain/handler"
	"github.com/pkg/errors"
)

type RealmSettings struct {
	next handler.RealmHandler
}

func (h RealmSettings) ServeRequest(ctx context.Context, realm *v1alpha1.KeycloakRealm, kClient keycloak.Client) error {
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

	if realm.Spec.BrowserSecurityHeaders == nil && realm.Spec.Themes == nil && len(realm.Spec.PasswordPolicies) == 0 &&
		realm.Spec.BruteForceProtection == nil {
		rLog.Info("Realm settings is not set, exit.")
		return nextServeOrNil(ctx, h.next, realm, kClient)
	}

	settings := adapter.RealmSettings{}
	setAdapterSettings(&settings, &realm.Spec)

	if err := kClient.UpdateRealmSettings(realm.Spec.RealmName, &settings); err != nil {
		return errors.Wrap(err, "unable to update realm settings")
	}

	rLog.Info("Realm settings is updating done.")
	return nextServeOrNil(ctx, h.next, realm, kClient)
}

func setAdapterSettings(settings *adapter.RealmSettings, spec *v1alpha1.KeycloakRealmSpec) {
	if spec.Themes != nil {
		settings.Themes = &adapter.RealmThemes{
			InternationalizationEnabled: spec.Themes.InternationalizationEnabled,
			EmailTheme:                  spec.Themes.EmailTheme,
			AdminConsoleTheme:           spec.Themes.AdminConsoleTheme,
			AccountTheme:                spec.Themes.AccountTheme,
			LoginTheme:                  spec.Themes.LoginTheme,
		}
	}

	if spec.BrowserSecurityHeaders != nil {
		settings.BrowserSecurityHeaders = spec.BrowserSecurityHeaders
	}

	if len(spec.PasswordPolicies) > 0 {
		settings.PasswordPolicies = make([]adapter.PasswordPolicy, 0, len(spec.PasswordPolicies))
		for _, v := range spec.PasswordPolicies {
			settings.PasswordPolicies = append(settings.PasswordPolicies,
				adapter.PasswordPolicy{Type: v.Type, Value: v.Value})
		}
	}

	if spec.BruteForceProtection != nil {
		settings.BruteForceProtection = &adapter.BruteForceProtection{
			Enabled:                      spec.BruteForceProtection.Enabled,
			FailureFactor:                spec.BruteForceProtection.FailureFactor,
			MaxDeltaTimeSeconds:          spec.BruteForceProtection.MaxDeltaTimeSeconds,
			MaxFailureWaitSeconds:        spec.BruteForceProtection.MaxFailureWaitSeconds,
			MinimumQuickLoginWaitSeconds: spec.BruteForceProtection.MinimumQuickLoginWaitSeconds,
			PermanentLockout:             spec.BruteForceProtection.PermanentLockout,
			QuickLoginCheckMilliSeconds:  spec.BruteForceProtection.QuickLoginCheckMilliSeconds,
			WaitIncrementSeconds:         spec.BruteForceProtection.WaitIncrementSeconds,
		}
	}
}
