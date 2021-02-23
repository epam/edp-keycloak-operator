package adapter

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Nerzal/gocloak"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/api"
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epmd-edp/keycloak-operator/pkg/consts"
	"github.com/epmd-edp/keycloak-operator/pkg/model"
	"gopkg.in/resty.v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	idPResource              = "/auth/admin/realms/{realm}/identity-provider/instances"
	idPMapperResource        = "/auth/admin/realms/{realm}/identity-provider/instances/{alias}/mappers"
	clientRoleMapperResource = "/auth/admin/realms/{realm}/users/{user}/role-mappings/clients/{client}"
	getOneIdP                = idPResource + "/{alias}"
	openIdConfig             = "/auth/realms/{realm}/.well-known/openid-configuration"
	authExecutions           = "/auth/admin/realms/{realm}/authentication/flows/browser/executions"
	authExecutionConfig      = "/auth/admin/realms/{realm}/authentication/executions/{id}/config"
	postClientScopeMapper    = "/auth/admin/realms/{realm}/client-scopes/{scopeId}/protocol-mappers/models"
	getOneClientScope        = "/auth/admin/realms/{realm}/client-scopes"
	linkClientScopeToClient  = "/auth/admin/realms/{realm}/clients/{clientId}/default-client-scopes/{scopeId}"
	postClientScope          = "/auth/admin/realms/{realm}/client-scopes"
)

var (
	log               = logf.Log.WithName("gocloak_adapter")
	AcCreatorUsername = "ac-creator"
	AcReaderUsername  = "ac-reader"
)

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

func (a GoCloakAdapter) DeleteRealm(realmName string) error {
	reqLog := log.WithValues("realm", realmName)
	reqLog.Info("Start deleting realm...")

	if err := a.client.DeleteRealm(a.token.AccessToken, realmName); err != nil {
		return err
	}

	reqLog.Info("End deletion realm")
	return nil
}

