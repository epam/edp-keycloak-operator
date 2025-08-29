package adapter

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Nerzal/gocloak/v12"
	"github.com/go-logr/logr"
	"github.com/go-resty/resty/v2"
	"golang.org/x/sync/errgroup"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/dto"
)

const (
	authPath                        = "/auth"
	openidConnectTokenPath          = "/protocol/openid-connect/token"
	realmEntity                     = "/admin/realms/{realm}"
	idPResource                     = "/admin/realms/{realm}/identity-provider/instances"
	idPMapperResource               = "/admin/realms/{realm}/identity-provider/instances/{alias}/mappers"
	getOneIdP                       = idPResource + "/{alias}"
	idpManagementPermissions        = idPResource + "/{alias}/management/permissions"
	openIdConfig                    = "/realms/{realm}/.well-known/openid-configuration"
	authExecutions                  = "/admin/realms/{realm}/authentication/flows/browser/executions"
	authExecutionConfig             = "/admin/realms/{realm}/authentication/executions/{id}/config"
	postClientScopeMapper           = "/admin/realms/{realm}/client-scopes/{scopeId}/protocol-mappers/models"
	getRealmClientScopes            = "/admin/realms/{realm}/client-scopes"
	postClientScope                 = "/admin/realms/{realm}/client-scopes"
	putClientScope                  = "/admin/realms/{realm}/client-scopes/{id}"
	getClientProtocolMappers        = "/admin/realms/{realm}/clients/{id}/protocol-mappers/models"
	mapperToIdentityProvider        = "/admin/realms/{realm}/identity-provider/instances/{alias}/mappers"
	updateMapperToIdentityProvider  = "/admin/realms/{realm}/identity-provider/instances/{alias}/mappers/{id}"
	authFlows                       = "/admin/realms/{realm}/authentication/flows"
	authFlow                        = "/admin/realms/{realm}/authentication/flows/{id}"
	authFlowExecutionCreate         = "/admin/realms/{realm}/authentication/executions"
	authFlowExecutionGetUpdate      = "/admin/realms/{realm}/authentication/flows/{alias}/executions"
	authFlowExecutionDelete         = "/admin/realms/{realm}/authentication/executions/{id}"
	raiseExecutionPriority          = "/admin/realms/{realm}/authentication/executions/{id}/raise-priority"
	lowerExecutionPriority          = "/admin/realms/{realm}/authentication/executions/{id}/lower-priority"
	authFlowExecutionConfig         = "/admin/realms/{realm}/authentication/executions/{id}/config"
	authFlowConfig                  = "/admin/realms/{realm}/authentication/config/{id}"
	realmAuthFlowParentExecutions   = "/admin/realms/{realm}/authentication/flows/{parentName}/executions/flow"
	deleteClientScopeProtocolMapper = "/admin/realms/{realm}/client-scopes/{clientScopeID}/" +
		"protocol-mappers/models/{protocolMapperID}"
	createClientScopeProtocolMapper = "/admin/realms/{realm}/client-scopes/{clientScopeID}/protocol-mappers/models"
	putDefaultClientScope           = "/admin/realms/{realm}/default-default-client-scopes/{clientScopeID}"
	deleteDefaultClientScope        = "/admin/realms/{realm}/default-default-client-scopes/{clientScopeID}"
	getDefaultClientScopes          = "/admin/realms/{realm}/default-default-client-scopes"
	realmEventConfigPut             = "/admin/realms/{realm}/events/config"
	realmComponent                  = "/admin/realms/{realm}/components"
	realmComponentEntity            = "/admin/realms/{realm}/components/{id}"
	identityProviderEntity          = "/admin/realms/{realm}/identity-provider/instances/{alias}"
	identityProviderCreateList      = "/admin/realms/{realm}/identity-provider/instances"
	idpMapperCreateList             = "/admin/realms/{realm}/identity-provider/instances/{alias}/mappers"
	idpMapperEntity                 = "/admin/realms/{realm}/identity-provider/instances/{alias}/mappers/{id}"
	deleteRealmUser                 = "/admin/realms/{realm}/users/{id}"
	setRealmUserPassword            = "/admin/realms/{realm}/users/{id}/reset-password"
	getUserRealmRoleMappings        = "/admin/realms/{realm}/users/{id}/role-mappings/realm"
	getUserGroupMappings            = "/admin/realms/{realm}/users/{id}/groups"
	manageUserGroups                = "/admin/realms/{realm}/users/{userID}/groups/{groupID}"
	realmUsersProfile               = "/admin/realms/{realm}/users/profile"
	getChildGroups                  = "/admin/realms/{realm}/groups/{groupID}/children"
	getGroup                        = "/admin/realms/{realm}/groups/{groupID}"
	clientManagementPermissions     = "/admin/realms/{realm}/clients/{id}/management/permissions"
	serverInfo                      = "/admin/serverinfo"
	logClientDTO                    = "client dto"
)

const (
	keycloakApiParamId            = "id"
	keycloakApiParamRole          = "role"
	keycloakApiParamRealm         = "realm"
	keycloakApiParamAlias         = "alias"
	keycloakApiParamClientScopeId = "clientScopeID"
)

const (
	logKeyUser     = "user dto"
	logKeyRealm    = "realm"
	logKeyRoleName = "roleName"
)

type TokenExpiredError string

func (e TokenExpiredError) Error() string {
	return string(e)
}

func IsErrTokenExpired(err error) bool {
	errTokenExpired := TokenExpiredError("")

	return errors.As(err, &errTokenExpired)
}

type GoCloakAdapter struct {
	client     GoCloak
	token      *gocloak.JWT
	log        logr.Logger
	basePath   string
	legacyMode bool
}

type JWTPayload struct {
	Exp int64 `json:"exp"`
}

type GoCloakConfig struct {
	Url                string
	User               string
	Password           string
	RootCertificate    string
	InsecureSkipVerify bool
}

func (a GoCloakAdapter) GetGoCloak() GoCloak {
	return a.client
}

