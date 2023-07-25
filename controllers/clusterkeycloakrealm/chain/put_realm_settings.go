package chain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

// PutRealmSettings is responsible for updating of keycloak realm settings.
type PutRealmSettings struct {
}

// NewPutRealmSettings creates a new PutRealmSettings handler.
func NewPutRealmSettings() *PutRealmSettings {
	return &PutRealmSettings{}
}

func (h PutRealmSettings) ServeRequest(ctx context.Context, realm *v1alpha1.ClusterKeycloakRealm, kClient keycloak.Client) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start updating of keycloak realm settings")

	if realm.Spec.RealmEventConfig != nil {
		if err := kClient.SetRealmEventConfig(realm.Spec.RealmName, &adapter.RealmEventConfig{
			AdminEventsDetailsEnabled: realm.Spec.RealmEventConfig.AdminEventsDetailsEnabled,
			AdminEventsEnabled:        realm.Spec.RealmEventConfig.AdminEventsEnabled,
			EnabledEventTypes:         realm.Spec.RealmEventConfig.EnabledEventTypes,
			EventsEnabled:             realm.Spec.RealmEventConfig.EventsEnabled,
			EventsExpiration:          realm.Spec.RealmEventConfig.EventsExpiration,
			EventsListeners:           realm.Spec.RealmEventConfig.EventsListeners,
		}); err != nil {
			return fmt.Errorf("failed to set realm event config: %w", err)
		}
	}

	settings := adapter.RealmSettings{
		FrontendURL: realm.Spec.FrontendURL,
	}

	if realm.Spec.Themes != nil {
		settings.Themes = &adapter.RealmThemes{
			EmailTheme:        realm.Spec.Themes.EmailTheme,
			AdminConsoleTheme: realm.Spec.Themes.AdminConsoleTheme,
			AccountTheme:      realm.Spec.Themes.AccountTheme,
			LoginTheme:        realm.Spec.Themes.LoginTheme,
		}
	}

	if realm.Spec.Localization != nil {
		settings.Themes.InternationalizationEnabled = realm.Spec.Localization.InternationalizationEnabled
	}

	if realm.Spec.BrowserSecurityHeaders != nil {
		settings.BrowserSecurityHeaders = realm.Spec.BrowserSecurityHeaders
	}

	if len(realm.Spec.PasswordPolicies) > 0 {
		settings.PasswordPolicies = h.makePasswordPolicies(realm.Spec.PasswordPolicies)
	}

	if err := kClient.UpdateRealmSettings(realm.Spec.RealmName, &settings); err != nil {
		return errors.Wrap(err, "unable to update realm settings")
	}

	log.Info("Realm settings is updating done.")

	return nil
}

func (h PutRealmSettings) makePasswordPolicies(policiesSpec []v1alpha1.PasswordPolicy) []adapter.PasswordPolicy {
	policies := make([]adapter.PasswordPolicy, len(policiesSpec))
	for i, v := range policiesSpec {
		policies[i] = adapter.PasswordPolicy{Type: v.Type, Value: v.Value}
	}

	return policies
}