func (a GoCloakAdapter) CreateRealmWithDefaultConfig(realm dto.Realm) error {
	reqLog := log.WithValues("realm", realm)
	reqLog.Info("Start creating realm with default config...")

	err := a.client.CreateRealm(a.token.AccessToken, getDefaultRealm(realm))
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
			"alias": realm.SsoRealmName,
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

	idP := a.getCentralIdP(client, realm.SsoRealmName)

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

func (a GoCloakAdapter) getCentralIdP(client dto.Client, ssoRealmName string) api.IdentityProviderRepresentation {
	return api.IdentityProviderRepresentation{
		Alias:       ssoRealmName,
		DisplayName: "EDP SSO",
		Enabled:     true,
		ProviderId:  "keycloak-oidc",
		Config: api.IdentityProviderConfig{
			UserInfoUrl:      fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/userinfo", a.basePath, ssoRealmName),
			TokenUrl:         fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/token", a.basePath, ssoRealmName),
			JwksUrl:          fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/certs", a.basePath, ssoRealmName),
			Issuer:           fmt.Sprintf("%s/auth/realms/%s", a.basePath, ssoRealmName),
			AuthorizationUrl: fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/auth", a.basePath, ssoRealmName),
			LogoutUrl:        fmt.Sprintf("%s/auth/realms/%s/protocol/openid-connect/logout", a.basePath, ssoRealmName),
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
	body := getIdPMapper(externalRole, role, realm.SsoRealmName)
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realm.Name,
			"alias": realm.SsoRealmName,
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

func (a GoCloakAdapter) ExistClientRole(client dto.Client, clientRole string) (*bool, error) {
	reqLog := log.WithValues("client dto", client, "client role", clientRole)
	reqLog.Info("Start check client role in Keycloak...")

	id, err := a.GetClientId(client)
	if err != nil {
		return nil, err
	}

	clnr, err := a.client.GetClientRoles(a.token.AccessToken, client.RealmName, *id)

	_, err = strip404(err)

	if err != nil {
		return nil, err
	}

	res := checkFullClientRoleNameMatch(clientRole, clnr)

	reqLog.Info("End check client role in Keycloak", "result", res)
	return &res, nil
}

func (a GoCloakAdapter) CreateClientRole(client dto.Client, clientRole string) error {
	reqLog := log.WithValues("client dto", client, "client role", clientRole)
	reqLog.Info("Start create client role in Keycloak...")

	id, err := a.GetClientId(client)
	if err != nil {
		return err
	}

	err = a.client.CreateClientRole(a.token.AccessToken, client.RealmName, *id, gocloak.Role{
		Name:       clientRole,
		ClientRole: true,
	})
	if err != nil {
		return err
	}

	reqLog.Info("Keycloak client role has been created")
	return nil
}

func checkFullClientRoleNameMatch(clientRole string, roles *[]gocloak.Role) bool {
	if roles == nil {
		return false
	}

	for _, cl := range *roles {
		if cl.Name == clientRole {
			return true
		}
	}
	return false
}

func checkFullUsernameMatch(userName string, users *[]gocloak.User) bool {
	if users == nil {
		return false
	}
	for _, el := range *users {
		if el.Username == userName {
			return true
		}
	}
	return false
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

func (a GoCloakAdapter) DeleteClient(kkClientID string, client dto.Client) error {
	reqLog := log.WithValues("client dto", client)
	reqLog.Info("Start delete client in Keycloak...")

	if err := a.client.DeleteClient(a.token.AccessToken, client.RealmName, kkClientID); err != nil {
		return err
	}

	reqLog.Info("Keycloak client has been deleted")
	return nil
}

func (a GoCloakAdapter) CreateClient(client dto.Client) error {
	reqLog := log.WithValues("client dto", client)
	reqLog.Info("Start create client in Keycloak...")

	err := a.client.CreateClient(a.token.AccessToken, client.RealmName, getGclCln(client))
	if err != nil {
		return err
	}

	reqLog.Info("Keycloak client has been created")
	return nil
}

func getGclCln(client dto.Client) gocloak.Client {
	protocolMappers := getProtocolMappers(client.AdvancedProtocolMappers)
	return gocloak.Client{
		ClientID:                  client.ClientId,
		Secret:                    client.ClientSecret,
		PublicClient:              client.Public,
		DirectAccessGrantsEnabled: client.DirectAccess,
		RootURL:                   client.WebUrl,
		Protocol:                  client.Protocol,
		Attributes:                client.Attributes,
		RedirectURIs: []string{
			client.WebUrl + "/*",
		},
		WebOrigins: []string{
			client.WebUrl,
		},
		AdminURL:        client.WebUrl,
		ProtocolMappers: protocolMappers,
	}
}

func getProtocolMappers(need bool) []gocloak.ProtocolMapperRepresentation {
	if !need {
		return nil
	}
	return []gocloak.ProtocolMapperRepresentation{
		{
			Name:           "username",
			Protocol:       "openid-connect",
			ProtocolMapper: "oidc-usermodel-property-mapper",
			Config: map[string]string{
				"userinfo.token.claim": "true",
				"user.attribute":       "username",
				"id.token.claim":       "true",
				"access.token.claim":   "true",
				"claim.name":           "preferred_username",
				"jsonType.label":       "String",
			},
		},
		{
			Name:           "realm roles",
			Protocol:       "openid-connect",
			ProtocolMapper: "oidc-usermodel-realm-role-mapper",
			Config: map[string]string{
				"userinfo.token.claim": "true",
				"multivalued":          "true",
				"id.token.claim":       "true",
				"access.token.claim":   "false",
				"claim.name":           "roles",
				"jsonType.label":       "String",
			},
		},
	}
}

func (a GoCloakAdapter) GetClientId(client dto.Client) (*string, error) {
	clients, err := a.client.GetClients(a.token.AccessToken, client.RealmName, gocloak.GetClientsParams{
		ClientID: client.ClientId,
	})
	if err != nil {
		return nil, err
	}

	for _, item := range *clients {
		if item.ClientID == client.ClientId {
			return &item.ID, nil
		}
	}
	return nil, fmt.Errorf("unable to get Client ID. Client %v doesn't exist", client.ClientId)
}

func getIdPMapper(externalRole, role, ssoRealmName string) api.IdentityProviderMapperRepresentation {
	return api.IdentityProviderMapperRepresentation{
		Config: map[string]string{
			"external.role": externalRole,
			"role":          role,
		},
		IdentityProviderAlias:  ssoRealmName,
		IdentityProviderMapper: "keycloak-oidc-role-to-role-idp-mapper",
		Name:                   role,
	}
}

func (a GoCloakAdapter) CreateRealmUser(realmName string, user dto.User) error {
	reqLog := log.WithValues("user dto", user, "realm", realmName)
	reqLog.Info("Start create realm user in Keycloak...")

	userDto := gocloak.User{
		Username: user.Username,
		Email:    user.Username,
		Enabled:  true,
	}

	_, err := a.client.CreateUser(a.token.AccessToken, realmName, userDto)
	if err != nil {
		return err
	}

	reqLog.Info("Keycloak realm user has been created")
	return nil
}

func (a GoCloakAdapter) ExistRealmUser(realmName string, user dto.User) (*bool, error) {
	reqLog := log.WithValues("user dto", user, "realm", realmName)
	reqLog.Info("Start check user in Keycloak realm...")

	usr, err := a.client.GetUsers(a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: user.Username,
	})

	_, err = strip404(err)

	if err != nil {
		return nil, err
	}

	res := checkFullUsernameMatch(user.Username, usr)

	reqLog.Info("End check user in Keycloak", "result", res)
	return &res, nil
}

func (a GoCloakAdapter) HasUserClientRole(realmName string, clientId string, user dto.User, role string) (*bool, error) {
	reqLog := log.WithValues("role", role, "client", clientId, "realm", realmName, "user dto", user)
	reqLog.Info("Start check user roles in Keycloak realm...")

	users, err := a.client.GetUsers(a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: user.Username,
	})
	if err != nil {
		return nil, err
	}
	if len(*users) == 0 {
		return nil, fmt.Errorf("no such user %v has been found", user.Username)
	}

	rolesMapping, err := a.client.GetRoleMappingByUserID(a.token.AccessToken, realmName, (*users)[0].ID)
	if err != nil {
		return nil, err
	}

	clientRoles := rolesMapping.ClientMappings[clientId].Mappings

	res := checkFullClientRoleNameMatch(role, &clientRoles)

	reqLog.Info("End check user role in Keycloak", "result", res)
	return &res, nil
}

func (a GoCloakAdapter) AddClientRoleToUser(realmName string, clientId string, user dto.User, roleName string) error {
	reqLog := log.WithValues("role", roleName, "realm", realmName, "user", user.Username)
	reqLog.Info("Start mapping realm role to user in Keycloak...")

	client, err := a.client.GetClients(a.token.AccessToken, realmName, gocloak.GetClientsParams{
		ClientID: clientId,
	})
	if len(*client) == 0 {
		return fmt.Errorf("no such client %v has been found", clientId)
	}
	if err != nil {
		return err
	}

	role, err := a.client.GetClientRole(a.token.AccessToken, realmName, (*client)[0].ID, roleName)
	if err != nil {
		return err
	}
	if role == nil {
		return fmt.Errorf("no such client role %v has been found", roleName)
	}

	users, err := a.client.GetUsers(a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: user.Username,
	})

	err = a.addClientRoleToUser(realmName, (*users)[0].ID, []gocloak.Role{*role})
	if err != nil {
		return err
	}

	reqLog.Info("Role to user has been added")
	return nil
}

func (a GoCloakAdapter) addClientRoleToUser(realmName string, userId string, roles []gocloak.Role) error {
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm":  realmName,
			"user":   userId,
			"client": roles[0].ContainerID,
		}).
		SetBody(roles).
		Post(a.basePath + clientRoleMapperResource)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusNoContent {
		return fmt.Errorf("error in mapping client role to user %v", roles)
	}
	return nil
}

