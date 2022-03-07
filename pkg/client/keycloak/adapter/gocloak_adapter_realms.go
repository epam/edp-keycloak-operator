package adapter

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v10"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/pkg/errors"
)

type RealmSettings struct {
	Themes                 *RealmThemes
	BrowserSecurityHeaders *map[string]string
	PasswordPolicies       []PasswordPolicy
	BruteForceProtection   *BruteForceProtection
}

type PasswordPolicy struct {
	Type  string
	Value string
}

type BruteForceProtection struct {
	Enabled                      bool `json:"enabled"`
	PermanentLockout             bool `json:"permanentLockout"`
	FailureFactor                int  `json:"failureFactor"`
	QuickLoginCheckMilliSeconds  int  `json:"quickLoginCheckMilliSeconds"`
	MinimumQuickLoginWaitSeconds int  `json:"minimumQuickLoginWaitSeconds"`
	WaitIncrementSeconds         int  `json:"waitIncrementSeconds"`
	MaxFailureWaitSeconds        int  `json:"maxFailureWaitSeconds"`
	MaxDeltaTimeSeconds          int  `json:"maxDeltaTimeSeconds"`
}

type RealmThemes struct {
	LoginTheme                  *string
	AccountTheme                *string
	AdminConsoleTheme           *string
	EmailTheme                  *string
	InternationalizationEnabled *bool
}

func (realmSettings *RealmSettings) setToRealm(realm *gocloak.RealmRepresentation) {
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
		policies := make([]string, 0, len(realmSettings.PasswordPolicies))
		for _, v := range realmSettings.PasswordPolicies {
			policies = append(policies, fmt.Sprintf("%s(%s)", v.Type, v.Value))
		}

		realm.PasswordPolicy = gocloak.StringP(strings.Join(policies, " AND "))
	}

	if realmSettings.BruteForceProtection != nil {
		realm.BruteForceProtected = &realmSettings.BruteForceProtection.Enabled
		realm.PermanentLockout = &realmSettings.BruteForceProtection.PermanentLockout
		realm.FailureFactor = &realmSettings.BruteForceProtection.FailureFactor
		realm.QuickLoginCheckMilliSeconds = gocloak.Int64P(int64(realmSettings.BruteForceProtection.QuickLoginCheckMilliSeconds))
		realm.MinimumQuickLoginWaitSeconds = &realmSettings.BruteForceProtection.MinimumQuickLoginWaitSeconds
		realm.WaitIncrementSeconds = &realmSettings.BruteForceProtection.WaitIncrementSeconds
		realm.MaxFailureWaitSeconds = &realmSettings.BruteForceProtection.MaxFailureWaitSeconds
		realm.MaxDeltaTimeSeconds = &realmSettings.BruteForceProtection.MaxDeltaTimeSeconds
	}
}

func (a GoCloakAdapter) UpdateRealmSettings(realmName string, realmSettings *RealmSettings) error {
	realm, err := a.client.GetRealm(context.Background(), a.token.AccessToken, realmName)
	if err != nil {
		return errors.Wrapf(err, "unable to realm: %s", realmName)
	}

	realmSettings.setToRealm(realm)

	if err := a.client.UpdateRealm(context.Background(), a.token.AccessToken, *realm); err != nil {
		return errors.Wrap(err, "unable to update realm")
	}

	return nil
}

func (a GoCloakAdapter) ExistRealm(realmName string) (bool, error) {
	log := a.log.WithValues("realm", realmName)
	log.Info("Start check existing realm...")

	_, err := a.client.GetRealm(context.Background(), a.token.AccessToken, realmName)

	res, err := strip404(err)
	if err != nil {
		return false, err
	}

	log.Info("Check existing realm has been finished", "result", res)
	return res, nil
}

func (a GoCloakAdapter) CreateRealmWithDefaultConfig(realm *dto.Realm) error {
	log := a.log.WithValues("realm", realm)
	log.Info("Start creating realm with default config...")

	_, err := a.client.CreateRealm(context.Background(), a.token.AccessToken, getDefaultRealm(realm))
	if err != nil {
		return errors.Wrap(err, "unable to create realm")
	}

	log.Info("End creating realm with default config")
	return nil
}

func (a GoCloakAdapter) DeleteRealm(ctx context.Context, realmName string) error {
	log := a.log.WithValues("realm", realmName)
	log.Info("Start deleting realm...")

	if err := a.client.DeleteRealm(ctx, a.token.AccessToken, realmName); err != nil {
		return errors.Wrap(err, "unable to delete realm")
	}

	log.Info("End deletion realm")
	return nil
}

func (a GoCloakAdapter) SyncRealmIdentityProviderMappers(realmName string, mappers []dto.IdentityProviderMapper) error {
	realm, err := a.client.GetRealm(context.Background(), a.token.AccessToken, realmName)
	if err != nil {
		return errors.Wrapf(err, "unable to get realm by name: %s", realmName)
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
				return errors.Wrapf(err, "unable to update idp mapper: %+v", claimedMapper)
			}

			continue
		}

		if err := a.createIdentityProviderMapper(realmName, claimedMapper); err != nil {
			return errors.Wrapf(err, "unable to create idp mapper: %+v", claimedMapper)
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
		"alias": mapper.IdentityProviderAlias,
		"realm": realmName,
	}).SetBody(mapper).Post(a.basePath + mapperToIdentityProvider)

	if err != nil {
		return errors.Wrapf(err, "unable to create identity provider mapper: %+v", mapper)
	}

	if resp.StatusCode() != http.StatusCreated {
		return errors.Errorf("unable to create identity provider mapper: %+v, response: %s", mapper,
			resp.String())
	}

	return nil
}

func (a GoCloakAdapter) updateIdentityProviderMapper(realmName string, mapper dto.IdentityProviderMapper) error {
	resp, err := a.startRestyRequest().SetPathParams(map[string]string{
		"alias": mapper.IdentityProviderAlias,
		"realm": realmName,
		"id":    mapper.ID,
	}).SetBody(mapper).Put(a.basePath + updateMapperToIdentityProvider)

	if err := a.checkError(err, resp); err != nil {
		return errors.Wrapf(err, "unable to update identity provider mapper: %+v", mapper)
	}

	return nil
}
