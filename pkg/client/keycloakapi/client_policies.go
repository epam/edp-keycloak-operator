package keycloakapi

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
)

type (
	ClientPoliciesRepresentation = generated.ClientPoliciesRepresentation
	ClientProfilesRepresentation = generated.ClientProfilesRepresentation
	GetClientPoliciesParams      = generated.GetAdminRealmsRealmClientPoliciesPoliciesParams
	GetClientProfilesParams      = generated.GetAdminRealmsRealmClientPoliciesProfilesParams
)

// ClientPoliciesClient defines operations for managing Keycloak client policies and profiles.
type ClientPoliciesClient interface {
	// GetClientPolicies returns the client policies configuration for a realm.
	GetClientPolicies(
		ctx context.Context, realm string, params *GetClientPoliciesParams,
	) (*ClientPoliciesRepresentation, *Response, error)
	// UpdateClientPolicies updates the client policies configuration for a realm.
	UpdateClientPolicies(
		ctx context.Context, realm string, policies ClientPoliciesRepresentation,
	) (*Response, error)
	// GetClientProfiles returns the client profiles configuration for a realm.
	GetClientProfiles(
		ctx context.Context, realm string, params *GetClientProfilesParams,
	) (*ClientProfilesRepresentation, *Response, error)
	// UpdateClientProfiles updates the client profiles configuration for a realm.
	UpdateClientProfiles(
		ctx context.Context, realm string, profiles ClientProfilesRepresentation,
	) (*Response, error)
}

type clientPoliciesClient struct {
	client generated.ClientWithResponsesInterface
}

var _ ClientPoliciesClient = (*clientPoliciesClient)(nil)

func (c *clientPoliciesClient) GetClientPolicies(
	ctx context.Context,
	realm string,
	params *GetClientPoliciesParams,
) (*ClientPoliciesRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientPoliciesPoliciesWithResponse(ctx, realm, params)
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

func (c *clientPoliciesClient) UpdateClientPolicies(
	ctx context.Context,
	realm string,
	policies ClientPoliciesRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmClientPoliciesPoliciesWithResponse(ctx, realm, policies)
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

func (c *clientPoliciesClient) GetClientProfiles(
	ctx context.Context,
	realm string,
	params *GetClientProfilesParams,
) (*ClientProfilesRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmClientPoliciesProfilesWithResponse(ctx, realm, params)
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

func (c *clientPoliciesClient) UpdateClientProfiles(
	ctx context.Context,
	realm string,
	profiles ClientProfilesRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmClientPoliciesProfilesWithResponse(ctx, realm, profiles)
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