func getDefaultRealm(realm dto.Realm) gocloak.RealmRepresentation {
	aConf := make([]interface{}, 0)
	if realm.SsoRealmEnabled {
		aConf = append(aConf, map[string]interface{}{
			"alias": "edp sso",
			"config": map[string]string{
				"defaultProvider": realm.SsoRealmName,
			},
		})
	}
	rr := gocloak.RealmRepresentation{
		Realm:        realm.Name,
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
		Users: getDefUsers(realm),
	}
	return rr
}

func getDefUsers(realm dto.Realm) []interface{} {
	users := make([]interface{}, 0)
	users = append(users, map[string]interface{}{
		"username":  AcReaderUsername,
		"enabled":   true,
		"email":     "admin-console-reader@example.com",
		"firstName": "Reader",
		"lastName":  "EDP",
		"credentials": []map[string]string{{
			"type":  "password",
			"value": realm.ACReaderPass,
		},
		},
		"realmRoles": []string{"developer"},
	})
	users = append(users, map[string]interface{}{
		"username":  AcCreatorUsername,
		"enabled":   true,
		"email":     "admin-console-creator@example.com",
		"firstName": "Administrator",
		"lastName":  "EDP",
		"credentials": []map[string]string{{
			"type":  "password",
			"value": realm.ACCreatorPass,
		},
		},
		"realmRoles": []string{"administrator"},
	})
	return users
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

func (a GoCloakAdapter) GetOpenIdConfig(realm dto.Realm) (*string, error) {
	reqLog := log.WithValues("realm dto", realm)
	reqLog.Info("Start get openid configuration...")

	resp, err := a.client.RestyClient().R().
		SetPathParams(map[string]string{
			"realm": realm.Name,
		}).
		Get(a.basePath + openIdConfig)
	if err != nil {
		return nil, err
	}
	res := resp.String()

	reqLog.Info("End get openid configuration", "result", res)
	return &res, nil
}

func (a GoCloakAdapter) PutDefaultIdp(realm dto.Realm) error {
	reqLog := log.WithValues("realm dto", realm)
	reqLog.Info("Start put default IdP...")

	eId, err := a.getIdPRedirectExecutionId(realm)
	if err != nil {
		return err
	}
	err = a.createRedirectConfig(realm, *eId)
	if err != nil {
		return err
	}
	reqLog.Info("Default IdP was successfully configured!")
	return nil
}

func (a GoCloakAdapter) getIdPRedirectExecutionId(realm dto.Realm) (*string, error) {
	exs, err := a.getBrowserExecutions(realm)
	if err != nil {
		return nil, err
	}
	ex := getIdPRedirector(exs)
	return &ex.Id, nil
}

func getIdPRedirector(executions []api.SimpleAuthExecution) *api.SimpleAuthExecution {
	for _, ex := range executions {
		if ex.ProviderId == "identity-provider-redirector" {
			return &ex
		}
	}
	return nil
}

func (a GoCloakAdapter) createRedirectConfig(realm dto.Realm, eId string) error {
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realm.Name,
			"id":    eId,
		}).
		SetBody(map[string]interface{}{
			"alias": "edp-sso",
			"config": map[string]string{
				"defaultProvider": realm.SsoRealmName,
			},
		}).
		Post(a.basePath + authExecutionConfig)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 201 {
		return fmt.Errorf("response is not ok by create redirect config: Status: %v", resp.Status())
	}
	return nil
}

