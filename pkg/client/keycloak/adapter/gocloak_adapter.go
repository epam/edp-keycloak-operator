package adapter

import (
	"fmt"
	"gopkg.in/nerzal/gocloak.v2"
	"keycloak-operator/pkg/client/keycloak/api"
	"keycloak-operator/pkg/client/keycloak/dto"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
)

const (
	idPResource       = "/auth/admin/realms/{realm}/identity-provider/instances"
	idPMapperResource = "/auth/admin/realms/{realm}/identity-provider/instances/{alias}/mappers"
	getOneIdP         = idPResource + "/{alias}"
)

var log = logf.Log.WithName("gocloak_adapter")

type GoCloakAdapter struct {
	client gocloak.GoCloak
	token  gocloak.JWT

	basePath string
}

func (a GoCloakAdapter) ExistRealm(realm dto.Realm) (*bool, error) {
	reqLog := log.WithValues("realm", realm)
	reqLog.Info("Start check existing realm...")

	_, err := a.client.GetRealm(a.token.AccessToken, realm.Name)

	res, err := strip404(err)

	if err != nil {
		return nil, err
	}

	reqLog.Info("Check existing realm has been finished", "result", res)
	return &res, nil
}

func (a GoCloakAdapter) CreateRealmWithDefaultConfig(realm dto.Realm) error {
	reqLog := log.WithValues("realm", realm)
	reqLog.Info("Start creating realm with default config...")

	err := a.client.CreateRealm(a.token.AccessToken, getDefaultRealm(realm.Name))
	if err != nil {
		return err
	}

	reqLog.Info("End creating realm with default config")
	return nil
}

func (a GoCloakAdapter) ExistCentralIdentityProvider(realm dto.Realm) (*bool, error) {
	reqLog := log.WithValues("realm", realm)
	reqLog.Info("Start check central identity provider in realm")

	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realm.Name,
			"alias": "openshift",
		}).
		Get(a.basePath + "/" + getOneIdP)

	if err != nil {
		return nil, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		res := false
		return &res, nil
	}
	if resp.StatusCode() != http.StatusOK {
		err = fmt.Errorf("errors in get idP, responce: %s", resp.String())
		return nil, err
	}
	res := true

	reqLog.Info("End check central identity provider in realm")
	return &res, nil
}

func (a GoCloakAdapter) CreateCentralIdentityProvider(realm dto.Realm, client dto.Client) error {
	reqLog := log.WithValues("realm", realm, "keycloak client", client)
	reqLog.Info("Start create central identity provider...")

	idP := a.getCentralIdP(client)

	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realm.Name,
		}).
		SetBody(idP).
		Post(a.basePath + "/" + idPResource)

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusCreated {
		err = fmt.Errorf("error in create IdP, responce: %s", resp.String())
	}

	err = a.CreateCentralIdPMappers(realm, client)

	if err != nil {
		return err
	}

	reqLog.Info("End create central identity provider")
	return nil
}

func (a GoCloakAdapter) getCentralIdP(client dto.Client) api.IdentityProviderRepresentation {
	return api.IdentityProviderRepresentation{
		Alias:       "openshift",
		DisplayName: "EDP SSO",
		Enabled:     true,
		ProviderId:  "keycloak-oidc",
		Config: api.IdentityProviderConfig{
			UserInfoUrl:      a.basePath + "/auth/realms/openshift/protocol/openid-connect/userinfo",
			TokenUrl:         a.basePath + "/auth/realms/openshift/protocol/openid-connect/token",
			JwksUrl:          a.basePath + "/auth/realms/openshift/protocol/openid-connect/certs",
			Issuer:           a.basePath + "/auth/realms/openshift",
			AuthorizationUrl: a.basePath + "/auth/realms/openshift/protocol/openid-connect/auth",
			LogoutUrl:        a.basePath + "/auth/realms/openshift/protocol/openid-connect/logout",
			ClientId:         client.ClientId,
			ClientSecret:     client.ClientSecret,
		},
	}
}

func (a GoCloakAdapter) CreateCentralIdPMappers(realm dto.Realm, client dto.Client) error {
	reqLog := log.WithValues("realm", realm)
	reqLog.Info("Start create central IdP mappers...")

	err := a.createIdPMapper(realm, client.ClientId+".administrator", "administrator")
	if err != nil {
		return err
	}
	err = a.createIdPMapper(realm, client.ClientId+".developer", "developer")
	if err != nil {
		return err
	}
	err = a.createIdPMapper(realm, client.ClientId+".administrator", "realm-management.realm-admin")
	if err != nil {
		return err
	}

	reqLog.Info("End create central IdP mappers")
	return nil
}

func (a GoCloakAdapter) createIdPMapper(realm dto.Realm, externalRole string, role string) error {
	body := getIdPMapper(externalRole, role)
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realm.Name,
			"alias": "openshift",
		}).
		SetBody(body).
		Post(a.basePath + "/" + idPMapperResource)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("error in creation idP mapper by name %s", body.Name)
	}
	return nil
}

func getIdPMapper(externalRole, role string) api.IdentityProviderMapperRepresentation {
	return api.IdentityProviderMapperRepresentation{
		Config: map[string]string{
			"external.role": externalRole,
			"role":          role,
		},
		IdentityProviderAlias:  "openshift",
		IdentityProviderMapper: "keycloak-oidc-role-to-role-idp-mapper",
		Name:                   role,
	}
}

func getDefaultRealm(realmName string) gocloak.RealmRepresentation {
	return gocloak.RealmRepresentation{
		Realm:        realmName,
		Enabled:      true,
		DefaultRoles: []string{"developer"},
		Roles: map[string][]map[string]interface{}{
			"realm": {
				{
					"name": "administrator",
				},
				{
					"name": "developer",
				},
			},
		},
	}
}

func strip404(in error) (bool, error) {
	if in == nil {
		return true, nil
	}
	if is404(in) {
		return false, nil
	}
	return false, in
}

func is404(e error) bool {
	return strings.Contains(e.Error(), "404")
}
