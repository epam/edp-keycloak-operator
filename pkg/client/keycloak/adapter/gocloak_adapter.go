package adapter

import (
	"fmt"
	"github.com/Nerzal/gocloak"
	"github.com/epmd-edp/keycloak-operator/pkg/apis/v1/v1alpha1"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/api"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
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

	err := a.client.CreateRealm(a.token.AccessToken, getDefaultRealm(realm.Name, realm.Users))
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
		Get(a.basePath + getOneIdP)

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
		Post(a.basePath + idPResource)

	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusCreated {
		reqLog.Info("requested url", "url", resp.Request.URL)
		return fmt.Errorf("error in create IdP, responce status: %s", resp.Status())
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
		Post(a.basePath + idPMapperResource)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("error in creation idP mapper by name %s", body.Name)
	}
	return nil
}

func (a GoCloakAdapter) ExistClient(client dto.Client) (*bool, error) {
	reqLog := log.WithValues("client dto", client)
	reqLog.Info("Start check client in Keycloak...")

	clns, err := a.client.GetClients(a.token.AccessToken, client.RealmName, gocloak.GetClientsParams{
		ClientID: client.ClientId,
	})

	if err != nil {
		return nil, err
	}

	res := checkFullNameMatch(client, clns)

	reqLog.Info("End check client in Keycloak")
	return &res, nil
}

func checkFullNameMatch(client dto.Client, clients *[]gocloak.Client) bool {
	if clients == nil {
		return false
	}
	for _, el := range *clients {
		if el.ClientID == client.ClientId {
			return true
		}
	}
	return false
}

func (a GoCloakAdapter) CreateClient(client dto.Client) error {
	reqLog := log.WithValues("client dto", client)
	reqLog.Info("Start create client in Keycloak...")

	err := a.client.CreateClient(a.token.AccessToken, client.RealmName, gocloak.Client{
		ClientID: client.ClientId,
		Secret:   client.ClientSecret,
	})
	if err != nil {
		return err
	}

	reqLog.Info("Keycloak client has been created")
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

func getDefaultRealm(realmName string, users []v1alpha1.User) gocloak.RealmRepresentation {
	realmRepr := gocloak.RealmRepresentation{
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

	for _, user := range users {
		realmRepr.Users = append(realmRepr.Users, user)
	}

	return realmRepr
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

func (a GoCloakAdapter) ExistRealmRole(realm dto.Realm, role dto.RealmRole) (*bool, error) {
	reqLog := log.WithValues("realm dto", realm, "role dto", role)
	reqLog.Info("Start check existing realm role...")

	_, err := a.client.GetRealmRole(a.token.AccessToken, realm.Name, role.Name)

	res, err := strip404(err)

	if err != nil {
		return nil, err
	}

	reqLog.Info("Check existing realm role has been finished", "result", res)
	return &res, nil
}

func (a GoCloakAdapter) CreateRealmRole(realm dto.Realm, role dto.RealmRole) error {
	reqLog := log.WithValues("realm dto", realm, "role", role)
	reqLog.Info("Start create realm roles in Keycloak...")

	realmRole := gocloak.Role{
		Name: role.Name,
	}
	err := a.client.CreateRealmRole(a.token.AccessToken, realm.Name, realmRole)
	if err != nil {
		return err
	}

	persRole, err := a.client.GetRealmRole(a.token.AccessToken, realm.Name, role.Name)
	if err != nil {
		return err
	}

	err = a.client.AddRealmRoleComposite(a.token.AccessToken, realm.Name,
		role.Composite, []gocloak.Role{*persRole})

	if err != nil {
		return err
	}

	reqLog.Info("Keycloak roles has been created")
	return nil
}
