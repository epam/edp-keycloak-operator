package keycloakapi

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
)

type (
	ClientRepresentation          = generated.ClientRepresentation
	GetClientsParams              = generated.GetAdminRealmsRealmClientsParams
	GetClientRolesParams          = generated.GetAdminRealmsRealmClientsClientUuidRolesParams
	ProtocolMapperRepresentation  = generated.ProtocolMapperRepresentation
	ManagementPermissionReference = generated.ManagementPermissionReference
	ClientScopeRepresentation     = generated.ClientScopeRepresentation
	GetClientSessionsParams       = generated.GetAdminRealmsRealmClientsClientUuidUserSessionsParams
)

type clientsClient struct {
	client generated.ClientWithResponsesInterface
}

var _ ClientsClient = (*clientsClient)(nil)

// Client management methods

func (c *clientsClient) GetClients(
	ctx context.Context,
	realm string,
	params *GetClientsParams,
) ([]ClientRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsWithResponse(ctx, realm, params)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (c *clientsClient) GetClient(
	ctx context.Context,
	realm string,
	clientID string,
) (*ClientRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidWithResponse(ctx, realm, clientID)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.JSON200, response, nil
}

func (c *clientsClient) CreateClient(
	ctx context.Context,
	realm string,
	client ClientRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmClientsWithResponse(ctx, realm, client)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *clientsClient) DeleteClient(ctx context.Context, realm, clientID string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmClientsClientUuidWithResponse(ctx, realm, clientID)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

// Client role methods

func (c *clientsClient) GetClientRoles(
	ctx context.Context,
	realm string,
	clientID string,
	params *GetClientRolesParams,
) ([]RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidRolesWithResponse(ctx, realm, clientID, params)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (c *clientsClient) GetClientRole(
	ctx context.Context,
	realm string,
	clientID string,
	roleName string,
) (*RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidRolesRoleNameWithResponse(ctx, realm, clientID, roleName)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.JSON200, response, nil
}

func (c *clientsClient) CreateClientRole(
	ctx context.Context,
	realm string,
	clientID string,
	role RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmClientsClientUuidRolesWithResponse(ctx, realm, clientID, role)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *clientsClient) UpdateClientRole(
	ctx context.Context,
	realm, clientID, roleName string,
	role RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmClientsClientUuidRolesRoleNameWithResponse(
		ctx, realm, clientID, roleName, role,
	)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *clientsClient) DeleteClientRole(ctx context.Context, realm, clientID, roleName string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmClientsClientUuidRolesRoleNameWithResponse(ctx, realm, clientID, roleName)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

// UpdateClient updates a Keycloak client by its UUID.
func (c *clientsClient) UpdateClient(
	ctx context.Context,
	realm string,
	clientUUID string,
	client ClientRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmClientsClientUuidWithResponse(ctx, realm, clientUUID, client)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

// GetClientByClientID looks up a client by its clientId string (not UUID).
func (c *clientsClient) GetClientByClientID(
	ctx context.Context,
	realm string,
	clientID string,
) (*ClientRepresentation, *Response, error) {
	params := &GetClientsParams{ClientId: &clientID}

	clients, resp, err := c.GetClients(ctx, realm, params)
	if err != nil {
		return nil, resp, err
	}

	if len(clients) == 0 {
		return nil, resp, ErrNotFound
	}

	return &clients[0], resp, nil
}

// GetClientUUID returns the UUID of the client identified by clientID, or an error if not found.
func (c *clientsClient) GetClientUUID(ctx context.Context, realm, clientID string) (string, error) {
	client, _, err := c.GetClientByClientID(ctx, realm, clientID)
	if err != nil {
		return "", err
	}

	if client.Id == nil {
		return "", fmt.Errorf("client %s not found", clientID)
	}

	return *client.Id, nil
}

// Client scope management

func (c *clientsClient) GetDefaultClientScopes(
	ctx context.Context,
	realm string,
	clientUUID string,
) ([]ClientScopeRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidDefaultClientScopesWithResponse(ctx, realm, clientUUID)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (c *clientsClient) AddDefaultClientScope(
	ctx context.Context,
	realm string,
	clientUUID string,
	scopeID string,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmClientsClientUuidDefaultClientScopesClientScopeIdWithResponse(
		ctx, realm, clientUUID, scopeID,
	)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *clientsClient) GetOptionalClientScopes(
	ctx context.Context,
	realm string,
	clientUUID string,
) ([]ClientScopeRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidOptionalClientScopesWithResponse(ctx, realm, clientUUID)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (c *clientsClient) AddOptionalClientScope(
	ctx context.Context,
	realm string,
	clientUUID string,
	scopeID string,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmClientsClientUuidOptionalClientScopesClientScopeIdWithResponse(
		ctx, realm, clientUUID, scopeID,
	)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

// GetRealmClientScopes returns all client scopes for a realm.
func (c *clientsClient) GetRealmClientScopes(
	ctx context.Context,
	realm string,
) ([]ClientScopeRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientScopesWithResponse(ctx, realm)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

// Service account methods

func (c *clientsClient) GetServiceAccountUser(
	ctx context.Context,
	realm string,
	clientUUID string,
) (*UserRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidServiceAccountUserWithResponse(ctx, realm, clientUUID)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.JSON200, response, nil
}

// Protocol mapper methods

func (c *clientsClient) GetClientProtocolMappers(
	ctx context.Context,
	realm string,
	clientUUID string,
) ([]ProtocolMapperRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidProtocolMappersModelsWithResponse(ctx, realm, clientUUID)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (c *clientsClient) CreateClientProtocolMapper(
	ctx context.Context,
	realm string,
	clientUUID string,
	mapper ProtocolMapperRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmClientsClientUuidProtocolMappersModelsWithResponse(
		ctx, realm, clientUUID, mapper,
	)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *clientsClient) UpdateClientProtocolMapper(
	ctx context.Context,
	realm string,
	clientUUID string,
	mapperID string,
	mapper ProtocolMapperRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmClientsClientUuidProtocolMappersModelsIdWithResponse(
		ctx, realm, clientUUID, mapperID, mapper,
	)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *clientsClient) DeleteClientProtocolMapper(
	ctx context.Context,
	realm string,
	clientUUID string,
	mapperID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmClientsClientUuidProtocolMappersModelsIdWithResponse(
		ctx, realm, clientUUID, mapperID,
	)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

// Management permissions methods

func (c *clientsClient) GetClientManagementPermissions(
	ctx context.Context,
	realm string,
	clientUUID string,
) (*ManagementPermissionReference, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidManagementPermissionsWithResponse(ctx, realm, clientUUID)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.JSON200, response, nil
}

func (c *clientsClient) UpdateClientManagementPermissions(
	ctx context.Context,
	realm string,
	clientUUID string,
	permissions ManagementPermissionReference,
) (*ManagementPermissionReference, *Response, error) {
	res, err := c.client.PutAdminRealmsRealmClientsClientUuidManagementPermissionsWithResponse(
		ctx, realm, clientUUID, permissions,
	)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.JSON200, response, nil
}

// Client role composite methods

func (c *clientsClient) GetClientRoleComposites(
	ctx context.Context,
	realm string,
	clientUUID string,
	roleName string,
) ([]RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidRolesRoleNameCompositesWithResponse(
		ctx, realm, clientUUID, roleName,
	)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (c *clientsClient) AddClientRoleComposites(
	ctx context.Context,
	realm string,
	clientUUID string,
	roleName string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmClientsClientUuidRolesRoleNameCompositesWithResponse(
		ctx, realm, clientUUID, roleName, roles,
	)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *clientsClient) DeleteClientRoleComposites(
	ctx context.Context,
	realm string,
	clientUUID string,
	roleName string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmClientsClientUuidRolesRoleNameCompositesWithResponse(
		ctx, realm, clientUUID, roleName, roles,
	)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *clientsClient) GetClientSecret(
	ctx context.Context,
	realm, clientUUID string,
) (*CredentialRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidClientSecretWithResponse(ctx, realm, clientUUID)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.JSON200, response, nil
}

func (c *clientsClient) RegenerateClientSecret(
	ctx context.Context,
	realm, clientUUID string,
) (*CredentialRepresentation, *Response, error) {
	res, err := c.client.PostAdminRealmsRealmClientsClientUuidClientSecretWithResponse(ctx, realm, clientUUID)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.JSON200, response, nil
}

func (c *clientsClient) GetClientSessions(
	ctx context.Context,
	realm, clientUUID string,
	params *GetClientSessionsParams,
) ([]UserSessionRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidUserSessionsWithResponse(
		ctx, realm, clientUUID, params,
	)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (c *clientsClient) GetClientInstallationProvider(
	ctx context.Context,
	realm, clientUUID, providerID string,
) ([]byte, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientsClientUuidInstallationProvidersProviderIdWithResponse(
		ctx, realm, clientUUID, providerID,
	)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.Body, response, nil
}
