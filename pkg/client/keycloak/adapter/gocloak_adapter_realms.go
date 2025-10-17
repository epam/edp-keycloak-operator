package adapter

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/api/common"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type RealmSettings struct {
	Themes                 *RealmThemes
	BrowserSecurityHeaders *map[string]string
	PasswordPolicies       []PasswordPolicy
	DisplayHTMLName        string
	FrontendURL            string
	TokenSettings          *TokenSettings
	DisplayName            string
	AdminEventsExpiration  *int
	Login                  *RealmLogin
}

type PasswordPolicy struct {
	Type  string
	Value string
}

type RealmThemes struct {
	LoginTheme                  *string
	AccountTheme                *string
	AdminConsoleTheme           *string
	EmailTheme                  *string
	InternationalizationEnabled *bool
}

type TokenSettings struct {
	DefaultSignatureAlgorithm           string
	RevokeRefreshToken                  bool
	RefreshTokenMaxReuse                int
	AccessTokenLifespan                 int
	AccessTokenLifespanForImplicitFlow  int
	AccessCodeLifespan                  int
	ActionTokenGeneratedByUserLifespan  int
	ActionTokenGeneratedByAdminLifespan int
}

type RealmLogin struct {
	UserRegistration bool
	ForgotPassword   bool
	RememberMe       bool
	EmailAsUsername  bool
	LoginWithEmail   bool
	DuplicateEmails  bool
	VerifyEmail      bool
	EditUsername     bool
}

func (a GoCloakAdapter) UpdateRealmSettings(realmName string, realmSettings *RealmSettings) error {
	realm, err := a.client.GetRealm(context.Background(), a.token.AccessToken, realmName)
	if err != nil {
		return fmt.Errorf("unable to realm: %s: %w", realmName, err)
	}

	setRealmSettings(realm, realmSettings)

	if err := a.client.UpdateRealm(context.Background(), a.token.AccessToken, *realm); err != nil {
		return fmt.Errorf("unable to update realm: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) UpdateRealm(ctx context.Context, realm *gocloak.RealmRepresentation) error {
	if err := a.client.UpdateRealm(ctx, a.token.AccessToken, *realm); err != nil {
		return fmt.Errorf("unable to update realm: %w", err)
	}

	return nil
}

type RealmOrganizationsEnabled struct {
	Realm                string `json:"realm"`
	OrganizationsEnabled bool   `json:"organizationsEnabled"`
}

// SetRealmOrganizationsEnabled sets the organizations enabled flag for a realm.
// This method is workaround because the OrganizationsEnabled field is not available
// in the github.com/Nerzal/gocloak/v12 package.
// TODO: remove this method and use UpdateRealm when the OrganizationsEnabled field will be available
// in the github.com/Nerzal/gocloak/v12 package.
func (a GoCloakAdapter) SetRealmOrganizationsEnabled(ctx context.Context, realmName string, enabled bool) error {
	// Get current realm to check OrganizationsEnabled
	var currentRealm RealmOrganizationsEnabled
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetResult(&currentRealm).
		Get(a.buildPath(realmEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to get realm: %w", err)
	}

	// Check if OrganizationsEnabled is different
	if currentRealm.OrganizationsEnabled == enabled {
		return nil
	}

	// Create request body for updating organizations enabled
	requestBody := RealmOrganizationsEnabled{
		Realm:                realmName,
		OrganizationsEnabled: enabled,
	}

	// Make custom request to update realm organizations enabled
	rsp, err = a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
		}).
		SetBody(requestBody).
		Put(a.buildPath(realmEntity))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to set realm organizations enabled: %w", err)
	}

	return nil
}

