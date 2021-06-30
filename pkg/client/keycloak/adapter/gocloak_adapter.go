package adapter

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-logr/logr"

	"github.com/Nerzal/gocloak/v8"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/api"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
	"github.com/epam/edp-keycloak-operator/pkg/consts"
	"github.com/epam/edp-keycloak-operator/pkg/model"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
)

const (
	idPResource                    = "/auth/admin/realms/{realm}/identity-provider/instances"
	idPMapperResource              = "/auth/admin/realms/{realm}/identity-provider/instances/{alias}/mappers"
	getOneIdP                      = idPResource + "/{alias}"
	openIdConfig                   = "/auth/realms/{realm}/.well-known/openid-configuration"
	authExecutions                 = "/auth/admin/realms/{realm}/authentication/flows/browser/executions"
	authExecutionConfig            = "/auth/admin/realms/{realm}/authentication/executions/{id}/config"
	postClientScopeMapper          = "/auth/admin/realms/{realm}/client-scopes/{scopeId}/protocol-mappers/models"
	getOneClientScope              = "/auth/admin/realms/{realm}/client-scopes"
	linkClientScopeToClient        = "/auth/admin/realms/{realm}/clients/{clientId}/default-client-scopes/{scopeId}"
	postClientScope                = "/auth/admin/realms/{realm}/client-scopes"
	getClientProtocolMappers       = "/auth/admin/realms/{realm}/clients/{id}/protocol-mappers/models"
	mapperToIdentityProvider       = "/auth/admin/realms/{realm}/identity-provider/instances/{alias}/mappers"
	updateMapperToIdentityProvider = "/auth/admin/realms/{realm}/identity-provider/instances/{alias}/mappers/{id}"
	authFlows                      = "/auth/admin/realms/{realm}/authentication/flows"
	authFlow                       = "/auth/admin/realms/{realm}/authentication/flows/{id}"
	authFlowExecutionCreate        = "/auth/admin/realms/{realm}/authentication/executions"
	authFlowExecutionConfig        = "/auth/admin/realms/{realm}/authentication/executions/{id}/config"
)

type GoCloakAdapter struct {
	client   GoCloak
	token    *gocloak.JWT
	log      logr.Logger
	basePath string
}

func Make(url, user, password string, log logr.Logger) (*GoCloakAdapter, error) {
	kcCl := gocloak.NewClient(url)
	token, err := kcCl.LoginAdmin(context.Background(), user, password, "master")
	if err != nil {
		return nil, errors.Wrapf(err, "cannot login to keycloak server with user: %s", user)
	}

	return &GoCloakAdapter{
		client:   kcCl,
		token:    token,
		log:      log,
		basePath: url,
	}, nil
}

func (a GoCloakAdapter) ExistCentralIdentityProvider(realm *dto.Realm) (bool, error) {
	log := a.log.WithValues("realm", realm)
	log.Info("Start check central identity provider in realm")

	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realm.Name,
			"alias": realm.SsoRealmName,
		}).
		Get(a.basePath + getOneIdP)

	if err != nil {
		return false, err
	}
	if resp.StatusCode() == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode() != http.StatusOK {
		return false, errors.Errorf("errors in get idP, response: %s", resp.String())
	}

	log.Info("End check central identity provider in realm")
	return true, nil
}

func (a GoCloakAdapter) CreateCentralIdentityProvider(realm *dto.Realm, client *dto.Client) error {
	log := a.log.WithValues("realm", realm, "keycloak client", client)
	log.Info("Start create central identity provider...")

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
		log.Info("requested url", "url", resp.Request.URL)
		return fmt.Errorf("error in create IdP, responce status: %s", resp.Status())
	}

	err = a.CreateCentralIdPMappers(realm, client)

	if err != nil {
		return err
	}

	log.Info("End create central identity provider")
	return nil
}

