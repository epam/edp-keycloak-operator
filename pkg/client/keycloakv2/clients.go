package keycloakv2

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type (
	ClientRepresentation = generated.ClientRepresentation
	GetClientsParams     = generated.GetAdminRealmsRealmClientsParams
	GetClientRolesParams = generated.GetAdminRealmsRealmClientsClientUuidRolesParams
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
