package keycloakapi

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
)

type (
	ComponentRepresentation = generated.ComponentRepresentation
	GetComponentsParams     = generated.GetAdminRealmsRealmComponentsParams
)

// MultivaluedHashMapStringString is a map of string to slice of strings, used for component config.
type MultivaluedHashMapStringString = generated.MultivaluedHashMapStringString

type realmComponentsClient struct {
	client generated.ClientWithResponsesInterface
}

var _ RealmComponentsClient = (*realmComponentsClient)(nil)

func (c *realmComponentsClient) GetComponents(
	ctx context.Context,
	realm string,
	params *GetComponentsParams,
) ([]ComponentRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmComponentsWithResponse(ctx, realm, params)
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

func (c *realmComponentsClient) GetComponent(
	ctx context.Context,
	realm, componentID string,
) (*ComponentRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmComponentsIdWithResponse(ctx, realm, componentID)
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

func (c *realmComponentsClient) FindComponentByName(
	ctx context.Context,
	realm, componentName string,
) (*ComponentRepresentation, error) {
	components, _, err := c.GetComponents(ctx, realm, &GetComponentsParams{Name: &componentName})
	if err != nil {
		return nil, fmt.Errorf("failed to get components: %w", err)
	}

	for i := range components {
		if components[i].Name != nil && *components[i].Name == componentName {
			return &components[i], nil
		}
	}

	return nil, nil
}

func (c *realmComponentsClient) CreateComponent(
	ctx context.Context,
	realm string,
	component ComponentRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmComponentsWithResponse(ctx, realm, component)
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

func (c *realmComponentsClient) UpdateComponent(
	ctx context.Context,
	realm, componentID string,
	component ComponentRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmComponentsIdWithResponse(ctx, realm, componentID, component)
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

func (c *realmComponentsClient) DeleteComponent(
	ctx context.Context,
	realm, componentID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmComponentsIdWithResponse(ctx, realm, componentID)
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
