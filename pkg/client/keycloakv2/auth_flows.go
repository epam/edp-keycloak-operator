package keycloakv2

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

// AuthenticationExecution requirement constants.
const (
	AuthExecutionRequirementRequired    = "REQUIRED"
	AuthExecutionRequirementAlternative = "ALTERNATIVE"
	AuthExecutionRequirementDisabled    = "DISABLED"
	AuthExecutionRequirementConditional = "CONDITIONAL"
)

// Type aliases for generated auth flow types.
type AuthFlowRepresentation = generated.AuthenticationFlowRepresentation
type AuthenticationExecutionInfoRepresentation = generated.AuthenticationExecutionInfoRepresentation
type AuthenticatorConfigRepresentation = generated.AuthenticatorConfigRepresentation
type AuthenticationExecutionRepresentation = generated.AuthenticationExecutionRepresentation

// AuthFlowsClient defines operations for managing Keycloak authentication flows.
type AuthFlowsClient interface {
	GetAuthFlows(ctx context.Context, realm string) ([]AuthFlowRepresentation, *Response, error)
	CreateAuthFlow(ctx context.Context, realm string, body AuthFlowRepresentation) (*Response, error)
	UpdateAuthFlow(ctx context.Context, realm, id string, body AuthFlowRepresentation) (*Response, error)
	DeleteAuthFlow(ctx context.Context, realm, id string) (*Response, error)
	GetFlowExecutions(ctx context.Context, realm, flowAlias string) (
		[]AuthenticationExecutionInfoRepresentation, *Response, error)
	UpdateFlowExecution(
		ctx context.Context, realm, flowAlias string, body AuthenticationExecutionInfoRepresentation,
	) (*Response, error)
	// AddExecutionToFlow adds a non-flow execution. The body must include ParentFlow (the flow's Keycloak ID).
	// Keycloak returns HTTP 201 with a Location header — use GetResourceIDFromResponse to extract the execution ID.
	AddExecutionToFlow(ctx context.Context, realm string, body AuthenticationExecutionRepresentation) (*Response, error)
	// AddChildFlowToFlow adds a sub-flow under the given parent flow alias.
	AddChildFlowToFlow(ctx context.Context, realm, flowAlias string, body map[string]any) (*Response, error)
	DeleteExecution(ctx context.Context, realm, executionID string) (*Response, error)
	CreateExecutionConfig(
		ctx context.Context, realm, executionID string, body AuthenticatorConfigRepresentation,
	) (*Response, error)
	GetAuthenticatorConfig(
		ctx context.Context, realm, configID string,
	) (*AuthenticatorConfigRepresentation, *Response, error)
}

type authFlowsClient struct {
	client generated.ClientWithResponsesInterface
}

var _ AuthFlowsClient = (*authFlowsClient)(nil)

func (c *authFlowsClient) GetAuthFlows(ctx context.Context, realm string) ([]AuthFlowRepresentation, *Response, error) {
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

func (c *authFlowsClient) CreateAuthFlow(
	ctx context.Context, realm string, body AuthFlowRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmAuthenticationFlowsWithResponse(ctx, realm, body)
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

func (c *authFlowsClient) UpdateAuthFlow(
	ctx context.Context, realm, id string, body AuthFlowRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmAuthenticationFlowsIdWithResponse(ctx, realm, id, body)
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

func (c *authFlowsClient) DeleteAuthFlow(ctx context.Context, realm, id string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmAuthenticationFlowsIdWithResponse(ctx, realm, id)
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

func (c *authFlowsClient) GetFlowExecutions(
	ctx context.Context, realm, flowAlias string,
) ([]AuthenticationExecutionInfoRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmAuthenticationFlowsFlowAliasExecutionsWithResponse(ctx, realm, flowAlias)
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

func (c *authFlowsClient) UpdateFlowExecution(
	ctx context.Context, realm, flowAlias string, body AuthenticationExecutionInfoRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmAuthenticationFlowsFlowAliasExecutionsWithResponse(ctx, realm, flowAlias, body)
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

func (c *authFlowsClient) AddExecutionToFlow(
	ctx context.Context, realm string, body AuthenticationExecutionRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmAuthenticationExecutionsWithResponse(ctx, realm, body)
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

func (c *authFlowsClient) AddChildFlowToFlow(
	ctx context.Context, realm, flowAlias string, body map[string]any,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmAuthenticationFlowsFlowAliasExecutionsFlowWithResponse(
		ctx, realm, flowAlias, body)
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

func (c *authFlowsClient) DeleteExecution(ctx context.Context, realm, executionID string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmAuthenticationExecutionsExecutionIdWithResponse(ctx, realm, executionID)
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

func (c *authFlowsClient) CreateExecutionConfig(
	ctx context.Context, realm, executionID string, body AuthenticatorConfigRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmAuthenticationExecutionsExecutionIdConfigWithResponse(
		ctx, realm, executionID, body)
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

func (c *authFlowsClient) GetAuthenticatorConfig(
	ctx context.Context, realm, configID string,
) (*AuthenticatorConfigRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmAuthenticationConfigIdWithResponse(ctx, realm, configID)
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