func MakeFromToken(conf GoCloakConfig, tokenData []byte, log logr.Logger) (*GoCloakAdapter, error) {
	var token gocloak.JWT
	if err := json.Unmarshal(tokenData, &token); err != nil {
		return nil, fmt.Errorf("unable decode json data: %w", err)
	}

	const requiredTokenParts = 3

	tokenParts := strings.Split(token.AccessToken, ".")

	if len(tokenParts) < requiredTokenParts {
		return nil, errors.New("wrong JWT token structure")
	}

	tokenPayload, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
	if err != nil {
		return nil, fmt.Errorf("wrong JWT token base64 encoding: %w", err)
	}

	var tokenPayloadDecoded JWTPayload
	if err = json.Unmarshal(tokenPayload, &tokenPayloadDecoded); err != nil {
		return nil, fmt.Errorf("unable to decode JWT payload json: %w", err)
	}

	if tokenPayloadDecoded.Exp < time.Now().Unix() {
		return nil, TokenExpiredError("token is expired")
	}

	kcCl, legacyMode, err := makeClientFromToken(conf, token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to make new keycloak client: %w", err)
	}

	return &GoCloakAdapter{
		client:     kcCl,
		token:      &token,
		log:        log,
		basePath:   conf.Url,
		legacyMode: legacyMode,
	}, nil
}

// makeClientFromToken returns Keycloak client, a bool flag indicating
// whether it was created in legacy mode and an error.
func makeClientFromToken(conf GoCloakConfig, token string) (*gocloak.GoCloak, bool, error) {
	restyClient := resty.New()

	if conf.InsecureSkipVerify {
		restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}

	if conf.RootCertificate != "" {
		restyClient.SetRootCertificateFromString(conf.RootCertificate)
	}

	kcCl := gocloak.NewClient(conf.Url)
	kcCl.SetRestyClient(restyClient)

	_, err := kcCl.GetRealms(context.Background(), token)
	if err == nil {
		return kcCl, false, nil
	}

	if isNotLegacyResponseCode(err) {
		return nil, false, fmt.Errorf("unexpected error received while trying to get realms using the modern client: %w", err)
	}

	kcCl = gocloak.NewClient(conf.Url, gocloak.SetLegacyWildFlySupport())
	kcCl.SetRestyClient(restyClient)

	if _, err := kcCl.GetRealms(context.Background(), token); err != nil {
		return nil, false, fmt.Errorf("failed to create both current and legacy clients: %w", err)
	}

	return kcCl, true, nil
}

func MakeFromServiceAccount(ctx context.Context,
	conf GoCloakConfig,
	realm string,
	log logr.Logger,
	restyClient *resty.Client,
) (*GoCloakAdapter, error) {
	if restyClient == nil {
		restyClient = resty.New()
	}

	if conf.InsecureSkipVerify {
		restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}

	if conf.RootCertificate != "" {
		restyClient.SetRootCertificateFromString(conf.RootCertificate)
	}

	kcCl := gocloak.NewClient(conf.Url)
	kcCl.SetRestyClient(restyClient)

	token, err := kcCl.LoginClient(ctx, conf.User, conf.Password, realm)
	if err == nil {
		return &GoCloakAdapter{
			client:     kcCl,
			token:      token,
			log:        log,
			basePath:   conf.Url,
			legacyMode: false,
		}, nil
	}

	if isNotLegacyResponseCode(err) {
		return nil, fmt.Errorf("unexpected error received while trying to get realms using the modern client: %w", err)
	}

	kcCl = gocloak.NewClient(conf.Url, gocloak.SetLegacyWildFlySupport())
	kcCl.SetRestyClient(restyClient)

	token, err = kcCl.LoginClient(ctx, conf.User, conf.Password, realm)
	if err != nil {
		return nil, fmt.Errorf("failed to login with client creds on both current and legacy clients - "+
			"clientID: %s, realm: %s: %w", conf.User, realm, err)
	}

	return &GoCloakAdapter{
		client:     kcCl,
		token:      token,
		log:        log,
		basePath:   conf.Url,
		legacyMode: true,
	}, nil
}

func isNotLegacyResponseCode(err error) bool {
	apiErr := new(gocloak.APIError)
	ok := errors.As(err, &apiErr)

	return !ok || (apiErr.Code != http.StatusNotFound && apiErr.Code != http.StatusServiceUnavailable)
}

func Make(
	ctx context.Context,
	conf GoCloakConfig,
	log logr.Logger,
	restyClient *resty.Client,
) (*GoCloakAdapter, error) {
	if restyClient == nil {
		restyClient = resty.New()
	}

	if conf.InsecureSkipVerify {
		restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}

	if conf.RootCertificate != "" {
		restyClient.SetRootCertificateFromString(conf.RootCertificate)
	}

	kcCl := gocloak.NewClient(conf.Url)
	kcCl.SetRestyClient(restyClient)

	token, err := kcCl.LoginAdmin(ctx, conf.User, conf.Password, "master")
	if err == nil {
		return &GoCloakAdapter{
			client:     kcCl,
			token:      token,
			log:        log,
			basePath:   conf.Url,
			legacyMode: false,
		}, nil
	}

	if isNotLegacyResponseCode(err) {
		return nil, fmt.Errorf("unexpected error received while trying to get realms using the modern client: %w", err)
	}

	kcCl = gocloak.NewClient(conf.Url, gocloak.SetLegacyWildFlySupport())
	kcCl.SetRestyClient(restyClient)

	token, err = kcCl.LoginAdmin(ctx, conf.User, conf.Password, "master")
	if err != nil {
		return nil, fmt.Errorf("cannot login to keycloak server with user: %s: %w", conf.User, err)
	}

	return &GoCloakAdapter{
		client:     kcCl,
		token:      token,
		log:        log,
		basePath:   conf.Url,
		legacyMode: true,
	}, nil
}

func (a GoCloakAdapter) ExportToken() ([]byte, error) {
	tokenData, err := json.Marshal(a.token)
	if err != nil {
		return nil, fmt.Errorf("unable to json encode token: %w", err)
	}

	return tokenData, nil
}

// buildPath returns request path corresponding with the mode the client is operating in.
func (a GoCloakAdapter) buildPath(endpoint string) string {
	if a.legacyMode {
		return a.basePath + authPath + endpoint
	}

	return a.basePath + endpoint
}

