package adapter

import (
	"fmt"
	"gopkg.in/nerzal/gocloak.v2"
	"keycloak-operator/pkg/apis/v1/v1alpha1"
	"keycloak-operator/pkg/client/keycloak/api"
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

func (a GoCloakAdapter) ExistRealm(spec v1alpha1.KeycloakRealmSpec) (*bool, error) {
	reqLog := log.WithValues("realm spec", spec)
	reqLog.Info("Start check existing realm...")

	_, err := a.client.GetRealm(a.token.AccessToken, spec.RealmName)

	res, err := strip404(err)

	if err != nil {
		return nil, err
	}

	reqLog.Info("Check existing realm has been finished", "result", res)
	return &res, nil
}

func (a GoCloakAdapter) CreateRealmWithDefaultConfig(spec v1alpha1.KeycloakRealmSpec) error {
	reqLog := log.WithValues("realm spec", spec)
	reqLog.Info("Start creating realm with default config...")

	err := a.client.CreateRealm(a.token.AccessToken, getDefaultRealm(spec.RealmName))
	if err != nil {
		return err
	}

	reqLog.Info("End creating realm with default config")
	return nil
}

func (a GoCloakAdapter) ExistCentralIdentityProvider(spec v1alpha1.KeycloakRealmSpec) (*bool, error) {
	reqLog := log.WithValues("spec")
	reqLog.Info("Start check central identity provider in realm")

	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": spec.RealmName,
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

func (a GoCloakAdapter) CreateCentralIdentityProvider(rSpec v1alpha1.KeycloakRealmSpec, cSpec v1alpha1.KeycloakClientSpec) error {
	reqLog := log.WithValues("realm spec", rSpec, "keycloak client spec", cSpec)
	reqLog.Info("Start create central identity provider...")

	idP := a.getCentralIdP(cSpec)

	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": rSpec.RealmName,
		}).
		SetBody(idP).
		Post(a.basePath + "/" + idPResource)

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusCreated {
		err = fmt.Errorf("error in create IdP, responce: %s", resp.String())
	}

	err = a.CreateCentralIdPMappers(rSpec, cSpec)

	if err != nil {
		return err
	}

	reqLog.Info("End create central identity provider")
	return nil
}

func (a GoCloakAdapter) getCentralIdP(cSpec v1alpha1.KeycloakClientSpec) api.IdentityProviderRepresentation {
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
			ClientId:         cSpec.ClientId,
			ClientSecret:     cSpec.ClientSecret,
		},
	}
}

func (a GoCloakAdapter) CreateCentralIdPMappers(rSpec v1alpha1.KeycloakRealmSpec, cSpec v1alpha1.KeycloakClientSpec) error {
	reqLog := log.WithValues("realm spec", rSpec)
	reqLog.Info("Start create central IdP mappers...")

	err := a.createIdPMapper(rSpec, cSpec.ClientId+".administrator", "administrator")
	if err != nil {
		return err
	}
	err = a.createIdPMapper(rSpec, cSpec.ClientId+".developer", "developer")
	if err != nil {
		return err
	}
	err = a.createIdPMapper(rSpec, cSpec.ClientId+".administrator", "realm-management.realm-admin")
	if err != nil {
		return err
	}

	reqLog.Info("End create central IdP mappers")
	return nil
}

func (a GoCloakAdapter) createIdPMapper(rSpec v1alpha1.KeycloakRealmSpec, externalRole string, role string) error {
	body := getIdPMapper(externalRole, role)
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": rSpec.RealmName,
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
