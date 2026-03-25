package keycloakv2

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type clientScopesClient struct {
	client generated.ClientWithResponsesInterface
}

var _ ClientScopesClient = (*clientScopesClient)(nil)

func (c *clientScopesClient) GetClientScopes(
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

func (c *clientScopesClient) GetClientScope(
	ctx context.Context,
	realm, scopeID string,
) (*ClientScopeRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientScopesClientScopeIdWithResponse(ctx, realm, scopeID)
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

func (c *clientScopesClient) CreateClientScope(
	ctx context.Context,
	realm string,
	scope ClientScopeRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmClientScopesWithResponse(ctx, realm, scope)
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

func (c *clientScopesClient) UpdateClientScope(
	ctx context.Context,
	realm, scopeID string,
	scope ClientScopeRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmClientScopesClientScopeIdWithResponse(ctx, realm, scopeID, scope)
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

func (c *clientScopesClient) DeleteClientScope(
	ctx context.Context,
	realm, scopeID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmClientScopesClientScopeIdWithResponse(ctx, realm, scopeID)
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

func (c *clientScopesClient) GetRealmDefaultClientScopes(
	ctx context.Context,
	realm string,
) ([]ClientScopeRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmDefaultDefaultClientScopesWithResponse(ctx, realm)
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

func (c *clientScopesClient) AddRealmDefaultClientScope(
	ctx context.Context,
	realm, scopeID string,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmDefaultDefaultClientScopesClientScopeIdWithResponse(ctx, realm, scopeID)
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

func (c *clientScopesClient) RemoveRealmDefaultClientScope(
	ctx context.Context,
	realm, scopeID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmDefaultDefaultClientScopesClientScopeIdWithResponse(ctx, realm, scopeID)
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

func (c *clientScopesClient) GetRealmOptionalClientScopes(
	ctx context.Context,
	realm string,
) ([]ClientScopeRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmDefaultOptionalClientScopesWithResponse(ctx, realm)
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

func (c *clientScopesClient) AddRealmOptionalClientScope(
	ctx context.Context,
	realm, scopeID string,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmDefaultOptionalClientScopesClientScopeIdWithResponse(ctx, realm, scopeID)
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

func (c *clientScopesClient) RemoveRealmOptionalClientScope(
	ctx context.Context,
	realm, scopeID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmDefaultOptionalClientScopesClientScopeIdWithResponse(ctx, realm, scopeID)
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

func (c *clientScopesClient) GetClientScopeProtocolMappers(
	ctx context.Context,
	realm, scopeID string,
) ([]ProtocolMapperRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientScopesClientScopeIdProtocolMappersModelsWithResponse(ctx, realm, scopeID)
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

func (c *clientScopesClient) CreateClientScopeProtocolMapper(
	ctx context.Context,
	realm, scopeID string,
	mapper ProtocolMapperRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmClientScopesClientScopeIdProtocolMappersModelsWithResponse(
		ctx, realm, scopeID, mapper,
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

func (c *clientScopesClient) DeleteClientScopeProtocolMapper(
	ctx context.Context,
	realm, scopeID, mapperID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmClientScopesClientScopeIdProtocolMappersModelsIdWithResponse(
		ctx, realm, scopeID, mapperID,
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
