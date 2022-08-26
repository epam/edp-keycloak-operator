package adapter

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak/v10"
	"github.com/pkg/errors"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

type RealmSettings struct {
	Themes                 *RealmThemes
	BrowserSecurityHeaders *map[string]string
	PasswordPolicies       []PasswordPolicy
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

func (a GoCloakAdapter) UpdateRealmSettings(realmName string, realmSettings *RealmSettings) error {
	realm, err := a.client.GetRealm(context.Background(), a.token.AccessToken, realmName)
	if err != nil {
		return errors.Wrapf(err, "unable to realm: %s", realmName)
	}

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

	if err := a.client.UpdateRealm(context.Background(), a.token.AccessToken, *realm); err != nil {
		return errors.Wrap(err, "unable to update realm")
	}

	return nil
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

func (a GoCloakAdapter) CreateRealmWithDefaultConfig(realm *dto.Realm) error {
	log := a.log.WithValues(logKeyRealm, realm)
	log.Info("Start creating realm with default config...")

	_, err := a.client.CreateRealm(context.Background(), a.token.AccessToken, getDefaultRealm(realm))
	if err != nil {
		return errors.Wrap(err, "unable to create realm")
	}

	log.Info("End creating realm with default config")

	return nil
}

func (a GoCloakAdapter) DeleteRealm(ctx context.Context, realmName string) error {
	log := a.log.WithValues(logKeyRealm, realmName)
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
		keycloakApiParamAlias: mapper.IdentityProviderAlias,
		keycloakApiParamRealm: realmName,
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
		keycloakApiParamAlias: mapper.IdentityProviderAlias,
		keycloakApiParamRealm: realmName,
		keycloakApiParamId:    mapper.ID,
	}).SetBody(mapper).Put(a.basePath + updateMapperToIdentityProvider)

	if err = a.checkError(err, resp); err != nil {
		return errors.Wrapf(err, "unable to update identity provider mapper: %+v", mapper)
	}

	return nil
}