func (a GoCloakAdapter) getCentralIdP(client *dto.Client, ssoRealmName string) api.IdentityProviderRepresentation {
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

func (a GoCloakAdapter) CreateCentralIdPMappers(realm *dto.Realm, client *dto.Client) error {
	log := a.log.WithValues("realm", realm)
	log.Info("Start create central IdP mappers...")

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

	log.Info("End create central IdP mappers")
	return nil
}

func (a GoCloakAdapter) createIdPMapper(realm *dto.Realm, externalRole string, role string) error {
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

func (a GoCloakAdapter) ExistClient(clientID, realm string) (bool, error) {
	log := a.log.WithValues("clientID", clientID, "realm", realm)
	log.Info("Start check client in Keycloak...")

	clns, err := a.client.GetClients(context.Background(), a.token.AccessToken, realm, gocloak.GetClientsParams{
		ClientID: &clientID,
	})

	if err != nil {
		return false, err
	}

	res := checkFullNameMatch(clientID, clns)

	log.Info("End check client in Keycloak")
	return res, nil
}

func (a GoCloakAdapter) ExistClientRole(client *dto.Client, clientRole string) (bool, error) {
	log := a.log.WithValues("client dto", client, "client role", clientRole)
	log.Info("Start check client role in Keycloak...")

	id, err := a.GetClientID(client.ClientId, client.RealmName)
	if err != nil {
		return false, err
	}

	clientRoles, err := a.client.GetClientRoles(context.Background(), a.token.AccessToken, client.RealmName, id)
	_, err = strip404(err)
	if err != nil {
		return false, err
	}

	res := false
	for _, cl := range clientRoles {
		if cl.Name != nil && *cl.Name == clientRole {
			res = true
			break
		}
	}

	log.Info("End check client role in Keycloak", "result", res)
	return res, nil
}

func (a GoCloakAdapter) CreateClientRole(client *dto.Client, clientRole string) error {
	log := a.log.WithValues("client dto", client, "client role", clientRole)
	log.Info("Start create client role in Keycloak...")

	id, err := a.GetClientID(client.ClientId, client.RealmName)
	if err != nil {
		return err
	}

	if _, err = a.client.CreateClientRole(context.Background(), a.token.AccessToken, client.RealmName, id, gocloak.Role{
		Name:       &clientRole,
		ClientRole: gocloak.BoolP(true),
	}); err != nil {
		return errors.Wrap(err, "unable to create client role")
	}

	log.Info("Keycloak client role has been created")
	return nil
}

func checkFullRoleNameMatch(role string, roles *[]gocloak.Role) bool {
	if roles == nil {
		return false
	}

	for _, cl := range *roles {
		if cl.Name != nil && *cl.Name == role {
			return true
		}
	}
	return false
}

func checkFullUsernameMatch(userName string, users []*gocloak.User) bool {
	if users == nil {
		return false
	}
	for _, el := range users {
		if el.Username != nil && *el.Username == userName {
			return true
		}
	}
	return false
}

func checkFullNameMatch(clientID string, clients []*gocloak.Client) bool {
	if clients == nil {
		return false
	}

	for _, el := range clients {
		if el.ClientID != nil && *el.ClientID == clientID {
			return true
		}
	}

	return false
}

func (a GoCloakAdapter) DeleteClient(kkClientID, realmName string) error {
	log := a.log.WithValues("client id", kkClientID)
	log.Info("Start delete client in Keycloak...")

	if err := a.client.DeleteClient(context.Background(), a.token.AccessToken, realmName, kkClientID); err != nil {
		return errors.Wrap(err, "unable to delete client")
	}

	log.Info("Keycloak client has been deleted")
	return nil
}

func (a GoCloakAdapter) CreateClient(client *dto.Client) error {
	log := a.log.WithValues("client dto", client)
	log.Info("Start create client in Keycloak...")

	_, err := a.client.CreateClient(context.Background(), a.token.AccessToken, client.RealmName, getGclCln(client))
	if err != nil {
		return err
	}

	log.Info("Keycloak client has been created")
	return nil
}

func getGclCln(client *dto.Client) gocloak.Client {
	//TODO: check collision with protocol mappers list in spec
	protocolMappers := getProtocolMappers(client.AdvancedProtocolMappers)

	return gocloak.Client{
		ClientID:                  &client.ClientId,
		Secret:                    &client.ClientSecret,
		PublicClient:              &client.Public,
		DirectAccessGrantsEnabled: &client.DirectAccess,
		RootURL:                   &client.WebUrl,
		Protocol:                  &client.Protocol,
		Attributes:                &client.Attributes,
		RedirectURIs: &[]string{
			client.WebUrl + "/*",
		},
		WebOrigins: &[]string{
			client.WebUrl,
		},
		AdminURL:               &client.WebUrl,
		ProtocolMappers:        &protocolMappers,
		ServiceAccountsEnabled: &client.ServiceAccountEnabled,
	}
}

func getProtocolMappers(need bool) []gocloak.ProtocolMapperRepresentation {
	if !need {
		return nil
	}

	return []gocloak.ProtocolMapperRepresentation{
		{
			Name:           gocloak.StringP("username"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-property-mapper"),
			Config: &map[string]string{
				"userinfo.token.claim": "true",
				"user.attribute":       "username",
				"id.token.claim":       "true",
				"access.token.claim":   "true",
				"claim.name":           "preferred_username",
				"jsonType.label":       "String",
			},
		},
		{
			Name:           gocloak.StringP("realm roles"),
			Protocol:       gocloak.StringP("openid-connect"),
			ProtocolMapper: gocloak.StringP("oidc-usermodel-realm-role-mapper"),
			Config: &map[string]string{
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

func (a GoCloakAdapter) GetClientID(clientID, realm string) (string, error) {
	clients, err := a.client.GetClients(context.Background(), a.token.AccessToken, realm,
		gocloak.GetClientsParams{
			ClientID: &clientID,
		})
	if err != nil {
		return "", err
	}

	for _, item := range clients {
		if item.ClientID != nil && *item.ClientID == clientID {
			return *item.ID, nil
		}
	}
	return "", fmt.Errorf("unable to get Client ID. Client %v doesn't exist", clientID)
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

func (a GoCloakAdapter) CreateRealmUser(realmName string, user *dto.User) error {
	log := a.log.WithValues("user dto", user, "realm", realmName)
	log.Info("Start create realm user in Keycloak...")

	userDto := gocloak.User{
		Username: &user.Username,
		Email:    &user.Username,
		Enabled:  gocloak.BoolP(true),
	}

	_, err := a.client.CreateUser(context.Background(), a.token.AccessToken, realmName, userDto)
	if err != nil {
		return err
	}

	log.Info("Keycloak realm user has been created")
	return nil
}

func (a GoCloakAdapter) ExistRealmUser(realmName string, user *dto.User) (bool, error) {
	log := a.log.WithValues("user dto", user, "realm", realmName)
	log.Info("Start check user in Keycloak realm...")

	usr, err := a.client.GetUsers(context.Background(), a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: &user.Username,
	})

	_, err = strip404(err)
	if err != nil {
		return false, err
	}

	res := checkFullUsernameMatch(user.Username, usr)

	log.Info("End check user in Keycloak", "result", res)
	return res, nil
}

func (a GoCloakAdapter) HasUserRealmRole(realmName string, user *dto.User, role string) (bool, error) {
	log := a.log.WithValues("role", role, "realm", realmName, "user dto", user)
	log.Info("Start check user roles in Keycloak realm...")

	users, err := a.client.GetUsers(context.Background(), a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: &user.Username,
	})
	if err != nil {
		return false, errors.Wrap(err, "unable to get users from keycloak")
	}
	if len(users) == 0 {
		return false, fmt.Errorf("no such user %v has been found", user.Username)
	}

	rolesMapping, err := a.client.GetRoleMappingByUserID(context.Background(), a.token.AccessToken, realmName,
		*users[0].ID)
	if err != nil {
		return false, errors.Wrap(err, "unable to GetRoleMappingByUserID")
	}

	res := checkFullRoleNameMatch(role, rolesMapping.RealmMappings)

	log.Info("End check user role in Keycloak", "result", res)
	return res, nil
}

func (a GoCloakAdapter) HasUserClientRole(realmName string, clientId string, user *dto.User, role string) (bool, error) {
	log := a.log.WithValues("role", role, "client", clientId, "realm", realmName, "user dto", user)
	log.Info("Start check user roles in Keycloak realm...")

	users, err := a.client.GetUsers(context.Background(), a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: &user.Username,
	})
	if err != nil {
		return false, err
	}
	if len(users) == 0 {
		return false, errors.Errorf("no such user %v has been found", user.Username)
	}

	rolesMapping, err := a.client.GetRoleMappingByUserID(context.Background(), a.token.AccessToken, realmName,
		*users[0].ID)
	if err != nil {
		return false, err
	}

	res := false
	if clientMap, ok := rolesMapping.ClientMappings[clientId]; ok && clientMap != nil && clientMap.Mappings != nil {
		res = checkFullRoleNameMatch(role, clientMap.Mappings)
	}

	log.Info("End check user role in Keycloak", "result", res)
	return res, nil
}

func (a GoCloakAdapter) AddRealmRoleToUser(realmName, username, roleName string) error {
	users, err := a.client.GetUsers(context.Background(), a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: &username,
	})
	if err != nil {
		return errors.Wrap(err, "error during get kc users")
	}
	if len(users) == 0 {
		return errors.Errorf("no users with username %s found", username)
	}

	rl, err := a.client.GetRealmRole(context.Background(), a.token.AccessToken, realmName, roleName)
	if err != nil {
		return errors.Wrap(err, "unable to get realm role from keycloak")
	}

	if err := a.client.AddRealmRoleToUser(context.Background(), a.token.AccessToken, realmName, *users[0].ID,
		[]gocloak.Role{
			*rl,
		}); err != nil {
		return errors.Wrap(err, "unable to add realm role to user")
	}

	return nil
}

func (a GoCloakAdapter) AddClientRoleToUser(realmName string, clientId string, user *dto.User, roleName string) error {
	log := a.log.WithValues("role", roleName, "realm", realmName, "user", user.Username)
	log.Info("Start mapping realm role to user in Keycloak...")

	client, err := a.client.GetClients(context.Background(), a.token.AccessToken, realmName, gocloak.GetClientsParams{
		ClientID: &clientId,
	})
	if err != nil {
		return err
	}
	if len(client) == 0 {
		return fmt.Errorf("no such client %v has been found", clientId)
	}

	role, err := a.client.GetClientRole(context.Background(), a.token.AccessToken, realmName, *client[0].ID, roleName)
	if err != nil {
		return errors.Wrap(err, "error during GetClientRole")
	}
	if role == nil {
		return errors.Errorf("no such client role %v has been found", roleName)
	}

	users, err := a.client.GetUsers(context.Background(), a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: &user.Username,
	})
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return fmt.Errorf("no such user %v has been found", user.Username)
	}

	err = a.addClientRoleToUser(realmName, *users[0].ID, []gocloak.Role{*role})
	if err != nil {
		return err
	}

	log.Info("Role to user has been added")
	return nil
}

func (a GoCloakAdapter) addClientRoleToUser(realmName string, userId string, roles []gocloak.Role) error {
	if err := a.client.AddClientRoleToUser(context.Background(), a.token.AccessToken, realmName,
		*roles[0].ContainerID, userId, roles); err != nil {
		return err
	}

	return nil
}

func getDefaultRealm(realm *dto.Realm) gocloak.RealmRepresentation {
	return gocloak.RealmRepresentation{
		Realm:   &realm.Name,
		Enabled: gocloak.BoolP(true),
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

func (a GoCloakAdapter) CreateIncludedRealmRole(realmName string, role *dto.IncludedRealmRole) error {
	log := a.log.WithValues("realm", realmName, "role", role)
	log.Info("Start create realm roles in Keycloak...")

	realmRole := gocloak.Role{
		Name: &role.Name,
	}

	_, err := a.client.CreateRealmRole(context.Background(), a.token.AccessToken, realmName, realmRole)
	if err != nil {
		return err
	}

	persRole, err := a.client.GetRealmRole(context.Background(), a.token.AccessToken, realmName, role.Name)
	if err != nil {
		return err
	}

	err = a.client.AddRealmRoleComposite(context.Background(), a.token.AccessToken, realmName,
		role.Composite, []gocloak.Role{*persRole})

	if err != nil {
		return err
	}

	log.Info("Keycloak roles has been created")
	return nil
}

func (a GoCloakAdapter) CreatePrimaryRealmRole(realmName string, role *dto.PrimaryRealmRole) error {
	log := a.log.WithValues("realm name", realmName, "role", role)
	log.Info("Start create realm roles in Keycloak...")

	realmRole := gocloak.Role{
		Name:        &role.Name,
		Description: &role.Description,
		Attributes:  &role.Attributes,
		Composite:   &role.IsComposite,
	}

	_, err := a.client.CreateRealmRole(context.Background(), a.token.AccessToken, realmName, realmRole)
	if err != nil {
		return errors.Wrap(err, "unable to create realm role")
	}

	if role.IsComposite && len(role.Composites) > 0 {
		compositeRoles := make([]gocloak.Role, 0, len(role.Composites))
		for _, composite := range role.Composites {
			role, err := a.client.GetRealmRole(context.Background(), a.token.AccessToken, realmName, composite)
			if err != nil {
				return errors.Wrap(err, "unable to get realm role")
			}
			compositeRoles = append(compositeRoles, *role)
		}

		if len(compositeRoles) > 0 {
			if err = a.client.AddRealmRoleComposite(context.Background(), a.token.AccessToken, realmName,
				role.Name, compositeRoles); err != nil {
				return errors.Wrap(err, "unable to add role composite")
			}
		}
	}

	log.Info("Keycloak roles has been created")
	return nil
}

func (a GoCloakAdapter) GetOpenIdConfig(realm *dto.Realm) (string, error) {
	log := a.log.WithValues("realm dto", realm)
	log.Info("Start get openid configuration...")

	resp, err := a.client.RestyClient().R().
		SetPathParams(map[string]string{
			"realm": realm.Name,
		}).
		Get(a.basePath + openIdConfig)
	if err != nil {
		return "", err
	}
	res := resp.String()

	log.Info("End get openid configuration", "result", res)
	return res, nil
}

func (a GoCloakAdapter) PutDefaultIdp(realm *dto.Realm) error {
	log := a.log.WithValues("realm dto", realm)
	log.Info("Start put default IdP...")

	eId, err := a.getIdPRedirectExecutionId(realm)
	if err != nil {
		return err
	}
	err = a.createRedirectConfig(realm, *eId)
	if err != nil {
		return err
	}
	log.Info("Default IdP was successfully configured!")
	return nil
}

func (a GoCloakAdapter) getIdPRedirectExecutionId(realm *dto.Realm) (*string, error) {
	exs, err := a.getBrowserExecutions(realm)
	if err != nil {
		return nil, err
	}
	ex, err := getIdPRedirector(exs)
	if err != nil {
		return nil, err
	}

	return &ex.Id, nil
}

func getIdPRedirector(executions []api.SimpleAuthExecution) (*api.SimpleAuthExecution, error) {
	for _, ex := range executions {
		if ex.ProviderId == "identity-provider-redirector" {
			return &ex, nil
		}
	}

	return nil, errors.New("identity provider not found")
}

func (a GoCloakAdapter) createRedirectConfig(realm *dto.Realm, eId string) error {
	resp, err := a.startRestyRequest().SetPathParams(map[string]string{
		"realm": realm.Name,
		"id":    eId,
	}).SetBody(map[string]interface{}{
		"alias": "edp-sso",
		"config": map[string]string{
			"defaultProvider": realm.SsoRealmName,
		},
	}).Post(a.basePath + authExecutionConfig)
	if err != nil {
		return errors.Wrap(err, "error during resty request")
	}
	if resp.StatusCode() != 201 {
		return errors.Errorf("response is not ok by create redirect config: Status: %v", resp.Status())
	}

	if !realm.SsoAutoRedirectEnabled {
		resp, err := a.startRestyRequest().SetPathParams(map[string]string{"realm": realm.Name}).
			SetBody(map[string]string{
				"id":          eId,
				"requirement": "DISABLED",
			}).Put(a.basePath + authExecutions)
		if err != nil {
			return errors.Wrap(err, "error during resty request")
		}

		if resp.StatusCode() != 202 {
			return errors.Errorf("response is not ok by create redirect config: Status: %v", resp.Status())
		}
	}

	return nil
}

func (a GoCloakAdapter) getBrowserExecutions(realm *dto.Realm) ([]api.SimpleAuthExecution, error) {
	res := make([]api.SimpleAuthExecution, 0)
	resp, err := a.startRestyRequest().
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

func (a GoCloakAdapter) prepareProtocolMapperMaps(client *dto.Client, clientID string,
	claimedMappers []gocloak.ProtocolMapperRepresentation) (
	currentMappersMap, claimedMappersMap map[string]gocloak.ProtocolMapperRepresentation, resultErr error) {

	currentMappers, err := a.GetClientProtocolMappers(client, clientID)
	if err != nil {
		resultErr = errors.Wrap(err, "unable to get client protocol mappers")
		return
	}

	currentMappersMap = make(map[string]gocloak.ProtocolMapperRepresentation)
	claimedMappersMap = make(map[string]gocloak.ProtocolMapperRepresentation)
	// build maps to optimize comparing loops
	for i, m := range currentMappers {
		currentMappersMap[*m.Name] = currentMappers[i]
	}

	for i, m := range claimedMappers {
		// this block needed to fix 500 error response from server and for proper work of DeepEqual
		if m.Config == nil || *m.Config == nil {
			claimedMappers[i].Config = &map[string]string{}
		}

		claimedMappersMap[*m.Name] = claimedMappers[i]
	}

	return
}

func (a GoCloakAdapter) mapperNeedsToBeCreated(
	claimed *gocloak.ProtocolMapperRepresentation, currentMappersMap map[string]gocloak.ProtocolMapperRepresentation,
	realmName, clientID string) error {

	if _, ok := currentMappersMap[*claimed.Name]; !ok { // not exists in kc, must be created
		if _, err := a.client.CreateClientProtocolMapper(context.Background(), a.token.AccessToken,
			realmName, clientID, *claimed); err != nil {
			return errors.Wrap(err, "unable to client create protocol mapper")
		}
	}

	return nil
}

func (a GoCloakAdapter) mapperNeedsToBeUpdated(
	claimed *gocloak.ProtocolMapperRepresentation, currentMappersMap map[string]gocloak.ProtocolMapperRepresentation,
	realmName, clientID string) error {

	if current, ok := currentMappersMap[*claimed.Name]; ok { // claimed exists in current state, must be checked for update
		claimed.ID = current.ID                   // set id from current entity to claimed for proper DeepEqual comparison
		if !reflect.DeepEqual(claimed, current) { // mappers is not equal, needs to update
			if err := a.client.UpdateClientProtocolMapper(context.Background(), a.token.AccessToken,
				realmName, clientID, *claimed.ID, *claimed); err != nil {
				return errors.Wrap(err, "unable to update client protocol mapper")
			}
		}
	}

	return nil
}

func (a GoCloakAdapter) SyncClientProtocolMapper(
	client *dto.Client, claimedMappers []gocloak.ProtocolMapperRepresentation) error {
	log := a.log.WithValues("clientId", client.ClientId)
	log.Info("Start put Client protocol mappers...")

	clientID, err := a.GetClientID(client.ClientId, client.RealmName)
	if err != nil {
		return errors.Wrap(err, "unable to get client id")
	}
	//prepare mapper entity maps for simplifying comparison procedure
	currentMappersMap, claimedMappersMap, err := a.prepareProtocolMapperMaps(client, clientID, claimedMappers)
	if err != nil {
		return errors.Wrap(err, "unable to prepare protocol mapper maps")
	}
	//compare actual client protocol mappers from keycloak to desired mappers, and sync them
	for _, claimed := range claimedMappers {
		if err := a.mapperNeedsToBeCreated(&claimed, currentMappersMap, client.RealmName, clientID); err != nil {
			return errors.Wrap(err, "error during mapperNeedsToBeCreated")
		}

		if err := a.mapperNeedsToBeUpdated(&claimed, currentMappersMap, client.RealmName, clientID); err != nil {
			return errors.Wrap(err, "error during mapperNeedsToBeUpdated")
		}
	}

	for _, kc := range currentMappersMap {
		if _, ok := claimedMappersMap[*kc.Name]; !ok { //current mapper not exists in claimed, must be deleted
			if err := a.client.DeleteClientProtocolMapper(context.Background(), a.token.AccessToken, client.RealmName,
				clientID, *kc.ID); err != nil {
				return errors.Wrap(err, "unable to delete client protocol mapper")
			}
		}
	}

	log.Info("Client protocol mapper was successfully configured!")
	return nil
}

func (a GoCloakAdapter) GetClientProtocolMappers(client *dto.Client,
	clientID string) ([]gocloak.ProtocolMapperRepresentation, error) {
	var mappers []gocloak.ProtocolMapperRepresentation

	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": client.RealmName,
			"id":    clientID,
		}).
		SetResult(&mappers).Get(a.basePath + getClientProtocolMappers)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get client protocol mappers")
	}

	if resp.IsError() {
		return nil, errors.New(resp.String())
	}

	return mappers, nil
}

func (a GoCloakAdapter) PutClientScopeMapper(clientName, scopeId, realmName string) error {
	log := a.log.WithValues("scopeId", scopeId, "realm", realmName, "clientId", clientName)
	log.Info("Start put Client Scope mapper...")
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm":   realmName,
			"scopeId": scopeId,
		}).
		SetBody(getProtocolMapper(clientName)).
		Post(a.basePath + postClientScopeMapper)
	if err := a.checkError(err, resp); err != nil {
		return err
	}
	log.Info("Client Scope mapper was successfully configured!")
	return nil
}

