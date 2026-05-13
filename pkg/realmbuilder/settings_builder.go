package realmbuilder

import (
	"context"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/api/common"
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/api/v1alpha1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// commonRealmSpec holds the normalized, API-version-agnostic fields shared by
// KeycloakRealmSpec and ClusterKeycloakRealmSpec.
type commonRealmSpec struct {
	DisplayName                 string
	DisplayHTMLName             string
	OrganizationsEnabled        bool
	FrontendURL                 string
	BrowserSecurityHeaders      *map[string]string
	PasswordPolicy              string // pre-formatted "type(value) and …" string, empty if none
	TokenSettings               *common.TokenSettings
	RealmEventConfig            *common.RealmEventConfig
	Login                       *keycloakApi.RealmLogin
	Sessions                    *common.RealmSessions
	LoginTheme                  *string
	AccountTheme                *string
	AdminTheme                  *string
	EmailTheme                  *string
	InternationalizationEnabled *bool
	SupportedLocales            []string
	DefaultLocale               *string
}

// ApplyRealmEventConfig sets the realm event configuration in Keycloak.
// It fetches the current config first, overlays only the fields the user explicitly
// set (non-nil pointer fields), and writes the result back. This ensures that omitting
// a boolean field in the CR means "preserve current Keycloak value" rather than
// silently resetting it to false.
// It is a no-op if cfg is nil.
func ApplyRealmEventConfig(
	ctx context.Context,
	realmName string,
	cfg *common.RealmEventConfig,
	eventsClient keycloakapi.EventsClient,
) error {
	if cfg == nil {
		return nil
	}

	current, _, err := eventsClient.GetEventsConfig(ctx, realmName)
	if err != nil {
		return fmt.Errorf("unable to get current realm event config: %w", err)
	}

	if current == nil {
		current = &keycloakapi.RealmEventsConfigRepresentation{}
	}

	if cfg.AdminEventsDetailsEnabled != nil {
		current.AdminEventsDetailsEnabled = cfg.AdminEventsDetailsEnabled
	}

	if cfg.AdminEventsEnabled != nil {
		current.AdminEventsEnabled = cfg.AdminEventsEnabled
	}

	if cfg.EventsEnabled != nil {
		current.EventsEnabled = cfg.EventsEnabled
	}

	if cfg.EventsExpiration != nil {
		current.EventsExpiration = ptr.To(int64(*cfg.EventsExpiration))
	}

	if cfg.EnabledEventTypes != nil {
		current.EnabledEventTypes = &cfg.EnabledEventTypes
	}

	if cfg.EventsListeners != nil {
		current.EventsListeners = &cfg.EventsListeners
	}

	if _, err := eventsClient.SetEventsConfig(ctx, realmName, *current); err != nil {
		return fmt.Errorf("unable to set realm event config: %w", err)
	}

	return nil
}

// ApplyRealmSettings fetches the current realm from Keycloak, merges the overlay into it,
// and writes it back.
func ApplyRealmSettings(
	ctx context.Context,
	realmName string,
	overlay keycloakapi.RealmRepresentation,
	realmClient keycloakapi.RealmClient,
) error {
	current, _, err := realmClient.GetRealm(ctx, realmName)
	if err != nil {
		return fmt.Errorf("unable to get realm: %w", err)
	}

	MergeRealmRepresentation(current, &overlay)

	if _, err := realmClient.UpdateRealm(ctx, realmName, *current); err != nil {
		return fmt.Errorf("unable to update realm settings: %w", err)
	}

	return nil
}

// BuildRealmRepresentationFromV1 builds a keycloakapi.RealmRepresentation with only the
// operator-managed fields populated from a v1.KeycloakRealm spec.
func BuildRealmRepresentationFromV1(realm *keycloakApi.KeycloakRealm) keycloakapi.RealmRepresentation {
	spec := &realm.Spec

	c := commonRealmSpec{
		DisplayName:            spec.DisplayName,
		DisplayHTMLName:        spec.DisplayHTMLName,
		OrganizationsEnabled:   spec.OrganizationsEnabled,
		FrontendURL:            spec.FrontendURL,
		BrowserSecurityHeaders: spec.BrowserSecurityHeaders,
		TokenSettings:          spec.TokenSettings,
		RealmEventConfig:       spec.RealmEventConfig,
		Login:                  spec.Login,
		Sessions:               spec.Sessions,
		PasswordPolicy:         buildPasswordPolicy(spec.PasswordPolicies),
	}

	if spec.Themes != nil {
		c.LoginTheme = spec.Themes.LoginTheme
		c.AccountTheme = spec.Themes.AccountTheme
		c.AdminTheme = spec.Themes.AdminConsoleTheme
		c.EmailTheme = spec.Themes.EmailTheme
		//nolint:staticcheck // deprecated field still merged for backward compatibility; spec.localization overrides below
		c.InternationalizationEnabled = spec.Themes.InternationalizationEnabled
	}

	if spec.Localization != nil {
		loc := spec.Localization
		if loc.InternationalizationEnabled != nil {
			c.InternationalizationEnabled = loc.InternationalizationEnabled
		}

		if len(loc.SupportedLocales) > 0 {
			c.SupportedLocales = loc.SupportedLocales
		}

		if loc.DefaultLocale != nil {
			c.DefaultLocale = loc.DefaultLocale
		}

	}

	return buildRealmRepresentationFromCommon(c)
}

// BuildRealmRepresentationFromV1Alpha1 builds a keycloakapi.RealmRepresentation with only the
// operator-managed fields populated from a v1alpha1.ClusterKeycloakRealm spec.
func BuildRealmRepresentationFromV1Alpha1(realm *v1alpha1.ClusterKeycloakRealm) keycloakapi.RealmRepresentation {
	spec := &realm.Spec

	c := commonRealmSpec{
		DisplayName:            spec.DisplayName,
		DisplayHTMLName:        spec.DisplayHTMLName,
		OrganizationsEnabled:   spec.OrganizationsEnabled,
		FrontendURL:            spec.FrontendURL,
		BrowserSecurityHeaders: spec.BrowserSecurityHeaders,
		TokenSettings:          spec.TokenSettings,
		RealmEventConfig:       spec.RealmEventConfig,
		Login:                  spec.Login,
		Sessions:               spec.Sessions,
		PasswordPolicy:         buildPasswordPolicy(spec.PasswordPolicies),
	}

	if spec.Themes != nil {
		c.LoginTheme = spec.Themes.LoginTheme
		c.AccountTheme = spec.Themes.AccountTheme
		c.AdminTheme = spec.Themes.AdminConsoleTheme
		c.EmailTheme = spec.Themes.EmailTheme
	}

	if spec.Localization != nil {
		loc := spec.Localization
		if loc.InternationalizationEnabled != nil {
			c.InternationalizationEnabled = loc.InternationalizationEnabled
		}

		if len(loc.SupportedLocales) > 0 {
			c.SupportedLocales = loc.SupportedLocales
		}

		if loc.DefaultLocale != nil {
			c.DefaultLocale = loc.DefaultLocale
		}

	}

	return buildRealmRepresentationFromCommon(c)
}

// buildPasswordPolicy formats a slice of PasswordPolicy into the Keycloak string format,
// e.g. "length(8) and upperCase(1)". Returns empty string if the slice is empty.
func buildPasswordPolicy(policies []common.PasswordPolicy) string {
	if len(policies) == 0 {
		return ""
	}

	parts := make([]string, len(policies))
	for i, p := range policies {
		parts[i] = p.Type + "(" + p.Value + ")"
	}

	return strings.Join(parts, " and ")
}

// buildRealmRepresentationFromCommon constructs a RealmRepresentation from the
// normalized common spec. All version-specific field mapping is done by the callers.
func buildRealmRepresentationFromCommon(spec commonRealmSpec) keycloakapi.RealmRepresentation {
	rep := keycloakapi.RealmRepresentation{
		DisplayName:                 ptr.To(spec.DisplayName),
		DisplayNameHtml:             ptr.To(spec.DisplayHTMLName),
		OrganizationsEnabled:        ptr.To(spec.OrganizationsEnabled),
		LoginTheme:                  spec.LoginTheme,
		AccountTheme:                spec.AccountTheme,
		AdminTheme:                  spec.AdminTheme,
		EmailTheme:                  spec.EmailTheme,
		InternationalizationEnabled: spec.InternationalizationEnabled,
	}

	if len(spec.SupportedLocales) > 0 {
		sl := spec.SupportedLocales
		rep.SupportedLocales = &sl
	}

	if spec.DefaultLocale != nil {
		rep.DefaultLocale = spec.DefaultLocale
	}

	if spec.FrontendURL != "" {
		attrs := make(map[string]string)
		rep.Attributes = &attrs
		(*rep.Attributes)["frontendUrl"] = spec.FrontendURL
	}

	if spec.BrowserSecurityHeaders != nil {
		rep.BrowserSecurityHeaders = spec.BrowserSecurityHeaders
	}

	if spec.PasswordPolicy != "" {
		rep.PasswordPolicy = ptr.To(spec.PasswordPolicy)
	}

	if ts := spec.TokenSettings; ts != nil {
		rep.DefaultSignatureAlgorithm = ptr.To(ts.DefaultSignatureAlgorithm)
		rep.RevokeRefreshToken = ptr.To(ts.RevokeRefreshToken)
		rep.RefreshTokenMaxReuse = ptr.To(int32(ts.RefreshTokenMaxReuse))
		rep.AccessTokenLifespan = ptr.To(int32(ts.AccessTokenLifespan))
		rep.AccessTokenLifespanForImplicitFlow = ptr.To(int32(ts.AccessTokenLifespanForImplicitFlow))
		rep.AccessCodeLifespan = ptr.To(int32(ts.AccessCodeLifespan))
		rep.ActionTokenGeneratedByUserLifespan = ptr.To(int32(ts.ActionTokenGeneratedByUserLifespan))
		rep.ActionTokenGeneratedByAdminLifespan = ptr.To(int32(ts.ActionTokenGeneratedByAdminLifespan))
	}

	eventCfg := spec.RealmEventConfig
	if eventCfg != nil && eventCfg.AdminEventsEnabled != nil && *eventCfg.AdminEventsEnabled {
		if rep.Attributes == nil {
			attrs := make(map[string]string)
			rep.Attributes = &attrs
		}

		(*rep.Attributes)["adminEventsExpiration"] = strconv.Itoa(eventCfg.AdminEventsExpiration)
	}

	if l := spec.Login; l != nil {
		rep.RegistrationAllowed = ptr.To(l.UserRegistration)
		rep.ResetPasswordAllowed = ptr.To(l.ForgotPassword)
		rep.RememberMe = ptr.To(l.RememberMe)
		rep.RegistrationEmailAsUsername = ptr.To(l.EmailAsUsername)
		rep.LoginWithEmailAllowed = ptr.To(l.LoginWithEmail)
		rep.DuplicateEmailsAllowed = ptr.To(l.DuplicateEmails)
		rep.VerifyEmail = ptr.To(l.VerifyEmail)
		rep.EditUsernameAllowed = ptr.To(l.EditUsername)
	}

	setRealmRepSessionSettings(&rep, spec.Sessions)

	return rep
}

// MergeRealmRepresentation copies only the operator-managed fields from overlay onto base,
// merging map fields key-by-key to preserve live Keycloak values the operator doesn't manage.
func MergeRealmRepresentation(base, overlay *keycloakapi.RealmRepresentation) {
	mergeRealmAppearance(base, overlay)
	mergeRealmTokenSettings(base, overlay)
	mergeRealmLoginSettings(base, overlay)
	mergeRealmSessionSettings(base, overlay)
	mergeRealmMaps(base, overlay)
}

func mergeRealmAppearance(base, overlay *keycloakapi.RealmRepresentation) {
	mergePtr(&base.DisplayName, &overlay.DisplayName)
	mergePtr(&base.DisplayNameHtml, &overlay.DisplayNameHtml)
	mergePtr(&base.OrganizationsEnabled, &overlay.OrganizationsEnabled)
	mergePtr(&base.LoginTheme, &overlay.LoginTheme)
	mergePtr(&base.AccountTheme, &overlay.AccountTheme)
	mergePtr(&base.AdminTheme, &overlay.AdminTheme)
	mergePtr(&base.EmailTheme, &overlay.EmailTheme)
	mergePtr(&base.InternationalizationEnabled, &overlay.InternationalizationEnabled)
	mergePtr(&base.PasswordPolicy, &overlay.PasswordPolicy)
	mergePtr(&base.SupportedLocales, &overlay.SupportedLocales)
	mergePtr(&base.DefaultLocale, &overlay.DefaultLocale)
}

func mergeRealmTokenSettings(base, overlay *keycloakapi.RealmRepresentation) {
	mergePtr(&base.DefaultSignatureAlgorithm, &overlay.DefaultSignatureAlgorithm)
	mergePtr(&base.RevokeRefreshToken, &overlay.RevokeRefreshToken)
	mergePtr(&base.RefreshTokenMaxReuse, &overlay.RefreshTokenMaxReuse)
	mergePtr(&base.AccessTokenLifespan, &overlay.AccessTokenLifespan)
	mergePtr(&base.AccessTokenLifespanForImplicitFlow, &overlay.AccessTokenLifespanForImplicitFlow)
	mergePtr(&base.AccessCodeLifespan, &overlay.AccessCodeLifespan)
	mergePtr(&base.ActionTokenGeneratedByUserLifespan, &overlay.ActionTokenGeneratedByUserLifespan)
	mergePtr(&base.ActionTokenGeneratedByAdminLifespan, &overlay.ActionTokenGeneratedByAdminLifespan)
}

func mergeRealmLoginSettings(base, overlay *keycloakapi.RealmRepresentation) {
	mergePtr(&base.RegistrationAllowed, &overlay.RegistrationAllowed)
	mergePtr(&base.ResetPasswordAllowed, &overlay.ResetPasswordAllowed)
	mergePtr(&base.RememberMe, &overlay.RememberMe)
	mergePtr(&base.RegistrationEmailAsUsername, &overlay.RegistrationEmailAsUsername)
	mergePtr(&base.LoginWithEmailAllowed, &overlay.LoginWithEmailAllowed)
	mergePtr(&base.DuplicateEmailsAllowed, &overlay.DuplicateEmailsAllowed)
	mergePtr(&base.VerifyEmail, &overlay.VerifyEmail)
	mergePtr(&base.EditUsernameAllowed, &overlay.EditUsernameAllowed)
}

func mergeRealmSessionSettings(base, overlay *keycloakapi.RealmRepresentation) {
	mergePtr(&base.SsoSessionIdleTimeout, &overlay.SsoSessionIdleTimeout)
	mergePtr(&base.SsoSessionMaxLifespan, &overlay.SsoSessionMaxLifespan)
	mergePtr(&base.SsoSessionIdleTimeoutRememberMe, &overlay.SsoSessionIdleTimeoutRememberMe)
	mergePtr(&base.SsoSessionMaxLifespanRememberMe, &overlay.SsoSessionMaxLifespanRememberMe)
	mergePtr(&base.OfflineSessionIdleTimeout, &overlay.OfflineSessionIdleTimeout)
	mergePtr(&base.OfflineSessionMaxLifespanEnabled, &overlay.OfflineSessionMaxLifespanEnabled)
	mergePtr(&base.OfflineSessionMaxLifespan, &overlay.OfflineSessionMaxLifespan)
	mergePtr(&base.AccessCodeLifespanLogin, &overlay.AccessCodeLifespanLogin)
	mergePtr(&base.AccessCodeLifespanUserAction, &overlay.AccessCodeLifespanUserAction)
}

func mergeRealmMaps(base, overlay *keycloakapi.RealmRepresentation) {
	// BrowserSecurityHeaders: merge keys into base map
	if overlay.BrowserSecurityHeaders != nil {
		if base.BrowserSecurityHeaders == nil {
			m := make(map[string]string)
			base.BrowserSecurityHeaders = &m
		}

		maps.Copy(*base.BrowserSecurityHeaders, *overlay.BrowserSecurityHeaders)
	}

	// Attributes: merge keys into base map
	if overlay.Attributes != nil {
		if base.Attributes == nil {
			attrs := make(map[string]string)
			base.Attributes = &attrs
		}

		maps.Copy(*base.Attributes, *overlay.Attributes)
	}
}

func setRealmRepSessionSettings(rep *keycloakapi.RealmRepresentation, sessions *common.RealmSessions) {
	if sessions == nil {
		return
	}

	if s := sessions.SSOSessionSettings; s != nil {
		rep.SsoSessionIdleTimeout = ptr.To(int32(s.IdleTimeout))
		rep.SsoSessionMaxLifespan = ptr.To(int32(s.MaxLifespan))
		rep.SsoSessionIdleTimeoutRememberMe = ptr.To(int32(s.IdleTimeoutRememberMe))
		rep.SsoSessionMaxLifespanRememberMe = ptr.To(int32(s.MaxLifespanRememberMe))
	}

	if s := sessions.SSOOfflineSessionSettings; s != nil {
		rep.OfflineSessionIdleTimeout = ptr.To(int32(s.IdleTimeout))
		rep.OfflineSessionMaxLifespanEnabled = ptr.To(s.MaxLifespanEnabled)
		rep.OfflineSessionMaxLifespan = ptr.To(int32(s.MaxLifespan))
	}

	if s := sessions.SSOLoginSettings; s != nil {
		rep.AccessCodeLifespanLogin = ptr.To(int32(s.AccessCodeLifespanLogin))
		rep.AccessCodeLifespanUserAction = ptr.To(int32(s.AccessCodeLifespanUserAction))
	}
}

// mergePtr copies *overlay into *base only when *overlay is non-nil.
func mergePtr[T any](base, overlay **T) {
	if *overlay != nil {
		*base = *overlay
	}
}
