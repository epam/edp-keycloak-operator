package keycloakapi

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
)

type (
	RoleRepresentation  = generated.RoleRepresentation
	GetRealmRolesParams = generated.GetAdminRealmsRealmRolesParams
)

// RequireRealmRoleWithID validates a GetRealmRole result for work that needs the role's id:
// role mappings, composites and default-role entries. Keycloak matches a mapping payload on
// name and id together, and a composite on id alone, and answers either mismatch with 404.
// The returned representation carries the requested name and a non-empty Id.
//
// GetRealmRole itself does not enforce this. A realm role is addressable by name on other
// endpoints, so a response without an id is incomplete rather than missing.
func RequireRealmRoleWithID(role *RoleRepresentation, realm, name string) (RoleRepresentation, error) {
	return requireRoleWithID(role, "realm role", name, "in realm", realm)
}

// RequireClientRoleWithID validates a GetClientRole result for work that needs the role's id:
// role mappings and composites. Same rule as [RequireRealmRoleWithID]; GetClientRole does not
// enforce it. clientID is the human-readable clientId and only names the client in the error.
// The returned representation carries the requested name and a non-empty Id.
func RequireClientRoleWithID(role *RoleRepresentation, clientID, name string) (RoleRepresentation, error) {
	return requireRoleWithID(role, "client role", name, "for client", clientID)
}

func requireRoleWithID(role *RoleRepresentation, kind, name, containerLabel, containerName string) (RoleRepresentation, error) {
	if role == nil {
		return RoleRepresentation{}, fmt.Errorf("%s %q not found %s %q", kind, name, containerLabel, containerName)
	}

	if role.Name == nil {
		return RoleRepresentation{}, fmt.Errorf("%s %q has no name", kind, name)
	}

	if *role.Name != name {
		return RoleRepresentation{}, fmt.Errorf("%s lookup for %q returned %q", kind, name, *role.Name)
	}

	if role.Id == nil || *role.Id == "" {
		return RoleRepresentation{}, fmt.Errorf("%s %q has no id", kind, name)
	}

	return *role, nil
}

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

func (c *rolesClient) UpdateRealmRole(
	ctx context.Context,
	realm, roleName string,
	role RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmRolesRoleNameWithResponse(ctx, realm, roleName, role)
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

func (c *rolesClient) GetRealmRoleComposites(
	ctx context.Context,
	realm, roleName string,
) ([]RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmRolesRoleNameCompositesWithResponse(ctx, realm, roleName)
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

func (c *rolesClient) AddRealmRoleComposites(
	ctx context.Context,
	realm, roleName string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmRolesRoleNameCompositesWithResponse(ctx, realm, roleName, roles)
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

func (c *rolesClient) DeleteRealmRoleComposites(
	ctx context.Context,
	realm, roleName string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmRolesRoleNameCompositesWithResponse(ctx, realm, roleName, roles)
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
