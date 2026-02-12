package keycloakv2

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type (
	RoleRepresentation  = generated.RoleRepresentation
	GetRealmRolesParams = generated.GetAdminRealmsRealmRolesParams
)

type rolesClient struct {
	client generated.ClientWithResponsesInterface
}

var _ RolesClient = (*rolesClient)(nil)

func (c *rolesClient) GetRealmRoles(
	ctx context.Context,
	realm string,
	params *GetRealmRolesParams,
) ([]RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmRolesWithResponse(ctx, realm, params)
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

func (c *rolesClient) GetRealmRole(
	ctx context.Context,
	realm string,
	roleName string,
) (*RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmRolesRoleNameWithResponse(ctx, realm, roleName)
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

func (c *rolesClient) CreateRealmRole(ctx context.Context, realm string, role RoleRepresentation) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmRolesWithResponse(ctx, realm, role)
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

func (c *rolesClient) DeleteRealmRole(ctx context.Context, realm, roleName string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmRolesRoleNameWithResponse(ctx, realm, roleName)
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