func (a GoCloakAdapter) checkError(err error, response *resty.Response) error {
	if err != nil {
		return err
	}
	if response == nil {
		return errors.New("empty response")
	}
	if response.IsError() {
		if response.StatusCode() == 409 {
			a.log.Info("entity already exists. creating skipped", "url", response.Request.URL)
			return nil
		}
		return errors.New(response.Status())
	}
	return nil
}

func getProtocolMapper(clientId string) model.ProtocolMappers {
	return model.ProtocolMappers{
		Name:           gocloak.StringP(fmt.Sprintf("%v-%v", clientId, "audience")),
		Protocol:       gocloak.StringP(consts.OpenIdProtocol),
		ProtocolMapper: gocloak.StringP(consts.ProtocolMapper),
		ProtocolMappersConfig: &model.ProtocolMappersConfig{
			AccessTokenClaim:       gocloak.StringP("true"),
			IncludedClientAudience: gocloak.StringP(clientId),
		},
	}
}

func (a GoCloakAdapter) GetClientScope(scopeName, realmName string) (*model.ClientScope, error) {
	log := a.log.WithValues("scopeName", scopeName, "realm", realmName)
	log.Info("Start get Client Scope...")
	var result []*model.ClientScope
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetResult(&result).
		Get(a.basePath + getOneClientScope)
	if err := a.checkError(err, resp); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("realm %v doesnt contain client scopes", realmName)
	}
	scope, err := getClientScope(scopeName, result)
	if err != nil {
		return nil, err
	}
	log.Info("End get Client Scope", "scope", scope)
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

