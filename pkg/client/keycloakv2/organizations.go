package keycloakv2

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type (
	OrganizationRepresentation              = generated.OrganizationRepresentation
	OrganizationDomainRepresentation        = generated.OrganizationDomainRepresentation
	GetOrganizationsParams                  = generated.GetAdminRealmsRealmOrganizationsParams
	MemberRepresentation                    = generated.MemberRepresentation
	GetOrganizationMembersParams            = generated.GetAdminRealmsRealmOrganizationsOrgIdMembersParams
	InviteExistingOrganizationMemberRequest = generated.PostAdminRealmsRealmOrganizationsOrgIdMembersInviteExistingUserFormdataRequestBody //nolint:lll // generated type alias
	InviteNewOrganizationMemberRequest      = generated.PostAdminRealmsRealmOrganizationsOrgIdMembersInviteUserFormdataRequestBody         //nolint:lll // generated type alias
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

func (c *organizationsClient) GetOrganizationMembers(
	ctx context.Context,
	realm, orgID string,
	params *GetOrganizationMembersParams,
) ([]MemberRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmOrganizationsOrgIdMembersWithResponse(ctx, realm, orgID, params)
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

func (c *organizationsClient) AddOrganizationMember(
	ctx context.Context,
	realm, orgID, userID string,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmOrganizationsOrgIdMembersWithResponse(ctx, realm, orgID, userID)
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

func (c *organizationsClient) RemoveOrganizationMember(
	ctx context.Context,
	realm, orgID, memberID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmOrganizationsOrgIdMembersMemberIdWithResponse(
		ctx, realm, orgID, memberID,
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

func (c *organizationsClient) InviteExistingOrganizationMember(
	ctx context.Context,
	realm, orgID, userID string,
) (*Response, error) {
	body := InviteExistingOrganizationMemberRequest{
		Id: &userID,
	}

	res, err := c.client.PostAdminRealmsRealmOrganizationsOrgIdMembersInviteExistingUserWithFormdataBodyWithResponse(
		ctx, realm, orgID, body,
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

func (c *organizationsClient) InviteNewOrganizationMember(
	ctx context.Context,
	realm, orgID, email, firstName, lastName string,
) (*Response, error) {
	body := InviteNewOrganizationMemberRequest{
		Email:     &email,
		FirstName: &firstName,
		LastName:  &lastName,
	}

	res, err := c.client.PostAdminRealmsRealmOrganizationsOrgIdMembersInviteUserWithFormdataBodyWithResponse(
		ctx, realm, orgID, body,
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
