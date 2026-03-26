package keycloakv2

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type IdentityProviderRepresentation = generated.IdentityProviderRepresentation
type IdentityProviderMapperRepresentation = generated.IdentityProviderMapperRepresentation

type identityProvidersClient struct {
	client generated.ClientWithResponsesInterface
}

var _ IdentityProvidersClient = (*identityProvidersClient)(nil)

func (c *identityProvidersClient) GetIdentityProvider(
	ctx context.Context,
	realm, alias string,
) (*IdentityProviderRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmIdentityProviderInstancesAliasWithResponse(ctx, realm, alias)
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

func (c *identityProvidersClient) CreateIdentityProvider(
	ctx context.Context,
	realm string,
	idp IdentityProviderRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmIdentityProviderInstancesWithResponse(ctx, realm, idp)
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

func (c *identityProvidersClient) UpdateIdentityProvider(
	ctx context.Context,
	realm, alias string,
	idp IdentityProviderRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmIdentityProviderInstancesAliasWithResponse(ctx, realm, alias, idp)
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

func (c *identityProvidersClient) DeleteIdentityProvider(
	ctx context.Context,
	realm, alias string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmIdentityProviderInstancesAliasWithResponse(ctx, realm, alias)
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

func (c *identityProvidersClient) GetIDPMappers(
	ctx context.Context,
	realm, alias string,
) ([]IdentityProviderMapperRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmIdentityProviderInstancesAliasMappersWithResponse(ctx, realm, alias)
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

func (c *identityProvidersClient) CreateIDPMapper(
	ctx context.Context,
	realm, alias string,
	mapper IdentityProviderMapperRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmIdentityProviderInstancesAliasMappersWithResponse(ctx, realm, alias, mapper)
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

func (c *identityProvidersClient) DeleteIDPMapper(
	ctx context.Context,
	realm, alias, mapperID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmIdentityProviderInstancesAliasMappersIdWithResponse(
		ctx, realm, alias, mapperID,
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

func (c *identityProvidersClient) GetIDPManagementPermissions(
	ctx context.Context,
	realm, alias string,
) (*ManagementPermissionReference, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmIdentityProviderInstancesAliasManagementPermissionsWithResponse(
		ctx, realm, alias,
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

func (c *identityProvidersClient) UpdateIDPManagementPermissions(
	ctx context.Context,
	realm, alias string,
	permissions ManagementPermissionReference,
) (*ManagementPermissionReference, *Response, error) {
	res, err := c.client.PutAdminRealmsRealmIdentityProviderInstancesAliasManagementPermissionsWithResponse(
		ctx, realm, alias, permissions,
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