func (a GoCloakAdapter) ExistClient(clientID, realm string) (bool, error) {
	log := a.log.WithValues("clientID", clientID, logKeyRealm, realm)
	log.Info("Start check client in Keycloak...")

	clns, err := a.client.GetClients(context.Background(), a.token.AccessToken, realm, gocloak.GetClientsParams{
		ClientID: &clientID,
	})

	if err != nil {
		return false, fmt.Errorf("failed to get clients for realm %s: %w", realm, err)
	}

	res := checkFullNameMatch(clientID, clns)

	log.Info("End check client in Keycloak")

	return res, nil
}

func (a GoCloakAdapter) SyncClientRoles(ctx context.Context, realmName string, client *dto.Client) error {
	log := ctrl.LoggerFrom(ctx).WithValues("clientId", client.ClientId, "realm", realmName)
	log.Info("Start syncing client roles...")

	// Get client ID from Keycloak
	clientID, err := a.GetClientID(client.ClientId, realmName)
	if err != nil {
		return fmt.Errorf("failed to get client ID: %w", err)
	}

	// Get all client roles from Keycloak
	existingRolesMap, err := a.getExistingClientRolesMap(ctx, realmName, clientID)
	if err != nil {
		return fmt.Errorf("failed to get client roles: %w", err)
	}

	desiredRolesMap := createDesiredRolesMap(client.Roles)

	// Sync existing roles (create missing and update existing)
	err = a.syncExistingRoles(ctx, realmName, clientID, client, existingRolesMap)
	if err != nil {
		return err
	}

	// Delete removed roles
	err = a.deleteRemovedRoles(ctx, realmName, clientID, existingRolesMap, desiredRolesMap)
	if err != nil {
		return err
	}

	// Handle AssociatedClientRoles
	if err := a.syncClientRoleComposites(ctx, realmName, clientID, client.Roles); err != nil {
		return fmt.Errorf("failed to sync client role composites: %w", err)
	}

	log.Info("Client roles synchronization completed")

	return nil
}

// createClientRole creates a new client role in Keycloak.
func (a GoCloakAdapter) createClientRole(ctx context.Context, realmName, clientID string, role dto.ClientRole) error {
	log := ctrl.LoggerFrom(ctx).WithValues(logKeyRoleName, role.Name)
	log.Info("Creating missing client role")

	gocloakRole := gocloak.Role{
		Name:        gocloak.StringP(role.Name),
		Description: gocloak.StringP(role.Description),
		ClientRole:  gocloak.BoolP(true),
	}

	_, err := a.client.CreateClientRole(ctx, a.token.AccessToken, realmName, clientID, gocloakRole)
	if err != nil {
		return fmt.Errorf("failed to create client role %s: %w", role.Name, err)
	}

	return nil
}

// updateClientRole updates an existing client role in Keycloak.
func (a GoCloakAdapter) updateClientRole(
	ctx context.Context,
	realmName,
	clientID string,
	role dto.ClientRole,
	existingRole *gocloak.Role,
) error {
	// Check if the role needs to be updated by comparing properties
	needsUpdate := existingRole.Description == nil || *existingRole.Description != role.Description

	// Compare description

	if !needsUpdate {
		return nil
	}

	log := ctrl.LoggerFrom(ctx).WithValues(logKeyRoleName, role.Name)
	log.Info("Updating existing client role")

	// Create updated role with desired properties
	updatedRole := gocloak.Role{
		ID:          existingRole.ID,
		Name:        gocloak.StringP(role.Name),
		Description: gocloak.StringP(role.Description),
		ClientRole:  gocloak.BoolP(true),
	}

	err := a.client.UpdateRole(ctx, a.token.AccessToken, realmName, clientID, updatedRole)
	if err != nil {
		return fmt.Errorf("failed to update client role %s: %w", role.Name, err)
	}

	return nil
}

// deleteRemovedRoles removes roles that are no longer desired.
func (a GoCloakAdapter) deleteRemovedRoles(
	ctx context.Context,
	realmName,
	clientID string,
	existingRolesMap map[string]*gocloak.Role,
	desiredRolesMap map[string]dto.ClientRole,
) error {
	log := ctrl.LoggerFrom(ctx)

	for roleName := range existingRolesMap {
		if _, desired := desiredRolesMap[roleName]; !desired {
			log.WithValues(logKeyRoleName, roleName).Info("Deleting removed client role")

			err := a.client.DeleteClientRole(ctx, a.token.AccessToken, realmName, clientID, roleName)
			if err != nil {
				return fmt.Errorf("failed to delete client role %s: %w", roleName, err)
			}
		}
	}

	return nil
}

