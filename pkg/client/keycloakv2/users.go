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
