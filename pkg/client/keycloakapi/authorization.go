package keycloakapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
)

type (
	ScopeRepresentation              = generated.ScopeRepresentation
	ResourceRepresentation           = generated.ResourceRepresentation
	PolicyRepresentation             = generated.PolicyRepresentation
	AbstractPolicyRepresentation     = generated.AbstractPolicyRepresentation
	DecisionStrategy                 = generated.DecisionStrategy
	Logic                            = generated.Logic
	AuthenticationFlowRepresentation = generated.AuthenticationFlowRepresentation

	GetAuthzScopesParams      = generated.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerScopeParams
	GetAuthzResourcesParams   = generated.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerResourceParams
	GetAuthzPoliciesParams    = generated.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerPolicyParams
	GetAuthzPermissionsParams = generated.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerPermissionParams
	PostAuthzResourcesParams  = generated.PostAdminRealmsRealmClientsClientUuidAuthzResourceServerResourceParams
)

type authorizationClient struct {
	client generated.ClientWithResponsesInterface
	kc     *KeycloakClient
}

var _ AuthorizationClient = (*authorizationClient)(nil)

// Scopes

func (a *authorizationClient) GetScopes(
	ctx context.Context,
	realm string,
	clientUUID string,
) ([]ScopeRepresentation, *Response, error) {
	res, err := a.client.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerScopeWithResponse(
		ctx, realm, clientUUID, nil,
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

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (a *authorizationClient) CreateScope(
	ctx context.Context,
	realm string,
	clientUUID string,
	scope ScopeRepresentation,
) (*Response, error) {
	res, err := a.client.PostAdminRealmsRealmClientsClientUuidAuthzResourceServerScopeWithResponse(
		ctx, realm, clientUUID, scope,
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

func (a *authorizationClient) GetScope(
	ctx context.Context,
	realm string,
	clientUUID string,
	scopeID string,
) (*ScopeRepresentation, *Response, error) {
	res, err := a.client.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerScopeScopeIdWithResponse(
		ctx, realm, clientUUID, scopeID,
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

func (a *authorizationClient) UpdateScope(
	ctx context.Context,
	realm string,
	clientUUID string,
	scopeID string,
	scope ScopeRepresentation,
) (*Response, error) {
	res, err := a.client.PutAdminRealmsRealmClientsClientUuidAuthzResourceServerScopeScopeIdWithResponse(
		ctx, realm, clientUUID, scopeID, scope,
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

func (a *authorizationClient) DeleteScope(
	ctx context.Context,
	realm string,
	clientUUID string,
	scopeID string,
) (*Response, error) {
	res, err := a.client.DeleteAdminRealmsRealmClientsClientUuidAuthzResourceServerScopeScopeIdWithResponse(
		ctx, realm, clientUUID, scopeID,
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

// Resources

func (a *authorizationClient) GetResources(
	ctx context.Context,
	realm string,
	clientUUID string,
) ([]ResourceRepresentation, *Response, error) {
	res, err := a.client.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerResourceWithResponse(
		ctx, realm, clientUUID, nil,
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

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (a *authorizationClient) GetResource(
	ctx context.Context,
	realm string,
	clientUUID string,
	resourceID string,
) (*ResourceRepresentation, *Response, error) {
	res, err := a.client.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerResourceResourceIdWithResponse(
		ctx, realm, clientUUID, resourceID, nil,
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

func (a *authorizationClient) CreateResource(
	ctx context.Context,
	realm string,
	clientUUID string,
	resource ResourceRepresentation,
) (*ResourceRepresentation, *Response, error) {
	res, err := a.client.PostAdminRealmsRealmClientsClientUuidAuthzResourceServerResourceWithResponse(
		ctx, realm, clientUUID, nil, resource,
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

	return res.JSON201, response, nil
}

func (a *authorizationClient) UpdateResource(
	ctx context.Context,
	realm string,
	clientUUID string,
	resourceID string,
	resource ResourceRepresentation,
) (*Response, error) {
	res, err := a.client.PutAdminRealmsRealmClientsClientUuidAuthzResourceServerResourceResourceIdWithResponse(
		ctx, realm, clientUUID, resourceID, nil, resource,
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

func (a *authorizationClient) DeleteResource(
	ctx context.Context,
	realm string,
	clientUUID string,
	resourceID string,
) (*Response, error) {
	res, err := a.client.DeleteAdminRealmsRealmClientsClientUuidAuthzResourceServerResourceResourceIdWithResponse(
		ctx, realm, clientUUID, resourceID, nil,
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

// Policies - List uses generated client, CRUD by ID uses custom HTTP (not in OpenAPI spec)

func (a *authorizationClient) GetPolicies(
	ctx context.Context,
	realm string,
	clientUUID string,
) ([]AbstractPolicyRepresentation, *Response, error) {
	res, err := a.client.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerPolicyWithResponse(
		ctx, realm, clientUUID, nil,
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

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (a *authorizationClient) CreatePolicy(
	ctx context.Context,
	realm string,
	clientUUID string,
	policyType string,
	policy any,
) (*PolicyRepresentation, *Response, error) {
	reqURL := fmt.Sprintf(
		"%s/admin/realms/%s/clients/%s/authz/resource-server/policy/%s",
		a.kc.baseUrl, url.PathEscape(realm), url.PathEscape(clientUUID), url.PathEscape(policyType),
	)

	return a.doJSONRequest(ctx, http.MethodPost, reqURL, policy)
}

func (a *authorizationClient) UpdatePolicy(
	ctx context.Context,
	realm string,
	clientUUID string,
	policyType string,
	policyID string,
	policy any,
) (*Response, error) {
	reqURL := fmt.Sprintf(
		"%s/admin/realms/%s/clients/%s/authz/resource-server/policy/%s/%s",
		a.kc.baseUrl, url.PathEscape(realm), url.PathEscape(clientUUID), url.PathEscape(policyType), url.PathEscape(policyID),
	)

	_, resp, err := a.doJSONRequest(ctx, http.MethodPut, reqURL, policy)

	return resp, err
}

func (a *authorizationClient) GetPolicy(
	ctx context.Context,
	realm string,
	clientUUID string,
	policyType string,
	policyID string,
) (*Response, error) {
	reqURL := fmt.Sprintf(
		"%s/admin/realms/%s/clients/%s/authz/resource-server/policy/%s/%s",
		a.kc.baseUrl, url.PathEscape(realm), url.PathEscape(clientUUID), url.PathEscape(policyType), url.PathEscape(policyID),
	)

	_, resp, err := a.doJSONRequest(ctx, http.MethodGet, reqURL, nil)

	return resp, err
}

func (a *authorizationClient) DeletePolicy(
	ctx context.Context,
	realm string,
	clientUUID string,
	policyID string,
) (*Response, error) {
	reqURL := fmt.Sprintf(
		"%s/admin/realms/%s/clients/%s/authz/resource-server/policy/%s",
		a.kc.baseUrl, url.PathEscape(realm), url.PathEscape(clientUUID), url.PathEscape(policyID),
	)

	_, resp, err := a.doJSONRequest(ctx, http.MethodDelete, reqURL, nil)

	return resp, err
}

// Permissions - same pattern as Policies

func (a *authorizationClient) GetPermissions(
	ctx context.Context,
	realm string,
	clientUUID string,
) ([]AbstractPolicyRepresentation, *Response, error) {
	res, err := a.client.GetAdminRealmsRealmClientsClientUuidAuthzResourceServerPermissionWithResponse(
		ctx, realm, clientUUID, nil,
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

	if res.JSON200 == nil {
		return nil, response, nil
	}

	return *res.JSON200, response, nil
}

func (a *authorizationClient) CreatePermission(
	ctx context.Context,
	realm string,
	clientUUID string,
	permType string,
	perm PolicyRepresentation,
) (*PolicyRepresentation, *Response, error) {
	reqURL := fmt.Sprintf(
		"%s/admin/realms/%s/clients/%s/authz/resource-server/permission/%s",
		a.kc.baseUrl, url.PathEscape(realm), url.PathEscape(clientUUID), url.PathEscape(permType),
	)

	return a.doJSONRequest(ctx, http.MethodPost, reqURL, perm)
}

func (a *authorizationClient) UpdatePermission(
	ctx context.Context,
	realm string,
	clientUUID string,
	permType string,
	permID string,
	perm PolicyRepresentation,
) (*Response, error) {
	reqURL := fmt.Sprintf(
		"%s/admin/realms/%s/clients/%s/authz/resource-server/permission/%s/%s",
		a.kc.baseUrl, url.PathEscape(realm), url.PathEscape(clientUUID), url.PathEscape(permType), url.PathEscape(permID),
	)

	_, resp, err := a.doJSONRequest(ctx, http.MethodPut, reqURL, perm)

	return resp, err
}

func (a *authorizationClient) DeletePermission(
	ctx context.Context,
	realm string,
	clientUUID string,
	permID string,
) (*Response, error) {
	reqURL := fmt.Sprintf(
		"%s/admin/realms/%s/clients/%s/authz/resource-server/permission/%s",
		a.kc.baseUrl, url.PathEscape(realm), url.PathEscape(clientUUID), url.PathEscape(permID),
	)

	_, resp, err := a.doJSONRequest(ctx, http.MethodDelete, reqURL, nil)

	return resp, err
}

// doJSONRequest is a helper for custom HTTP calls not available in the generated client.
func (a *authorizationClient) doJSONRequest(
	ctx context.Context,
	method string,
	reqURL string,
	body any,
) (*PolicyRepresentation, *Response, error) {
	var bodyReader io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal request body: %w", err)
		}

		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := (&keycloakDoer{kc: a.kc}).Do(req)
	if err != nil {
		return nil, nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	response := &Response{HTTPResponse: resp, Body: respBody}

	if err := checkResponseError(resp, respBody); err != nil {
		return nil, response, err
	}

	if method == http.MethodPost && len(respBody) > 0 {
		var result PolicyRepresentation
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, response, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return &result, response, nil
	}

	return nil, response, nil
}