// syncExistingRoles handles the creation and updating of existing roles.
func (a GoCloakAdapter) syncExistingRoles(
	ctx context.Context,
	realmName,
	clientID string,
	client *dto.Client,
	existingRolesMap map[string]*gocloak.Role,
) error {
	for _, desiredRole := range client.Roles {
		existingRole, exists := existingRolesMap[desiredRole.Name]
		if !exists {
			err := a.createClientRole(ctx, realmName, clientID, desiredRole)
			if err != nil {
				return err
			}

			continue
		}

		// Update existing role
		err := a.updateClientRole(ctx, realmName, clientID, desiredRole, existingRole)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a GoCloakAdapter) syncClientRoleComposites(
	ctx context.Context,
	realmName,
	clientID string,
	desiredRoles []dto.ClientRole,
) error {
	// Get all existing client roles
	existingRolesMap, err := a.getExistingClientRolesMap(ctx, realmName, clientID)
	if err != nil {
		return fmt.Errorf("failed to get client roles: %w", err)
	}

	for _, desiredRole := range desiredRoles {
		existingRole, exists := existingRolesMap[desiredRole.Name]
		if !exists {
			return fmt.Errorf("failed to setup composite roles: role %s does not exist in Keycloak", desiredRole.Name)
		}

		err = a.processSingleRoleComposites(ctx, realmName, clientID, desiredRole, existingRole)
		if err != nil {
			return err
		}
	}

	return nil
}

// addCompositeRoles adds the specified roles as composites to the given role.
func (a GoCloakAdapter) addCompositeRoles(
	ctx context.Context,
	realmName string,
	roleID string,
	rolesToAdd []gocloak.Role,
	roleName string,
) error {
	if len(rolesToAdd) == 0 {
		return nil
	}

	log := ctrl.LoggerFrom(ctx).WithValues(logKeyRoleName, roleName, "compositeRoles", len(rolesToAdd))
	log.Info("Adding composite roles")

	err := a.client.AddClientRoleComposite(ctx, a.token.AccessToken, realmName, roleID, rolesToAdd)
	if err != nil {
		return fmt.Errorf("failed to add composite roles to %s: %w", roleName, err)
	}

	return nil
}

// removeCompositeRoles removes the specified roles as composites from the given role.
func (a GoCloakAdapter) removeCompositeRoles(
	ctx context.Context,
	realmName string,
	roleID string,
	rolesToRemove []gocloak.Role,
	roleName string,
) error {
	if len(rolesToRemove) == 0 {
		return nil
	}

	log := ctrl.LoggerFrom(ctx).WithValues(logKeyRoleName, roleName, "compositeRoles", len(rolesToRemove))
	log.Info("Removing composite roles")

	err := a.client.DeleteClientRoleComposite(ctx, a.token.AccessToken, realmName, roleID, rolesToRemove)
	if err != nil {
		return fmt.Errorf("failed to remove composite roles from %s: %w", roleName, err)
	}

	return nil
}

// handleEmptyComposites handles the case when no associated roles are desired.
func (a GoCloakAdapter) handleEmptyComposites(
	ctx context.Context,
	realmName string,
	roleID string,
	currentCompositeRoles []*gocloak.Role,
	roleName string,
) error {
	if len(currentCompositeRoles) == 0 {
		return nil
	}

	rolesToRemove := convertRolePointersToValues(currentCompositeRoles)

	return a.removeCompositeRoles(ctx, realmName, roleID, rolesToRemove, roleName)
}

// getAssociatedRoleDetails retrieves the details of an associated role.
func (a GoCloakAdapter) getAssociatedRoleDetails(
	ctx context.Context,
	realmName,
	clientID,
	roleName string,
) (*gocloak.Role, error) {
	associatedRole, err := a.client.GetClientRole(ctx, a.token.AccessToken, realmName, clientID, roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to get associated client role %s: %w", roleName, err)
	}

	return associatedRole, nil
}

// calculateRolesToAdd determines which roles need to be added as composites.
func (a GoCloakAdapter) calculateRolesToAdd(
	ctx context.Context,
	realmName,
	clientID string,
	desiredRoles []string,
	currentCompositeMap map[string]*gocloak.Role,
) ([]gocloak.Role, error) {
	rolesToAdd := make([]gocloak.Role, 0)

	for _, associatedRoleName := range desiredRoles {
		if _, exists := currentCompositeMap[associatedRoleName]; !exists {
			associatedRole, err := a.getAssociatedRoleDetails(ctx, realmName, clientID, associatedRoleName)
			if err != nil {
				return nil, err
			}

			rolesToAdd = append(rolesToAdd, *associatedRole)
		}
	}

	return rolesToAdd, nil
}

// calculateRolesToRemove determines which roles need to be removed from composites.
func (a GoCloakAdapter) calculateRolesToRemove(
	currentCompositeRoles []*gocloak.Role,
	desiredRoles []string,
) []gocloak.Role {
	rolesToRemove := make([]gocloak.Role, 0)

	for _, currentRole := range currentCompositeRoles {
		if currentRole != nil && currentRole.Name != nil {
			if !slices.Contains(desiredRoles, *currentRole.Name) {
				rolesToRemove = append(rolesToRemove, *currentRole)
			}
		}
	}

	return rolesToRemove
}

// processSingleRoleComposites handles the composite role synchronization for a single role.
func (a GoCloakAdapter) processSingleRoleComposites(
	ctx context.Context,
	realmName,
	clientID string,
	desiredRole dto.ClientRole,
	existingRole *gocloak.Role,
) error {
	// Get current composite roles for this role
	currentCompositeRoles, err := a.client.GetCompositeRolesByRoleID(ctx, a.token.AccessToken, realmName, *existingRole.ID)
	if err != nil {
		return fmt.Errorf("failed to get composite roles for role %s: %w", desiredRole.Name, err)
	}

	// If no associated roles are desired, handle empty composites case
	if len(desiredRole.AssociatedClientRoles) == 0 {
		return a.handleEmptyComposites(ctx, realmName, *existingRole.ID, currentCompositeRoles, desiredRole.Name)
	}

	// Create maps for efficient lookup
	currentCompositeMap := createRoleNameMap(currentCompositeRoles)

	// Calculate roles to add and remove
	rolesToAdd, err := a.calculateRolesToAdd(
		ctx,
		realmName,
		clientID,
		desiredRole.AssociatedClientRoles,
		currentCompositeMap,
	)
	if err != nil {
		return err
	}

	rolesToRemove := a.calculateRolesToRemove(currentCompositeRoles, desiredRole.AssociatedClientRoles)

	// Add new composite roles
	err = a.addCompositeRoles(ctx, realmName, *existingRole.ID, rolesToAdd, desiredRole.Name)
	if err != nil {
		return err
	}

	// Remove unwanted composite roles
	err = a.removeCompositeRoles(ctx, realmName, *existingRole.ID, rolesToRemove, desiredRole.Name)
	if err != nil {
		return err
	}

	return nil
}

// getExistingClientRolesMap retrieves all client roles and returns them as a map for efficient lookup.
func (a GoCloakAdapter) getExistingClientRolesMap(
	ctx context.Context,
	realmName,
	clientID string,
) (map[string]*gocloak.Role, error) {
	existingRoles, err := a.client.GetClientRoles(ctx, a.token.AccessToken, realmName, clientID, gocloak.GetRoleParams{
		Max: gocloak.IntP(1000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get client roles: %w", err)
	}

	// Create maps for efficient lookup with preallocated memory
	existingRolesMap := createRoleNameMap(existingRoles)

	return existingRolesMap, nil
}

// convertRolePointersToValues converts a slice of role pointers to a slice of role values, filtering out nil pointers.
func convertRolePointersToValues(roles []*gocloak.Role) []gocloak.Role {
	result := make([]gocloak.Role, 0, len(roles))

	for _, role := range roles {
		if role != nil {
			result = append(result, *role)
		}
	}

	return result
}

// createRoleNameMap creates a map of role names to role pointers for efficient lookup.
func createRoleNameMap(roles []*gocloak.Role) map[string]*gocloak.Role {
	roleMap := make(map[string]*gocloak.Role, len(roles))

	for _, role := range roles {
		if role != nil && role.Name != nil {
			roleMap[*role.Name] = role
		}
	}

	return roleMap
}

// createDesiredRolesMap creates a map of role names to dto.ClientRole for efficient lookup.
func createDesiredRolesMap(roles []dto.ClientRole) map[string]dto.ClientRole {
	roleMap := make(map[string]dto.ClientRole, len(roles))

	for _, role := range roles {
		roleMap[role.Name] = role
	}

	return roleMap
}

func (a GoCloakAdapter) GetRealmRoles(ctx context.Context, realm string) (map[string]gocloak.Role, error) {
	roles, err := a.client.GetRealmRoles(
		ctx,
		a.token.AccessToken,
		realm,
		gocloak.GetRoleParams{
			Max: gocloak.IntP(100),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get realm roles: %w", err)
	}

	rolesMap := make(map[string]gocloak.Role, len(roles))

	for _, r := range roles {
		if r != nil && r.Name != nil {
			rolesMap[*r.Name] = *r
		}
	}

	return rolesMap, nil
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

func checkFullUsernameMatch(userName string, users []*gocloak.User) (*gocloak.User, bool) {
	if users == nil {
		return nil, false
	}

	for _, el := range users {
		if el.Username != nil && *el.Username == userName {
			return el, true
		}
	}

	return nil, false
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

func (a GoCloakAdapter) DeleteClient(ctx context.Context, kcClientID, realmName string) error {
	log := a.log.WithValues("client id", kcClientID)
	log.Info("Start delete client in Keycloak...")

	if err := a.client.DeleteClient(ctx, a.token.AccessToken, realmName, kcClientID); err != nil {
		return fmt.Errorf("unable to delete client: %w", err)
	}

	log.Info("Keycloak client has been deleted")

	return nil
}

func (a GoCloakAdapter) UpdateClient(ctx context.Context, client *dto.Client) error {
	log := a.log.WithValues(logClientDTO, client)
	log.Info("Start update client in Keycloak...")

	if err := a.client.UpdateClient(ctx, a.token.AccessToken, client.RealmName, getGclCln(client)); err != nil {
		return fmt.Errorf("unable to update keycloak client: %w", err)
	}

	log.Info("Keycloak client has been updated")

	return nil
}

func (a GoCloakAdapter) CreateClient(ctx context.Context, client *dto.Client) error {
	log := a.log.WithValues(logClientDTO, client)
	log.Info("Start create client in Keycloak...")

	_, err := a.client.CreateClient(ctx, a.token.AccessToken, client.RealmName, getGclCln(client))
	if err != nil {
		return fmt.Errorf("failed to create keycloak client: %w", err)
	}

	log.Info("Keycloak client has been created")

	return nil
}

func getGclCln(client *dto.Client) gocloak.Client {
	// TODO: check collision with protocol mappers list in spec
	protocolMappers := getProtocolMappers(client.AdvancedProtocolMappers)

	cl := gocloak.Client{
		Attributes:                   &client.Attributes,
		AuthorizationServicesEnabled: &client.AuthorizationServicesEnabled,
		BearerOnly:                   &client.BearerOnly,
		ClientAuthenticatorType:      &client.ClientAuthenticatorType,
		ClientID:                     &client.ClientId,
		ConsentRequired:              &client.ConsentRequired,
		Description:                  &client.Description,
		DirectAccessGrantsEnabled:    &client.DirectAccess,
		Enabled:                      &client.Enabled,
		FrontChannelLogout:           &client.FrontChannelLogout,
		FullScopeAllowed:             &client.FullScopeAllowed,
		ImplicitFlowEnabled:          &client.ImplicitFlowEnabled,
		Name:                         &client.Name,
		Origin:                       &client.Origin,
		Protocol:                     &client.Protocol,
		ProtocolMappers:              &protocolMappers,
		PublicClient:                 &client.PublicClient,
		RedirectURIs: &[]string{
			client.WebUrl + "/*",
		},
		RegistrationAccessToken:            &client.RegistrationAccessToken,
		RootURL:                            &client.WebUrl,
		AdminURL:                           &client.AdminUrl,
		BaseURL:                            &client.HomeUrl,
		Secret:                             &client.ClientSecret,
		ServiceAccountsEnabled:             &client.ServiceAccountEnabled,
		StandardFlowEnabled:                &client.StandardFlowEnabled,
		SurrogateAuthRequired:              &client.SurrogateAuthRequired,
		WebOrigins:                         &client.WebOrigins,
		AuthenticationFlowBindingOverrides: &client.AuthenticationFlowBindingOverrides,
	}

	// Set the admin URL to the web URL for backwards compatibility.
	// Before adding the admin URL field, the admin URL was the same as the web URL.
	if client.AdminUrl == "" {
		cl.AdminURL = &client.WebUrl
	}

	if len(client.RedirectUris) > 0 {
		cl.RedirectURIs = &client.RedirectUris
	}

	if client.ID != "" {
		cl.ID = &client.ID
	}

	return cl
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
				"userinfo.token.claim": strconv.FormatBool(true),
				"multivalued":          strconv.FormatBool(true),
				"id.token.claim":       strconv.FormatBool(true),
				"access.token.claim":   strconv.FormatBool(false),
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
		return "", fmt.Errorf("unable to get realm clients: %w", err)
	}

	for _, item := range clients {
		if item.ClientID != nil && *item.ClientID == clientID {
			return *item.ID, nil
		}
	}

	return "", NotFoundError(fmt.Sprintf("unable to get Client ID. Client %v doesn't exist", clientID))
}

func (a GoCloakAdapter) GetClients(ctx context.Context, realm string) (map[string]*gocloak.Client, error) {
	clients, err := a.client.GetClients(ctx, a.token.AccessToken, realm, gocloak.GetClientsParams{
		Max: gocloak.IntP(100),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get clients for realm %s: %w", realm, err)
	}

	cl := make(map[string]*gocloak.Client, len(clients))

	for _, c := range clients {
		if c.ClientID != nil {
			cl[*c.ClientID] = c
		}
	}

	return cl, nil
}

func (a GoCloakAdapter) GetClient(ctx context.Context, realm, client string) (*gocloak.Client, error) {
	cl, err := a.client.GetClients(ctx, a.token.AccessToken, realm, gocloak.GetClientsParams{
		ClientID: gocloak.StringP(client),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get clients for realm %s: %w", realm, err)
	}

	if len(cl) == 0 {
		return nil, NotFoundError(fmt.Sprintf("client %s doesn't exist", client))
	}

	if len(cl) > 1 {
		return nil, fmt.Errorf("more than one client with ID %s found", client)
	}

	return cl[0], nil
}

func (a GoCloakAdapter) CreateRealmUser(realmName string, user *dto.User) error {
	log := a.log.WithValues(logKeyUser, user, logKeyRealm, realmName)
	log.Info("Start create realm user in Keycloak...")

	userDto := gocloak.User{
		Username: &user.Username,
		Email:    &user.Username,
		Enabled:  gocloak.BoolP(true),
	}

	_, err := a.client.CreateUser(context.Background(), a.token.AccessToken, realmName, userDto)
	if err != nil {
		return fmt.Errorf("failed to create user in realm %s: %w", realmName, err)
	}

	log.Info("Keycloak realm user has been created")

	return nil
}

func (a GoCloakAdapter) ExistRealmUser(realmName string, user *dto.User) (bool, error) {
	log := a.log.WithValues(logKeyUser, user, logKeyRealm, realmName)
	log.Info("Start check user in Keycloak realm...")

	usr, err := a.client.GetUsers(context.Background(), a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: &user.Username,
	})

	_, err = strip404(err)
	if err != nil {
		return false, err
	}

	_, userExists := checkFullUsernameMatch(user.Username, usr)

	log.Info("End check user in Keycloak", "userExists", userExists)

	return userExists, nil
}

// GetUsersByNames returns a map of users by their names.
// The function use goroutines to get users in parallel because the Keycloak API doesn't support getting users by names.
func (a GoCloakAdapter) GetUsersByNames(
	ctx context.Context,
	realm string,
	names []string,
) (map[string]gocloak.User, error) {
	namesChan := make(chan string)
	go func() {
		defer close(namesChan)

		for _, name := range names {
			namesChan <- name
		}
	}()

	const workersCount = 10

	var wg sync.WaitGroup

	wg.Add(workersCount)

	results := make(chan *gocloak.User)
	errc := make(chan error, workersCount)

	for i := 0; i < workersCount; i++ {
		go func(ctx context.Context, realm string, names <-chan string, results chan<- *gocloak.User, errc chan<- error) {
			defer wg.Done()

			for userName := range names {
				users, err := a.client.GetUsers(ctx, a.token.AccessToken, realm, gocloak.GetUsersParams{
					Max:                 gocloak.IntP(100),
					BriefRepresentation: gocloak.BoolP(true),
					Username:            gocloak.StringP(userName),
				})

				if err != nil {
					errc <- fmt.Errorf("failed to get user %s from realm %s: %w", userName, realm, err)
					return
				}

				user, _ := checkFullUsernameMatch(userName, users)

				results <- user
			}
		}(ctx, realm, namesChan, results, errc)
	}

	go func() {
		wg.Wait()
		close(results)
		close(errc)
	}()

	users := make(map[string]gocloak.User, len(names))

	for user := range results {
		if user != nil && user.Username != nil {
			users[*user.Username] = *user
		}
	}

	if err := <-errc; err != nil {
		return nil, err
	}

	return users, nil
}

func (a GoCloakAdapter) DeleteRealmUser(ctx context.Context, realmName, username string) error {
	usrs, err := a.client.GetUsers(ctx, a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: &username,
	})

	if err != nil {
		return fmt.Errorf("unable to get users: %w", err)
	}

	usr, exists := checkFullUsernameMatch(username, usrs)
	if !exists {
		return NotFoundError("user not found")
	}

	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    *usr.ID,
		}).
		Delete(a.buildPath(deleteRealmUser))

	if err = a.checkError(err, rsp); err != nil {
		return fmt.Errorf("unable to delete user: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) HasUserRealmRole(realmName string, user *dto.User, role string) (bool, error) {
	log := a.log.WithValues(keycloakApiParamRole, role, logKeyRealm, realmName, logKeyUser, user)
	log.Info("Start check user roles in Keycloak realm...")

	users, err := a.client.GetUsers(context.Background(), a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: &user.Username,
	})
	if err != nil {
		return false, fmt.Errorf("unable to get users from keycloak: %w", err)
	}

	if len(users) == 0 {
		return false, fmt.Errorf("no such user %v has been found", user.Username)
	}

	rolesMapping, err := a.client.GetRoleMappingByUserID(context.Background(), a.token.AccessToken, realmName,
		*users[0].ID)
	if err != nil {
		return false, fmt.Errorf("unable to GetRoleMappingByUserID: %w", err)
	}

	hasRealmRole := checkFullRoleNameMatch(role, rolesMapping.RealmMappings)

	log.Info("End check user role in Keycloak", "hasRealmRole", hasRealmRole)

	return hasRealmRole, nil
}

func (a GoCloakAdapter) AddRealmRoleToUser(ctx context.Context, realmName, username, roleName string) error {
	users, err := a.client.GetUsers(ctx, a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: &username,
	})
	if err != nil {
		return fmt.Errorf("error during get kc users: %w", err)
	}

	if len(users) == 0 {
		return fmt.Errorf("no users with username %s found", username)
	}

	rl, err := a.client.GetRealmRole(ctx, a.token.AccessToken, realmName, roleName)
	if err != nil {
		return fmt.Errorf("unable to get realm role from keycloak: %w", err)
	}

	if err := a.client.AddRealmRoleToUser(ctx, a.token.AccessToken, realmName, *users[0].ID,
		[]gocloak.Role{
			*rl,
		}); err != nil {
		return fmt.Errorf("unable to add realm role to user: %w", err)
	}

	return nil
}

func getDefaultRealm(realm *dto.Realm) gocloak.RealmRepresentation {
	return gocloak.RealmRepresentation{
		Realm:   &realm.Name,
		Enabled: gocloak.BoolP(true),
		ID:      realm.ID,
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
	log := a.log.WithValues(logKeyRealm, realmName, keycloakApiParamRole, role)
	log.Info("Start create realm roles in Keycloak...")

	realmRole := gocloak.Role{
		Name: &role.Name,
	}

	_, err := a.client.CreateRealmRole(context.Background(), a.token.AccessToken, realmName, realmRole)
	if err != nil {
		return fmt.Errorf("failed to create realm role %s: %w", role.Name, err)
	}

	persRole, err := a.client.GetRealmRole(context.Background(), a.token.AccessToken, realmName, role.Name)
	if err != nil {
		return fmt.Errorf("failed to get realm role %s: %w", role.Name, err)
	}

	err = a.client.AddRealmRoleComposite(
		context.Background(),
		a.token.AccessToken,
		realmName,
		role.Composite,
		[]gocloak.Role{*persRole},
	)
	if err != nil {
		return fmt.Errorf("failed to add realm role composite: %w", err)
	}

	log.Info("Keycloak roles has been created")

	return nil
}

func (a GoCloakAdapter) CreatePrimaryRealmRole(
	ctx context.Context,
	realmName string,
	role *dto.PrimaryRealmRole,
) (string, error) {
	log := ctrl.LoggerFrom(ctx).WithValues("realm_name", realmName, keycloakApiParamRole, role)
	log.Info("Start create realm roles in Keycloak.")

	realmRole := gocloak.Role{
		Name:        &role.Name,
		Description: &role.Description,
		Attributes:  &role.Attributes,
		Composite:   &role.IsComposite,
	}

	if _, err := a.client.CreateRealmRole(ctx, a.token.AccessToken, realmName, realmRole); err != nil {
		return "", fmt.Errorf("failed to create realm role %s: %w", role.Name, err)
	}

	currentRealmRole, err := a.client.GetRealmRole(ctx, a.token.AccessToken, realmName, role.Name)
	if err != nil {
		return "", fmt.Errorf("failed to get created realm role %s: %w", role.Name, err)
	}

	role.ID = currentRealmRole.ID

	log.Info("Keycloak roles has been created.")

	return *role.ID, nil
}

func (a GoCloakAdapter) syncRoleComposites(ctx context.Context, realmName string, role *dto.PrimaryRealmRole) error {
	associatedRoles, err := a.getRolesAssociatedRoles(ctx, realmName, role)
	if err != nil {
		return err
	}

	realmRolesToAdd, err := a.processAssociatedRealmRoles(ctx, realmName, role, associatedRoles)
	if err != nil {
		return err
	}

	clientRolesToAdd, err := a.processAssociatedClientRoles(ctx, realmName, role, associatedRoles)
	if err != nil {
		return err
	}

	rolesToAdd := slices.Clone(realmRolesToAdd)
	rolesToAdd = append(rolesToAdd, clientRolesToAdd...)

	if len(rolesToAdd) > 0 {
		if err = a.client.AddRealmRoleComposite(ctx, a.token.AccessToken, realmName, role.Name, rolesToAdd); err != nil {
			return fmt.Errorf("unable to add realm role composite roles: %w", err)
		}
	}

	if len(associatedRoles) > 0 {
		if err = a.client.DeleteRealmRoleComposite(
			ctx,
			a.token.AccessToken,
			realmName,
			role.Name,
			slices.Collect(maps.Values(associatedRoles)),
		); err != nil {
			return fmt.Errorf("unable to delete realm role composite roles: %w", err)
		}
	}

	return nil
}

// processAssociatedRealmRoles returns realm roles to add to the role.
// It also removes roles from associatedRoles map that are already associated with the role.
func (a GoCloakAdapter) processAssociatedRealmRoles(
	ctx context.Context,
	realmName string,
	role *dto.PrimaryRealmRole,
	associatedRoles map[string]gocloak.Role,
) ([]gocloak.Role, error) {
	rolesToAdd := make([]gocloak.Role, 0, len(role.Composites))
	group := errgroup.Group{}
	m := sync.Mutex{}

	for _, composite := range role.Composites {
		roleName := composite

		if _, ok := associatedRoles[roleName]; ok {
			delete(associatedRoles, roleName)
			continue
		}

		group.Go(func() error {
			compositeRole, err := a.client.GetRealmRole(ctx, a.token.AccessToken, realmName, roleName)
			if err != nil {
				return fmt.Errorf("unable to get realm  role %s: %w", roleName, err)
			}

			m.Lock()
			rolesToAdd = append(rolesToAdd, *compositeRole)
			m.Unlock()

			return nil
		})
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("unable to get realm roles: %w", err)
	}

	return rolesToAdd, nil
}

// processAssociatedClientRoles returns client roles to add to the role.
// It also removes roles from associatedRoles map that are already associated with the role.
func (a GoCloakAdapter) processAssociatedClientRoles(
	ctx context.Context,
	realmName string,
	role *dto.PrimaryRealmRole,
	associatedRoles map[string]gocloak.Role,
) ([]gocloak.Role, error) {
	rolesToAdd := make([]gocloak.Role, 0)
	group := errgroup.Group{}
	m := sync.Mutex{}

	for cl, composite := range role.CompositesClientRoles {
		roles := composite

		client, err := a.GetClient(ctx, realmName, cl)
		if err != nil {
			return nil, fmt.Errorf("unable to get client %s: %w", cl, err)
		}

		for _, r := range roles {
			roleName := r
			clientID := *client.ID
			mapKey := fmt.Sprintf("%s-%s", clientID, roleName)

			if _, ok := associatedRoles[mapKey]; ok {
				delete(associatedRoles, mapKey)
				continue
			}

			group.Go(func() error {
				compositeRole, err := a.client.GetClientRole(ctx, a.token.AccessToken, realmName, clientID, roleName)
				if err != nil {
					return fmt.Errorf("unable to get client role %s: %w", roleName, err)
				}

				m.Lock()
				rolesToAdd = append(rolesToAdd, *compositeRole)
				m.Unlock()

				return nil
			})
		}
	}

	if err := group.Wait(); err != nil {
		return nil, fmt.Errorf("unable to get realm roles: %w", err)
	}

	return rolesToAdd, nil
}

// getRolesAssociatedRoles returns map of roles associated with role.
// Key is role name. If role is client role, key is client name + "-" + role name.
func (a GoCloakAdapter) getRolesAssociatedRoles(
	ctx context.Context,
	realmName string,
	role *dto.PrimaryRealmRole,
) (map[string]gocloak.Role, error) {
	currentAssociatedRoles, err := a.client.GetCompositeRolesByRoleID(ctx, a.token.AccessToken, realmName, *role.ID)
	if err != nil {
		return nil, fmt.Errorf("unable to get composite realm roles: %w", err)
	}

	currentAssociatedRolesMap := make(map[string]gocloak.Role, len(currentAssociatedRoles))

	for _, r := range currentAssociatedRoles {
		mapKey := *r.Name

		if r.ClientRole != nil && *r.ClientRole {
			mapKey = fmt.Sprintf("%s-%s", *r.ContainerID, *r.Name)
		}

		currentAssociatedRolesMap[mapKey] = *r
	}

	return currentAssociatedRolesMap, nil
}

func (a GoCloakAdapter) GetOpenIdConfig(realm *dto.Realm) (string, error) {
	log := a.log.WithValues("realm dto", realm)
	log.Info("Start get openid configuration...")

	resp, err := a.client.RestyClient().R().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realm.Name,
		}).
		Get(a.buildPath(openIdConfig))
	if err != nil {
		return "", fmt.Errorf("request get open id config failed: %w", err)
	}

	res := resp.String()

	log.Info("End get openid configuration", "openIdConfig", res)

	return res, nil
}

func (a GoCloakAdapter) prepareProtocolMapperMaps(
	client *dto.Client,
	clientID string,
	claimedMappers []gocloak.ProtocolMapperRepresentation,
) (
	currentMappersMap,
	claimedMappersMap map[string]gocloak.ProtocolMapperRepresentation,
	resultErr error,
) {
	currentMappers, err := a.GetClientProtocolMappers(client, clientID)
	if err != nil {
		resultErr = fmt.Errorf("unable to get client protocol mappers: %w", err)

		return currentMappersMap,
			claimedMappersMap,
			resultErr
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

	return currentMappersMap,
		claimedMappersMap,
		resultErr
}

func (a GoCloakAdapter) mapperNeedsToBeCreated(
	claimed *gocloak.ProtocolMapperRepresentation,
	currentMappersMap map[string]gocloak.ProtocolMapperRepresentation,
	realmName,
	clientID string,
) error {
	if _, ok := currentMappersMap[*claimed.Name]; !ok { // not exists in kc, must be created
		if _, err := a.client.CreateClientProtocolMapper(context.Background(), a.token.AccessToken,
			realmName, clientID, *claimed); err != nil {
			return fmt.Errorf("unable to client create protocol mapper: %w", err)
		}
	}

	return nil
}

func (a GoCloakAdapter) mapperNeedsToBeUpdated(
	claimed *gocloak.ProtocolMapperRepresentation,
	currentMappersMap map[string]gocloak.ProtocolMapperRepresentation,
	realmName,
	clientID string,
) error {
	if current, ok := currentMappersMap[*claimed.Name]; ok { // claimed exists in current state, must be checked for update
		claimed.ID = current.ID                   // set id from current entity to claimed for proper DeepEqual comparison
		if !reflect.DeepEqual(claimed, current) { // mappers is not equal, needs to update
			if err := a.client.UpdateClientProtocolMapper(context.Background(), a.token.AccessToken,
				realmName, clientID, *claimed.ID, *claimed); err != nil {
				return fmt.Errorf("unable to update client protocol mapper: %w", err)
			}
		}
	}

	return nil
}

func (a GoCloakAdapter) SyncClientProtocolMapper(
	client *dto.Client, claimedMappers []gocloak.ProtocolMapperRepresentation, addOnly bool) error {
	log := a.log.WithValues("clientId", client.ClientId)
	log.Info("Start put Client protocol mappers...")

	clientID, err := a.GetClientID(client.ClientId, client.RealmName)
	if err != nil {
		return fmt.Errorf("unable to get client id: %w", err)
	}
	// prepare mapper entity maps for simplifying comparison procedure
	currentMappersMap, claimedMappersMap, err := a.prepareProtocolMapperMaps(client, clientID, claimedMappers)
	if err != nil {
		return fmt.Errorf("unable to prepare protocol mapper maps: %w", err)
	}
	// compare actual client protocol mappers from keycloak to desired mappers, and sync them
	for _, claimed := range claimedMappers {
		if err := a.mapperNeedsToBeCreated(&claimed, currentMappersMap, client.RealmName, clientID); err != nil {
			return fmt.Errorf("error during mapperNeedsToBeCreated: %w", err)
		}

		if err := a.mapperNeedsToBeUpdated(&claimed, currentMappersMap, client.RealmName, clientID); err != nil {
			return fmt.Errorf("error during mapperNeedsToBeUpdated: %w", err)
		}
	}

	if !addOnly {
		for _, kc := range currentMappersMap {
			if _, ok := claimedMappersMap[*kc.Name]; !ok { // current mapper not exists in claimed, must be deleted
				if err := a.client.DeleteClientProtocolMapper(context.Background(), a.token.AccessToken, client.RealmName,
					clientID, *kc.ID); err != nil {
					return fmt.Errorf("unable to delete client protocol mapper: %w", err)
				}
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
		SetHeader(contentTypeHeader, contentTypeJson).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: client.RealmName,
			keycloakApiParamId:    clientID,
		}).
		SetResult(&mappers).Get(a.buildPath(getClientProtocolMappers))
	if err != nil {
		return nil, fmt.Errorf("failed to get client protocol mappers: %w", err)
	}

	if resp.IsError() {
		return nil, errors.New(resp.String())
	}

	return mappers, nil
}

func (a GoCloakAdapter) checkError(err error, response *resty.Response) error {
	if err != nil {
		return fmt.Errorf("response error: %w", err)
	}

	if response == nil {
		return errors.New("empty response")
	}

	if response.IsError() {
		respErr := fmt.Errorf("status: %s, body: %s", response.Status(), response.String())

		if response.StatusCode() == http.StatusNotFound {
			return NotFoundError(respErr.Error())
		}

		return respErr
	}

	return nil
}
