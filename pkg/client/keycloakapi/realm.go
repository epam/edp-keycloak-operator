package keycloakapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
)

// Type aliases for generated types
type RealmRepresentation = generated.RealmRepresentation
type RealmEventsConfigRepresentation = generated.RealmEventsConfigRepresentation
type KeysMetadataRepresentation = generated.KeysMetadataRepresentation
type GetRealmLocalizationParams = generated.GetAdminRealmsRealmLocalizationLocaleParams

type realmClient struct {
	client generated.ClientWithResponsesInterface
}

// Ensure realmClient implements RealmClient
var _ RealmClient = (*realmClient)(nil)

func (c *realmClient) GetRealm(ctx context.Context, realm string) (*RealmRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmWithResponse(ctx, realm)
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

func (c *realmClient) CreateRealm(ctx context.Context, realmRep RealmRepresentation) (*Response, error) {
	// Note: The generated PostAdminRealmsJSONBody is incorrectly typed as openapi_types.File
	// in the OpenAPI spec, but Keycloak actually accepts RealmRepresentation JSON.
	// We use PostAdminRealmsWithBodyWithResponse with manual JSON marshaling as a workaround.
	body, err := json.Marshal(realmRep)
	if err != nil {
		return nil, err
	}

	res, err := c.client.PostAdminRealmsWithBodyWithResponse(ctx, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	// Check for non-2xx status codes and return ApiError
	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *realmClient) UpdateRealm(
	ctx context.Context,
	realm string,
	realmRep RealmRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmWithResponse(ctx, realm, realmRep)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	// Check for non-2xx status codes and return ApiError
	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *realmClient) DeleteRealm(ctx context.Context, realm string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmWithResponse(ctx, realm)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrNilResponse
	}

	response := &Response{HTTPResponse: res.HTTPResponse, Body: res.Body}

	// Check for non-2xx status codes and return ApiError
	if err := checkResponseError(res.HTTPResponse, res.Body); err != nil {
		return response, err
	}

	return response, nil
}

func (c *realmClient) SetRealmBrowserFlow(ctx context.Context, realm string, flowAlias string) (*Response, error) {
	current, resp, err := c.GetRealm(ctx, realm)
	if err != nil {
		return resp, fmt.Errorf("unable to get realm: %w", err)
	}

	current.BrowserFlow = &flowAlias

	return c.UpdateRealm(ctx, realm, *current)
}

func (c *realmClient) GetAuthenticationFlows(
	ctx context.Context,
	realm string,
) ([]AuthenticationFlowRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmAuthenticationFlowsWithResponse(ctx, realm)
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

func (c *realmClient) GetRealms(ctx context.Context) ([]RealmRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsWithResponse(ctx, nil)
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

func (c *realmClient) GetRealmKeys(
	ctx context.Context,
	realm string,
) (*KeysMetadataRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmKeysWithResponse(ctx, realm)
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

func (c *realmClient) GetRealmLocalization(
	ctx context.Context,
	realm, locale string,
) (map[string]string, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmLocalizationLocaleWithResponse(ctx, realm, locale, nil)
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

func (c *realmClient) PostRealmLocalization(
	ctx context.Context,
	realm, locale string,
	texts map[string]string,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmLocalizationLocaleWithResponse(ctx, realm, locale, texts)
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
