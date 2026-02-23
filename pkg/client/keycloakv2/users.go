package keycloakv2

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type (
	UserProfileConfig               = generated.UPConfig
	UserProfileAttribute            = generated.UPAttribute
	UserProfileAttributePermissions = generated.UPAttributePermissions
	UserProfileAttributeRequired    = generated.UPAttributeRequired
	UserProfileAttributeSelector    = generated.UPAttributeSelector
	UserProfileGroup                = generated.UPGroup
	UnmanagedAttributePolicy        = generated.UnmanagedAttributePolicy
	UserRepresentation              = generated.UserRepresentation
)

type usersClient struct {
	client generated.ClientWithResponsesInterface
}

// Ensure usersClient implements UsersClient
var _ UsersClient = (*usersClient)(nil)

func (c *usersClient) GetUsersProfile(ctx context.Context, realm string) (*UserProfileConfig, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersProfileWithResponse(ctx, realm)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	// Check for non-2xx status codes and return ApiError
	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.JSON200, response, nil
}

func (c *usersClient) UpdateUsersProfile(
	ctx context.Context,
	realm string,
	userProfile UserProfileConfig,
) (*UserProfileConfig, *Response, error) {
	res, err := c.client.PutAdminRealmsRealmUsersProfileWithResponse(ctx, realm, userProfile)
	if err != nil {
		return nil, nil, err
	}

	if res == nil {
		return nil, nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	// Check for non-2xx status codes and return ApiError
	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return nil, response, err
	}

	return res.JSON200, response, nil
}

func (c *usersClient) FindUserByUsername(
	ctx context.Context,
	realm, username string,
) (*UserRepresentation, *Response, error) {
	exact := true

	res, err := c.client.GetAdminRealmsRealmUsersWithResponse(ctx, realm, &generated.GetAdminRealmsRealmUsersParams{
		Username: &username,
		Exact:    &exact,
	})
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

	if res.JSON200 == nil || len(*res.JSON200) == 0 {
		return nil, response, nil
	}

	user := (*res.JSON200)[0]

	return &user, response, nil
}

func (c *usersClient) CreateUser(ctx context.Context, realm string, user UserRepresentation) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmUsersWithResponse(ctx, realm, user)
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

func (c *usersClient) GetUserRealmRoleMappings(
	ctx context.Context,
	realm, userID string,
) ([]RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersUserIdRoleMappingsRealmWithResponse(ctx, realm, userID)
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

func (c *usersClient) AddUserRealmRoles(
	ctx context.Context,
	realm, userID string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmUsersUserIdRoleMappingsRealmWithResponse(ctx, realm, userID, roles)
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
