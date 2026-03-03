package keycloakv2

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type (
	OrganizationRepresentation       = generated.OrganizationRepresentation
	OrganizationDomainRepresentation = generated.OrganizationDomainRepresentation
	GetOrganizationsParams           = generated.GetAdminRealmsRealmOrganizationsParams
)

type organizationsClient struct {
	client generated.ClientWithResponsesInterface
}

var _ OrganizationsClient = (*organizationsClient)(nil)

func (c *organizationsClient) GetOrganizations(
	ctx context.Context,
	realm string,
	params *GetOrganizationsParams,
) ([]OrganizationRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmOrganizationsWithResponse(ctx, realm, params)
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

func (c *organizationsClient) GetOrganizationByAlias(
	ctx context.Context,
	realm, alias string,
) (*OrganizationRepresentation, *Response, error) {
	orgs, response, err := c.GetOrganizations(ctx, realm, &GetOrganizationsParams{
		Q: ptr.To("alias:" + alias),
	})
	if err != nil {
		return nil, response, err
	}

	for i := range orgs {
		if orgs[i].Alias != nil && *orgs[i].Alias == alias {
			return &orgs[i], response, nil
		}
	}

	return nil, response, &ApiError{
		Code:    http.StatusNotFound,
		Message: fmt.Sprintf("organization with alias %s not found", alias),
	}
}

func (c *organizationsClient) CreateOrganization(
	ctx context.Context,
	realm string,
	org OrganizationRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmOrganizationsWithResponse(ctx, realm, org)
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

func (c *organizationsClient) UpdateOrganization(
	ctx context.Context,
	realm, orgID string,
	org OrganizationRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmOrganizationsOrgIdWithResponse(ctx, realm, orgID, org)
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

func (c *organizationsClient) DeleteOrganization(
	ctx context.Context,
	realm, orgID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmOrganizationsOrgIdWithResponse(ctx, realm, orgID)
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

func (c *organizationsClient) GetOrganizationIdentityProviders(
	ctx context.Context,
	realm, orgID string,
) ([]IdentityProviderRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmOrganizationsOrgIdIdentityProvidersWithResponse(ctx, realm, orgID)
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

func (c *organizationsClient) LinkIdentityProviderToOrganization(
	ctx context.Context,
	realm, orgID, alias string,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmOrganizationsOrgIdIdentityProvidersWithResponse(ctx, realm, orgID, alias)
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

func (c *organizationsClient) UnlinkIdentityProviderFromOrganization(
	ctx context.Context,
	realm, orgID, alias string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmOrganizationsOrgIdIdentityProvidersAliasWithResponse(
		ctx, realm, orgID, alias)
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
