package realmbuilder

import (
	"fmt"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

// SettingsBuilder provides common functionality for building realm settings
// from both KeycloakRealm and ClusterKeycloakRealm specs.
type SettingsBuilder struct{}

// NewSettingsBuilder creates a new SettingsBuilder.
func NewSettingsBuilder() *SettingsBuilder {
	return &SettingsBuilder{}
}

// SetRealmEventConfigFromV1 sets the realm event configuration from v1.KeycloakRealm spec.
func (b *SettingsBuilder) SetRealmEventConfigFromV1(
	kClient keycloak.Client,
	realmName string,
	eventConfig *keycloakApi.RealmEventConfig,
) error {
	if eventConfig == nil {
		return nil
	}

	if err := kClient.SetRealmEventConfig(realmName, &adapter.RealmEventConfig{
		AdminEventsDetailsEnabled: eventConfig.AdminEventsDetailsEnabled,
		AdminEventsEnabled:        eventConfig.AdminEventsEnabled,
		EnabledEventTypes:         eventConfig.EnabledEventTypes,
		EventsEnabled:             eventConfig.EventsEnabled,
		EventsExpiration:          eventConfig.EventsExpiration,
		EventsListeners:           eventConfig.EventsListeners,
	}); err != nil {
		return fmt.Errorf("failed to set realm event config: %w", err)
	}

	return nil
}

// SetRealmEventConfigFromV1Alpha1 sets the realm event configuration from v1alpha1.ClusterKeycloakRealm spec.
func (b *SettingsBuilder) SetRealmEventConfigFromV1Alpha1(
	kClient keycloak.Client,
	realmName string,
	eventConfig *v1alpha1.RealmEventConfig,
) error {
	if eventConfig == nil {
		return nil
	}

	if err := kClient.SetRealmEventConfig(realmName, &adapter.RealmEventConfig{
		AdminEventsDetailsEnabled: eventConfig.AdminEventsDetailsEnabled,
		AdminEventsEnabled:        eventConfig.AdminEventsEnabled,
		EnabledEventTypes:         eventConfig.EnabledEventTypes,
		EventsEnabled:             eventConfig.EventsEnabled,
		EventsExpiration:          eventConfig.EventsExpiration,
		EventsListeners:           eventConfig.EventsListeners,
	}); err != nil {
		return fmt.Errorf("failed to set realm event config: %w", err)
	}

	return nil
}

// BuildFromV1 builds adapter.RealmSettings from v1.KeycloakRealm spec.
func (b *SettingsBuilder) BuildFromV1(realm *keycloakApi.KeycloakRealm) adapter.RealmSettings {
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
		settings.PasswordPolicies = makePasswordPoliciesFromV1(realm.Spec.PasswordPolicies)
	}

	settings.TokenSettings = adapter.ToRealmTokenSettings(realm.Spec.TokenSettings)

	if realm.Spec.RealmEventConfig != nil && realm.Spec.RealmEventConfig.AdminEventsEnabled {
		eventCfCopy := realm.Spec.RealmEventConfig.DeepCopy()
		settings.AdminEventsExpiration = &eventCfCopy.AdminEventsExpiration
	}

	if realm.Spec.Login != nil {
		settings.Login = &adapter.RealmLogin{
			UserRegistration: realm.Spec.Login.UserRegistration,
			ForgotPassword:   realm.Spec.Login.ForgotPassword,
			RememberMe:       realm.Spec.Login.RememberMe,
			EmailAsUsername:  realm.Spec.Login.EmailAsUsername,
			LoginWithEmail:   realm.Spec.Login.LoginWithEmail,
			DuplicateEmails:  realm.Spec.Login.DuplicateEmails,
			VerifyEmail:      realm.Spec.Login.VerifyEmail,
			EditUsername:     realm.Spec.Login.EditUsername,
		}
	}

	return settings
}

// BuildFromV1Alpha1 builds adapter.RealmSettings from v1alpha1.ClusterKeycloakRealm spec.
func (b *SettingsBuilder) BuildFromV1Alpha1(realm *v1alpha1.ClusterKeycloakRealm) adapter.RealmSettings {
	settings := adapter.RealmSettings{
		FrontendURL:     realm.Spec.FrontendURL,
		DisplayHTMLName: realm.Spec.DisplayHTMLName,
		DisplayName:     realm.Spec.DisplayName,
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
		if settings.Themes == nil {
			settings.Themes = &adapter.RealmThemes{}
		}

		settings.Themes.InternationalizationEnabled = realm.Spec.Localization.InternationalizationEnabled
	}

	if realm.Spec.BrowserSecurityHeaders != nil {
		settings.BrowserSecurityHeaders = realm.Spec.BrowserSecurityHeaders
	}

	if len(realm.Spec.PasswordPolicies) > 0 {
		settings.PasswordPolicies = makePasswordPoliciesFromV1Alpha1(realm.Spec.PasswordPolicies)
	}

	settings.TokenSettings = adapter.ToRealmTokenSettings(realm.Spec.TokenSettings)

	if realm.Spec.RealmEventConfig != nil && realm.Spec.RealmEventConfig.AdminEventsEnabled {
		eventCfCopy := realm.Spec.RealmEventConfig.DeepCopy()
		settings.AdminEventsExpiration = &eventCfCopy.AdminEventsExpiration
	}

	if realm.Spec.Login != nil {
		settings.Login = &adapter.RealmLogin{
			UserRegistration: realm.Spec.Login.UserRegistration,
			ForgotPassword:   realm.Spec.Login.ForgotPassword,
			RememberMe:       realm.Spec.Login.RememberMe,
			EmailAsUsername:  realm.Spec.Login.EmailAsUsername,
			LoginWithEmail:   realm.Spec.Login.LoginWithEmail,
			DuplicateEmails:  realm.Spec.Login.DuplicateEmails,
			VerifyEmail:      realm.Spec.Login.VerifyEmail,
			EditUsername:     realm.Spec.Login.EditUsername,
		}
	}

	return settings
}

func makePasswordPoliciesFromV1(policiesSpec []keycloakApi.PasswordPolicy) []adapter.PasswordPolicy {
	policies := make([]adapter.PasswordPolicy, len(policiesSpec))
	for i, v := range policiesSpec {
		policies[i] = adapter.PasswordPolicy{Type: v.Type, Value: v.Value}
	}

	return policies
}

func makePasswordPoliciesFromV1Alpha1(policiesSpec []v1alpha1.PasswordPolicy) []adapter.PasswordPolicy {
	policies := make([]adapter.PasswordPolicy, len(policiesSpec))
	for i, v := range policiesSpec {
		policies[i] = adapter.PasswordPolicy{Type: v.Type, Value: v.Value}
	}

	return policies
}