func (a GoCloakAdapter) LinkClientScopeToClient(clientName, scopeID, realmName string) error {
	log := a.log.WithValues("clientName", clientName, "scopeId", scopeID, "realm", realmName)
	log.Info("Start link Client Scope to client...")
	clientID, err := a.GetClientID(clientName, realmName)
	if err != nil {
		return errors.Wrap(err, "error during GetClientId")
	}

	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm":    realmName,
			"clientId": clientID,
			"scopeId":  scopeID,
		}).
		Put(a.basePath + linkClientScopeToClient)
	if err := a.checkError(err, resp); err != nil {
		return errors.Wrapf(err, "error during %s", linkClientScopeToClient)
	}
	log.Info("End link Client Scope to client...")
	return nil
}

func (a GoCloakAdapter) CreateClientScope(realmName string, scope model.ClientScope) error {
	log := a.log.WithValues("realm", realmName, "scope", scope.Name)
	log.Info("Start creating Client Scope...")
	resp, err := a.client.RestyClient().R().
		SetAuthToken(a.token.AccessToken).
		SetHeader("Content-Type", "application/json").
		SetPathParams(map[string]string{
			"realm": realmName,
		}).
		SetBody(scope).
		Post(a.basePath + postClientScope)
	if err := a.checkError(err, resp); err != nil {
		return err
	}
	log.Info("Client Scope was created!")
	return nil
}

func extractError(resp *resty.Response) error {
	if !resp.IsSuccess() {
		return errors.Errorf("status: %d, body: %s", resp.StatusCode(), resp.String())
	}

	return nil
}