func (a GoCloakAdapter) getBrowserExecutions(realm dto.Realm) ([]api.SimpleAuthExecution, error) {
	res := make([]api.SimpleAuthExecution, 0)
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realm.Name,
		}).
		SetResult(&res).
		Get(a.basePath + authExecutions)
	if err != nil {
		return res, err
	}
	if resp.StatusCode() != 200 {
		return res, fmt.Errorf("response is not ok by get browser executions: Status: %v", resp.Status())
	}
	return res, nil
}

func (a GoCloakAdapter) PutClientScopeMapper(clientName, scopeId, realmName string) error {
	reqLog := log.WithValues("scopeId", scopeId, "realm", realmName, "clientId", clientName)
	reqLog.Info("Start put Client Scope mapper...")
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm":   realmName,
			"scopeId": scopeId,
		}).
		SetBody(getProtocolMapper(clientName)).
		Post(a.basePath + postClientScopeMapper)
	if err := checkError(err, resp); err != nil {
		return err
	}
	reqLog.Info("Client Scope mapper was successfully configured!")
	return nil
}

func checkError(err error, response *resty.Response) error {
	if err != nil {
		return err
	}
	if response == nil {
		return errors.New("empty response")
	}
	if response.IsError() {
		if response.StatusCode() == 409 {
			log.Info("entity already exists. creating skipped", "url", response.Request.URL)
			return nil
		}
		return errors.New(response.Status())
	}
	return nil
}

