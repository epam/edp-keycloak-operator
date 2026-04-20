package keycloakapi

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
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
type RequiredActionProviderRepresentation = generated.RequiredActionProviderRepresentation

// AuthFlowsClient defines operations for managing Keycloak authentication flows.
type AuthFlowsClient interface {
	// GetAuthFlows returns all authentication flows for a realm.
	GetAuthFlows(ctx context.Context, realm string) ([]AuthFlowRepresentation, *Response, error)
	// GetAuthFlow retrieves a single authentication flow by its Keycloak UUID.
	GetAuthFlow(ctx context.Context, realm, flowID string) (*AuthFlowRepresentation, *Response, error)
	// CreateAuthFlow creates a new top-level authentication flow.
	CreateAuthFlow(ctx context.Context, realm string, body AuthFlowRepresentation) (*Response, error)
	// UpdateAuthFlow updates an existing authentication flow.
	UpdateAuthFlow(ctx context.Context, realm, id string, body AuthFlowRepresentation) (*Response, error)
	// DeleteAuthFlow deletes an authentication flow by its Keycloak UUID.
	DeleteAuthFlow(ctx context.Context, realm, id string) (*Response, error)
	// CopyAuthFlow copies an existing flow under a new name. This is the standard Keycloak
	// pattern for customizing built-in flows: copy, then modify.
	CopyAuthFlow(ctx context.Context, realm, flowAlias, newName string) (*Response, error)
	// GetFlowExecutions returns all executions within an authentication flow.
	GetFlowExecutions(ctx context.Context, realm, flowAlias string) (
		[]AuthenticationExecutionInfoRepresentation, *Response, error)
	// UpdateFlowExecution updates an execution within an authentication flow (e.g., change requirement).
	UpdateFlowExecution(
		ctx context.Context, realm, flowAlias string, body AuthenticationExecutionInfoRepresentation,
	) (*Response, error)
	// AddExecutionToFlow adds a non-flow execution. The body must include ParentFlow (the flow's Keycloak ID).
	// Keycloak returns HTTP 201 with a Location header — use GetResourceIDFromResponse to extract the execution ID.
	AddExecutionToFlow(ctx context.Context, realm string, body AuthenticationExecutionRepresentation) (*Response, error)
	// AddChildFlowToFlow adds a sub-flow under the given parent flow alias.
	// The body map should include the following keys:
	//   - "alias" (string, required): alias for the new sub-flow
	//   - "type" (string): e.g., "basic-flow"
	//   - "provider" (string): e.g., "registration-page-form"
	//   - "description" (string): human-readable description
	AddChildFlowToFlow(ctx context.Context, realm, flowAlias string, body map[string]any) (*Response, error)
	// DeleteExecution deletes an authentication execution by its Keycloak UUID.
	DeleteExecution(ctx context.Context, realm, executionID string) (*Response, error)
	// CreateExecutionConfig creates a configuration for an authentication execution.
	CreateExecutionConfig(
		ctx context.Context, realm, executionID string, body AuthenticatorConfigRepresentation,
	) (*Response, error)
	// GetAuthenticatorConfig retrieves an authenticator configuration by its Keycloak UUID.
	GetAuthenticatorConfig(
		ctx context.Context, realm, configID string,
	) (*AuthenticatorConfigRepresentation, *Response, error)
	// UpdateAuthenticatorConfig updates an existing authenticator configuration.
	UpdateAuthenticatorConfig(
		ctx context.Context, realm, configID string, body AuthenticatorConfigRepresentation,
	) (*Response, error)
	// DeleteAuthenticatorConfig removes an authenticator configuration.
	DeleteAuthenticatorConfig(ctx context.Context, realm, configID string) (*Response, error)
	// GetRequiredActions returns all required actions for a realm (e.g., verify email, update profile).
	GetRequiredActions(ctx context.Context, realm string) ([]RequiredActionProviderRepresentation, *Response, error)
	// UpdateRequiredAction updates a required action (enable/disable/reorder).
	UpdateRequiredAction(
		ctx context.Context, realm, alias string, action RequiredActionProviderRepresentation,
	) (*Response, error)
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

func (c *authFlowsClient) GetAuthFlow(
	ctx context.Context, realm, flowID string,
) (*AuthFlowRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmAuthenticationFlowsIdWithResponse(ctx, realm, flowID)
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

func (c *authFlowsClient) CopyAuthFlow(
	ctx context.Context, realm, flowAlias, newName string,
) (*Response, error) {
	body := generated.PostAdminRealmsRealmAuthenticationFlowsFlowAliasCopyJSONRequestBody{
		"newName": newName,
	}

	res, err := c.client.PostAdminRealmsRealmAuthenticationFlowsFlowAliasCopyWithResponse(ctx, realm, flowAlias, body)
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

func (c *authFlowsClient) UpdateAuthenticatorConfig(
	ctx context.Context, realm, configID string, body AuthenticatorConfigRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmAuthenticationConfigIdWithResponse(ctx, realm, configID, body)
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

func (c *authFlowsClient) DeleteAuthenticatorConfig(ctx context.Context, realm, configID string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmAuthenticationConfigIdWithResponse(ctx, realm, configID)
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

func (c *authFlowsClient) GetRequiredActions(
	ctx context.Context, realm string,
) ([]RequiredActionProviderRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmAuthenticationRequiredActionsWithResponse(ctx, realm)
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

func (c *authFlowsClient) UpdateRequiredAction(
	ctx context.Context, realm, alias string, action RequiredActionProviderRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmAuthenticationRequiredActionsAliasWithResponse(ctx, realm, alias, action)
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
