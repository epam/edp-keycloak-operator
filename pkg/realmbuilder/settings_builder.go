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
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
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
}

// ApplyRealmEventConfig sets the realm event configuration in Keycloak.
// It is a no-op if cfg is nil.
func ApplyRealmEventConfig(
	ctx context.Context,
	realmName string,
	cfg *common.RealmEventConfig,
	realmClient keycloakv2.RealmClient,
) error {
	if cfg == nil {
		return nil
	}

	rep := keycloakv2.RealmEventsConfigRepresentation{
		AdminEventsDetailsEnabled: ptr.To(cfg.AdminEventsDetailsEnabled),
		AdminEventsEnabled:        ptr.To(cfg.AdminEventsEnabled),
		EventsEnabled:             ptr.To(cfg.EventsEnabled),
		EventsExpiration:          ptr.To(int64(cfg.EventsExpiration)),
	}

	if cfg.EnabledEventTypes != nil {
		rep.EnabledEventTypes = &cfg.EnabledEventTypes
	}

	if cfg.EventsListeners != nil {
		rep.EventsListeners = &cfg.EventsListeners
	}

	if _, err := realmClient.SetRealmEventConfig(ctx, realmName, rep); err != nil {
		return fmt.Errorf("unable to set realm event config: %w", err)
	}

	return nil
}

// ApplyRealmSettings fetches the current realm from Keycloak, merges the overlay into it,
// and writes it back.
func ApplyRealmSettings(
	ctx context.Context,
	realmName string,
	overlay keycloakv2.RealmRepresentation,
	realmClient keycloakv2.RealmClient,
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

// BuildRealmRepresentationFromV1 builds a keycloakv2.RealmRepresentation with only the
// operator-managed fields populated from a v1.KeycloakRealm spec.
func BuildRealmRepresentationFromV1(realm *keycloakApi.KeycloakRealm) keycloakv2.RealmRepresentation {
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
		c.InternationalizationEnabled = spec.Themes.InternationalizationEnabled
	}

	return buildRealmRepresentationFromCommon(c)
}

// BuildRealmRepresentationFromV1Alpha1 builds a keycloakv2.RealmRepresentation with only the
// operator-managed fields populated from a v1alpha1.ClusterKeycloakRealm spec.
func BuildRealmRepresentationFromV1Alpha1(realm *v1alpha1.ClusterKeycloakRealm) keycloakv2.RealmRepresentation {
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
		c.InternationalizationEnabled = spec.Localization.InternationalizationEnabled
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
func buildRealmRepresentationFromCommon(spec commonRealmSpec) keycloakv2.RealmRepresentation {
	rep := keycloakv2.RealmRepresentation{
		DisplayName:                 ptr.To(spec.DisplayName),
		DisplayNameHtml:             ptr.To(spec.DisplayHTMLName),
		OrganizationsEnabled:        ptr.To(spec.OrganizationsEnabled),
		LoginTheme:                  spec.LoginTheme,
		AccountTheme:                spec.AccountTheme,
		AdminTheme:                  spec.AdminTheme,
		EmailTheme:                  spec.EmailTheme,
		InternationalizationEnabled: spec.InternationalizationEnabled,
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

	if spec.RealmEventConfig != nil && spec.RealmEventConfig.AdminEventsEnabled {
		if rep.Attributes == nil {
			attrs := make(map[string]string)
			rep.Attributes = &attrs
		}

		(*rep.Attributes)["adminEventsExpiration"] = strconv.Itoa(spec.RealmEventConfig.AdminEventsExpiration)
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
func MergeRealmRepresentation(base, overlay *keycloakv2.RealmRepresentation) {
	mergeRealmAppearance(base, overlay)
	mergeRealmTokenSettings(base, overlay)
	mergeRealmLoginSettings(base, overlay)
	mergeRealmSessionSettings(base, overlay)
	mergeRealmMaps(base, overlay)
}

func mergeRealmAppearance(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.DisplayName, &overlay.DisplayName)
	mergePtr(&base.DisplayNameHtml, &overlay.DisplayNameHtml)
	mergePtr(&base.OrganizationsEnabled, &overlay.OrganizationsEnabled)
	mergePtr(&base.LoginTheme, &overlay.LoginTheme)
	mergePtr(&base.AccountTheme, &overlay.AccountTheme)
	mergePtr(&base.AdminTheme, &overlay.AdminTheme)
	mergePtr(&base.EmailTheme, &overlay.EmailTheme)
	mergePtr(&base.InternationalizationEnabled, &overlay.InternationalizationEnabled)
	mergePtr(&base.PasswordPolicy, &overlay.PasswordPolicy)
}

func mergeRealmTokenSettings(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.DefaultSignatureAlgorithm, &overlay.DefaultSignatureAlgorithm)
	mergePtr(&base.RevokeRefreshToken, &overlay.RevokeRefreshToken)
	mergePtr(&base.RefreshTokenMaxReuse, &overlay.RefreshTokenMaxReuse)
	mergePtr(&base.AccessTokenLifespan, &overlay.AccessTokenLifespan)
	mergePtr(&base.AccessTokenLifespanForImplicitFlow, &overlay.AccessTokenLifespanForImplicitFlow)
	mergePtr(&base.AccessCodeLifespan, &overlay.AccessCodeLifespan)
	mergePtr(&base.ActionTokenGeneratedByUserLifespan, &overlay.ActionTokenGeneratedByUserLifespan)
	mergePtr(&base.ActionTokenGeneratedByAdminLifespan, &overlay.ActionTokenGeneratedByAdminLifespan)
}

func mergeRealmLoginSettings(base, overlay *keycloakv2.RealmRepresentation) {
	mergePtr(&base.RegistrationAllowed, &overlay.RegistrationAllowed)
	mergePtr(&base.ResetPasswordAllowed, &overlay.ResetPasswordAllowed)
	mergePtr(&base.RememberMe, &overlay.RememberMe)
	mergePtr(&base.RegistrationEmailAsUsername, &overlay.RegistrationEmailAsUsername)
	mergePtr(&base.LoginWithEmailAllowed, &overlay.LoginWithEmailAllowed)
	mergePtr(&base.DuplicateEmailsAllowed, &overlay.DuplicateEmailsAllowed)
	mergePtr(&base.VerifyEmail, &overlay.VerifyEmail)
	mergePtr(&base.EditUsernameAllowed, &overlay.EditUsernameAllowed)
}

func mergeRealmSessionSettings(base, overlay *keycloakv2.RealmRepresentation) {
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

func mergeRealmMaps(base, overlay *keycloakv2.RealmRepresentation) {
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

func setRealmRepSessionSettings(rep *keycloakv2.RealmRepresentation, sessions *common.RealmSessions) {
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