func getProtocolMapper(clientId string) model.ProtocolMappers {
	return model.ProtocolMappers{
		Name:           stringP(fmt.Sprintf("%v-%v", clientId, "audience")),
		Protocol:       stringP(consts.OpenIdProtocol),
		ProtocolMapper: stringP(consts.ProtocolMapper),
		ProtocolMappersConfig: &model.ProtocolMappersConfig{
			AccessTokenClaim:       stringP("true"),
			IncludedClientAudience: stringP(clientId),
		},
	}
}

func (a GoCloakAdapter) GetClientScope(scopeName, realmName string) (*model.ClientScope, error) {
	reqLog := log.WithValues("scopeName", scopeName, "realm", realmName)
	reqLog.Info("Start get Client Scope...")
	var result []*model.ClientScope
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetResult(&result).
		Get(a.basePath + getOneClientScope)
	if err := checkError(err, resp); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("realm %v doesnt contain client scopes", realmName)
	}
	scope, err := getClientScope(scopeName, result)
	if err != nil {
		return nil, err
	}
	reqLog.Info("End get Client Scope", "scope", scope)
	return scope, err
}

func getClientScope(name string, clientScopes []*model.ClientScope) (*model.ClientScope, error) {
	for _, cs := range clientScopes {
		if *cs.Name == name {
			return cs, nil
		}
	}
	return nil, fmt.Errorf("scope %v was not found", name)
}

func stringP(value string) *string {
	return &value
}

func (a GoCloakAdapter) LinkClientScopeToClient(clientName, scopeId, realmName string) error {
	reqLog := log.WithValues("clientName", clientName, "scopeId", scopeId, "realm", realmName)
	reqLog.Info("Start link Client Scope to client...")
	clientId, err := a.GetClientId(dto.Client{ClientId: clientName, RealmName: realmName})
	if err != nil {
		return err
	}

	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm":    realmName,
			"clientId": *clientId,
			"scopeId":  scopeId,
		}).
		Put(a.basePath + linkClientScopeToClient)
	if err := checkError(err, resp); err != nil {
		return err
	}
	reqLog.Info("End link Client Scope to client...")
	return nil
}

func (a GoCloakAdapter) CreateClientScope(realmName string, scope model.ClientScope) error {
	reqLog := log.WithValues("realm", realmName, "scope", scope.Name)
	reqLog.Info("Start creating Client Scope...")
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetBody(scope).
		Post(a.basePath + postClientScope)
	if err := checkError(err, resp); err != nil {
		return err
	}
	reqLog.Info("Client Scope was created!")
	return nil
}
