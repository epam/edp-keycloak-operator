package keycloakv2

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

// Type aliases for generated types
type RealmRepresentation = generated.RealmRepresentation

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