func setRealmSettings(realm *gocloak.RealmRepresentation, realmSettings *RealmSettings) {
	if realmSettings.Themes != nil {
		realm.LoginTheme = realmSettings.Themes.LoginTheme
		realm.AccountTheme = realmSettings.Themes.AccountTheme
		realm.AdminTheme = realmSettings.Themes.AdminConsoleTheme
		realm.EmailTheme = realmSettings.Themes.EmailTheme
		realm.InternationalizationEnabled = realmSettings.Themes.InternationalizationEnabled
	}

	if realmSettings.BrowserSecurityHeaders != nil {
		if realm.BrowserSecurityHeaders == nil {
			bsh := make(map[string]string)
			realm.BrowserSecurityHeaders = &bsh
		}

		realmBrowserSecurityHeaders := *realm.BrowserSecurityHeaders
		for k, v := range *realmSettings.BrowserSecurityHeaders {
			realmBrowserSecurityHeaders[k] = v
		}

		realm.BrowserSecurityHeaders = &realmBrowserSecurityHeaders
	}

	if len(realmSettings.PasswordPolicies) > 0 {
		policies := make([]string, len(realmSettings.PasswordPolicies))
		for i, v := range realmSettings.PasswordPolicies {
			policies[i] = fmt.Sprintf("%s(%s)", v.Type, v.Value)
		}

		realm.PasswordPolicy = gocloak.StringP(strings.Join(policies, " and "))
	}

	if realmSettings.FrontendURL != "" {
		if realm.Attributes == nil {
			realm.Attributes = &map[string]string{}
		}

		(*realm.Attributes)["frontendUrl"] = realmSettings.FrontendURL
	}

	realm.DisplayName = gocloak.StringP(realmSettings.DisplayName)
	realm.DisplayNameHTML = gocloak.StringP(realmSettings.DisplayHTMLName)

	if realmSettings.TokenSettings != nil {
		realm.DefaultSignatureAlgorithm = gocloak.StringP(realmSettings.TokenSettings.DefaultSignatureAlgorithm)
		realm.RevokeRefreshToken = gocloak.BoolP(realmSettings.TokenSettings.RevokeRefreshToken)
		realm.RefreshTokenMaxReuse = gocloak.IntP(realmSettings.TokenSettings.RefreshTokenMaxReuse)
		realm.AccessTokenLifespan = gocloak.IntP(realmSettings.TokenSettings.AccessTokenLifespan)
		realm.AccessTokenLifespanForImplicitFlow = gocloak.IntP(
			realmSettings.TokenSettings.AccessTokenLifespanForImplicitFlow,
		)
		realm.AccessCodeLifespan = gocloak.IntP(realmSettings.TokenSettings.AccessCodeLifespan)
		realm.ActionTokenGeneratedByUserLifespan = gocloak.IntP(
			realmSettings.TokenSettings.ActionTokenGeneratedByUserLifespan,
		)
		realm.ActionTokenGeneratedByAdminLifespan = gocloak.IntP(
			realmSettings.TokenSettings.ActionTokenGeneratedByAdminLifespan,
		)
	}

	if realmSettings.AdminEventsExpiration != nil {
		(*realm.Attributes)["adminEventsExpiration"] = strconv.Itoa(*realmSettings.AdminEventsExpiration)
	}

	if realmSettings.Login != nil {
		realm.RegistrationAllowed = gocloak.BoolP(realmSettings.Login.UserRegistration)
		realm.ResetPasswordAllowed = gocloak.BoolP(realmSettings.Login.ForgotPassword)
		realm.RememberMe = gocloak.BoolP(realmSettings.Login.RememberMe)
		realm.RegistrationEmailAsUsername = gocloak.BoolP(realmSettings.Login.EmailAsUsername)
		realm.LoginWithEmailAllowed = gocloak.BoolP(realmSettings.Login.LoginWithEmail)
		realm.DuplicateEmailsAllowed = gocloak.BoolP(realmSettings.Login.DuplicateEmails)
		realm.VerifyEmail = gocloak.BoolP(realmSettings.Login.VerifyEmail)
		realm.EditUsernameAllowed = gocloak.BoolP(realmSettings.Login.EditUsername)
	}
}

func (a GoCloakAdapter) ExistRealm(realmName string) (bool, error) {
	log := a.log.WithValues(logKeyRealm, realmName)
	log.Info("Start check existing realm...")

	_, err := a.client.GetRealm(context.Background(), a.token.AccessToken, realmName)

	res, err := strip404(err)
	if err != nil {
		return false, err
	}

	log.Info("Check existing realm has been finished", "result", res)

	return res, nil
}

// GetRealm returns realm by name.
func (a GoCloakAdapter) GetRealm(ctx context.Context, realmName string) (*gocloak.RealmRepresentation, error) {
	log := ctrl.LoggerFrom(ctx).WithValues(logKeyRealm, realmName)
	log.Info("Start getting realm")

	r, err := a.client.GetRealm(ctx, a.token.AccessToken, realmName)
	if err != nil {
		return nil, fmt.Errorf("unable to get realm: %w", err)
	}

	return r, nil
}

func (a GoCloakAdapter) CreateRealmWithDefaultConfig(realm *dto.Realm) error {
	log := a.log.WithValues(logKeyRealm, realm)
	log.Info("Start creating realm with default config...")

	_, err := a.client.CreateRealm(context.Background(), a.token.AccessToken, getDefaultRealm(realm))
	if err != nil {
		return fmt.Errorf("unable to create realm: %w", err)
	}

	log.Info("End creating realm with default config")

	return nil
}

func (a GoCloakAdapter) DeleteRealm(ctx context.Context, realmName string) error {
	log := a.log.WithValues(logKeyRealm, realmName)
	log.Info("Start deleting realm...")

	if err := a.client.DeleteRealm(ctx, a.token.AccessToken, realmName); err != nil {
		return fmt.Errorf("unable to delete realm: %w", err)
	}

	log.Info("End deletion realm")

	return nil
}

