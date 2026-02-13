package keycloakv2

import (
	"context"

	"k8s.io/utils/ptr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2/generated"
)

type (
	GroupRepresentation          = generated.GroupRepresentation
	MappingsRepresentation       = generated.MappingsRepresentation
	ClientMappingsRepresentation = generated.ClientMappingsRepresentation
	GetGroupsParams              = generated.GetAdminRealmsRealmGroupsParams
	GetChildGroupsParams         = generated.GetAdminRealmsRealmGroupsGroupIdChildrenParams
)

type groupsClient struct {
	client generated.ClientWithResponsesInterface
}

var _ GroupsClient = (*groupsClient)(nil)

func (c *groupsClient) GetGroups(
	ctx context.Context,
	realm string,
	params *GetGroupsParams,
) ([]GroupRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmGroupsWithResponse(ctx, realm, params)
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

func (c *groupsClient) GetGroup(ctx context.Context, realm, groupID string) (*GroupRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmGroupsGroupIdWithResponse(ctx, realm, groupID)
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

func (c *groupsClient) CreateGroup(ctx context.Context, realm string, group GroupRepresentation) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmGroupsWithResponse(ctx, realm, group)
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

func (c *groupsClient) UpdateGroup(
	ctx context.Context,
	realm string,
	groupID string,
	group GroupRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmGroupsGroupIdWithResponse(ctx, realm, groupID, group)
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

func (c *groupsClient) DeleteGroup(ctx context.Context, realm, groupID string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmGroupsGroupIdWithResponse(ctx, realm, groupID)
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

func (c *groupsClient) GetChildGroups(
	ctx context.Context,
	realm string,
	groupID string,
	params *GetChildGroupsParams,
) ([]GroupRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmGroupsGroupIdChildrenWithResponse(ctx, realm, groupID, params)
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

func (c *groupsClient) CreateChildGroup(
	ctx context.Context,
	realm string,
	parentGroupID string,
	group GroupRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmGroupsGroupIdChildrenWithResponse(ctx, realm, parentGroupID, group)
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

func (c *groupsClient) FindGroupByName(
	ctx context.Context,
	realm string,
	groupName string,
) (*GroupRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmGroupsWithResponse(ctx, realm, &GetGroupsParams{
		Search: &groupName,
		Exact:  ptr.To(true),
	})
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

	return findGroupInList(*res.JSON200, groupName), response, nil
}

func (c *groupsClient) FindChildGroupByName(
	ctx context.Context,
	realm string,
	parentGroupID string,
	groupName string,
) (*GroupRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmGroupsGroupIdChildrenWithResponse(
		ctx,
		realm,
		parentGroupID,
		&GetChildGroupsParams{
			Search: &groupName,
			Exact:  ptr.To(true),
		},
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

	return findGroupInList(*res.JSON200, groupName), response, nil
}

func (c *groupsClient) GetRoleMappings(
	ctx context.Context,
	realm string,
	groupID string,
) (*MappingsRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmGroupsGroupIdRoleMappingsWithResponse(ctx, realm, groupID)
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

func (c *groupsClient) GetRealmRoleMappings(
	ctx context.Context,
	realm string,
	groupID string,
) ([]RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmGroupsGroupIdRoleMappingsRealmWithResponse(ctx, realm, groupID)
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

func (c *groupsClient) AddRealmRoleMappings(
	ctx context.Context,
	realm string,
	groupID string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmGroupsGroupIdRoleMappingsRealmWithResponse(ctx, realm, groupID, roles)
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

func (c *groupsClient) DeleteRealmRoleMappings(
	ctx context.Context,
	realm string,
	groupID string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmGroupsGroupIdRoleMappingsRealmWithResponse(ctx, realm, groupID, roles)
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

func (c *groupsClient) GetClientRoleMappings(
	ctx context.Context,
	realm string,
	groupID string,
	clientID string,
) ([]RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmGroupsGroupIdRoleMappingsClientsClientIdWithResponse(
		ctx, realm, groupID, clientID,
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

func (c *groupsClient) AddClientRoleMappings(
	ctx context.Context,
	realm string,
	groupID string,
	clientID string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmGroupsGroupIdRoleMappingsClientsClientIdWithResponse(
		ctx, realm, groupID, clientID, roles,
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

func (c *groupsClient) DeleteClientRoleMappings(
	ctx context.Context,
	realm string,
	groupID string,
	clientID string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmGroupsGroupIdRoleMappingsClientsClientIdWithResponse(
		ctx, realm, groupID, clientID, roles,
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

// findGroupInList searches for a group by name in a list of groups.
func findGroupInList(groups []GroupRepresentation, name string) *GroupRepresentation {
	for i := range groups {
		if groups[i].Name != nil && *groups[i].Name == name {
			return &groups[i]
		}
	}

	return nil
}