func (a GoCloakAdapter) SyncRealmIdentityProviderMappers(realmName string, mappers []dto.IdentityProviderMapper) error {
	realm, err := a.client.GetRealm(context.Background(), a.token.AccessToken, realmName)
	if err != nil {
		return fmt.Errorf("unable to get realm by name: %s: %w", realmName, err)
	}

	currentMappers := make(map[string]*dto.IdentityProviderMapper)

	if realm.IdentityProviderMappers != nil {
		for _, idpm := range *realm.IdentityProviderMappers {
			if idpmTyped, ok := decodeIdentityProviderMapper(idpm); ok {
				currentMappers[idpmTyped.Name] = idpmTyped
			}
		}
	}

	for _, claimedMapper := range mappers {
		if idpmTyped, ok := currentMappers[claimedMapper.Name]; ok {
			claimedMapper.ID = idpmTyped.ID
			if err := a.updateIdentityProviderMapper(realmName, claimedMapper); err != nil {
				return fmt.Errorf("unable to update idp mapper: %+v: %w", claimedMapper, err)
			}

			continue
		}

		if err := a.createIdentityProviderMapper(realmName, claimedMapper); err != nil {
			return fmt.Errorf("unable to create idp mapper: %+v: %w", claimedMapper, err)
		}
	}

	return nil
}

func decodeIdentityProviderMapper(mp interface{}) (*dto.IdentityProviderMapper, bool) {
	mapInterface, ok := mp.(map[string]interface{})
	if !ok {
		return nil, false
	}

	mapper := dto.IdentityProviderMapper{
		Config: make(map[string]string),
	}

	if idRaw, ok := mapInterface["id"]; ok {
		if id, ok := idRaw.(string); ok {
			mapper.ID = id
		}
	}

	if nameRaw, ok := mapInterface["name"]; ok {
		if name, ok := nameRaw.(string); ok {
			mapper.Name = name
		}
	}

	if identityProviderAliasRaw, ok := mapInterface["identityProviderAlias"]; ok {
		if identityProviderAlias, ok := identityProviderAliasRaw.(string); ok {
			mapper.IdentityProviderAlias = identityProviderAlias
		}
	}

	if identityProviderMapperRaw, ok := mapInterface["identityProviderMapper"]; ok {
		if identityProviderMapper, ok := identityProviderMapperRaw.(string); ok {
			mapper.IdentityProviderMapper = identityProviderMapper
		}
	}

	if configRaw, ok := mapInterface["config"]; ok {
		if configInterface, ok := configRaw.(map[string]interface{}); ok {
			for k, v := range configInterface {
				if value, ok := v.(string); ok {
					mapper.Config[k] = value
				}
			}
		}
	}

	return &mapper, true
}

func (a GoCloakAdapter) createIdentityProviderMapper(realmName string, mapper dto.IdentityProviderMapper) error {
	resp, err := a.startRestyRequest().SetPathParams(map[string]string{
		keycloakApiParamAlias: mapper.IdentityProviderAlias,
		keycloakApiParamRealm: realmName,
	}).SetBody(mapper).Post(a.buildPath(mapperToIdentityProvider))

	if err != nil {
		return fmt.Errorf("failed to create identity provider mapper - %+v: %w", mapper, err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("failed to create identity provider mapper - %+v, response: %s", mapper,
			resp.String())
	}

	return nil
}

func (a GoCloakAdapter) updateIdentityProviderMapper(realmName string, mapper dto.IdentityProviderMapper) error {
	resp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamAlias: mapper.IdentityProviderAlias,
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    mapper.ID,
		}).
		SetBody(mapper).
		Put(a.buildPath(updateMapperToIdentityProvider))

	if err = a.checkError(err, resp); err != nil {
		return fmt.Errorf("failed to update identity provider mapper - %+v: %w", mapper, err)
	}

	return nil
}

func ToRealmTokenSettings(tokenSettings *common.TokenSettings) *TokenSettings {
	if tokenSettings == nil {
		return nil
	}

	return &TokenSettings{
		DefaultSignatureAlgorithm:           tokenSettings.DefaultSignatureAlgorithm,
		RevokeRefreshToken:                  tokenSettings.RevokeRefreshToken,
		RefreshTokenMaxReuse:                tokenSettings.RefreshTokenMaxReuse,
		AccessTokenLifespan:                 tokenSettings.AccessTokenLifespan,
		AccessTokenLifespanForImplicitFlow:  tokenSettings.AccessTokenLifespanForImplicitFlow,
		AccessCodeLifespan:                  tokenSettings.AccessCodeLifespan,
		ActionTokenGeneratedByUserLifespan:  tokenSettings.ActionTokenGeneratedByUserLifespan,
		ActionTokenGeneratedByAdminLifespan: tokenSettings.ActionTokenGeneratedByAdminLifespan,
	}
}
