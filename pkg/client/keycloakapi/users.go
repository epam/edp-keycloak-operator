package keycloakapi

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi/generated"
)

type (
	UserProfileConfig               = generated.UPConfig
	UserProfileAttribute            = generated.UPAttribute
	UserProfileAttributePermissions = generated.UPAttributePermissions
	UserProfileAttributeRequired    = generated.UPAttributeRequired
	UserProfileAttributeSelector    = generated.UPAttributeSelector
	UserProfileGroup                = generated.UPGroup
	UnmanagedAttributePolicy        = generated.UnmanagedAttributePolicy
	UserRepresentation              = generated.UserRepresentation
	CredentialRepresentation        = generated.CredentialRepresentation
	FederatedIdentityRepresentation = generated.FederatedIdentityRepresentation
	UserSessionRepresentation       = generated.UserSessionRepresentation
	GetUsersParams                  = generated.GetAdminRealmsRealmUsersParams
)

type usersClient struct {
	client generated.ClientWithResponsesInterface
}

// Ensure usersClient implements UsersClient
var _ UsersClient = (*usersClient)(nil)

func (c *usersClient) GetUsersProfile(ctx context.Context, realm string) (*UserProfileConfig, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersProfileWithResponse(ctx, realm)
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

func (c *usersClient) UpdateUsersProfile(
	ctx context.Context,
	realm string,
	userProfile UserProfileConfig,
) (*UserProfileConfig, *Response, error) {
	res, err := c.client.PutAdminRealmsRealmUsersProfileWithResponse(ctx, realm, userProfile)
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

func (c *usersClient) FindUserByUsername(
	ctx context.Context,
	realm, username string,
) (*UserRepresentation, *Response, error) {
	exact := true

	res, err := c.client.GetAdminRealmsRealmUsersWithResponse(ctx, realm, &generated.GetAdminRealmsRealmUsersParams{
		Username: &username,
		Exact:    &exact,
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

	if res.JSON200 == nil || len(*res.JSON200) == 0 {
		return nil, response, ErrNotFound
	}

	user := (*res.JSON200)[0]

	return &user, response, nil
}

func (c *usersClient) CreateUser(ctx context.Context, realm string, user UserRepresentation) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmUsersWithResponse(ctx, realm, user)
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

func (c *usersClient) GetUserRealmRoleMappings(
	ctx context.Context,
	realm, userID string,
) ([]RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersUserIdRoleMappingsRealmWithResponse(ctx, realm, userID)
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

func (c *usersClient) AddUserRealmRoles(
	ctx context.Context,
	realm, userID string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmUsersUserIdRoleMappingsRealmWithResponse(ctx, realm, userID, roles)
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

func (c *usersClient) GetUserGroups(
	ctx context.Context, realm, userID string,
) ([]GroupRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersUserIdGroupsWithResponse(ctx, realm, userID, nil)
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

func (c *usersClient) AddUserToGroup(ctx context.Context, realm, userID, groupID string) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmUsersUserIdGroupsGroupIdWithResponse(ctx, realm, userID, groupID)
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

func (c *usersClient) RemoveUserFromGroup(ctx context.Context, realm, userID, groupID string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmUsersUserIdGroupsGroupIdWithResponse(ctx, realm, userID, groupID)
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

func (c *usersClient) UpdateUser(
	ctx context.Context,
	realm, userID string,
	user UserRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmUsersUserIdWithResponse(ctx, realm, userID, user)
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

func (c *usersClient) DeleteUser(ctx context.Context, realm, userID string) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmUsersUserIdWithResponse(ctx, realm, userID)
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

func (c *usersClient) SetUserPassword(
	ctx context.Context,
	realm, userID string,
	cred CredentialRepresentation,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmUsersUserIdResetPasswordWithResponse(ctx, realm, userID, cred)
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

func (c *usersClient) DeleteUserRealmRoles(
	ctx context.Context,
	realm, userID string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmUsersUserIdRoleMappingsRealmWithResponse(ctx, realm, userID, roles)
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

func (c *usersClient) GetUserClientRoleMappings(
	ctx context.Context,
	realm, userID, clientID string,
) ([]RoleRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersUserIdRoleMappingsClientsClientIdWithResponse(
		ctx, realm, userID, clientID)
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

func (c *usersClient) AddUserClientRoles(
	ctx context.Context,
	realm, userID, clientID string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmUsersUserIdRoleMappingsClientsClientIdWithResponse(
		ctx, realm, userID, clientID, roles)
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

func (c *usersClient) DeleteUserClientRoles(
	ctx context.Context,
	realm, userID, clientID string,
	roles []RoleRepresentation,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmUsersUserIdRoleMappingsClientsClientIdWithResponse(
		ctx, realm, userID, clientID, roles)
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

func (c *usersClient) GetUserFederatedIdentities(
	ctx context.Context,
	realm, userID string,
) ([]FederatedIdentityRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersUserIdFederatedIdentityWithResponse(ctx, realm, userID)
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

func (c *usersClient) CreateUserFederatedIdentity(
	ctx context.Context,
	realm, userID, provider string,
	identity FederatedIdentityRepresentation,
) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmUsersUserIdFederatedIdentityProviderWithResponse(
		ctx, realm, userID, provider, identity)
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

func (c *usersClient) DeleteUserFederatedIdentity(
	ctx context.Context,
	realm, userID, provider string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmUsersUserIdFederatedIdentityProviderWithResponse(
		ctx, realm, userID, provider)
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

func (c *usersClient) GetUsers(
	ctx context.Context,
	realm string,
	params *GetUsersParams,
) ([]UserRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersWithResponse(ctx, realm, params)
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

func (c *usersClient) GetUser(
	ctx context.Context,
	realm, userID string,
) (*UserRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersUserIdWithResponse(ctx, realm, userID, nil)
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

func (c *usersClient) GetUserSessions(
	ctx context.Context,
	realm, userID string,
) ([]UserSessionRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersUserIdSessionsWithResponse(ctx, realm, userID)
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

func (c *usersClient) LogoutUser(ctx context.Context, realm, userID string) (*Response, error) {
	res, err := c.client.PostAdminRealmsRealmUsersUserIdLogoutWithResponse(ctx, realm, userID)
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

func (c *usersClient) GetUserCredentials(
	ctx context.Context,
	realm, userID string,
) ([]CredentialRepresentation, *Response, error) {
	res, err := c.client.GetAdminRealmsRealmUsersUserIdCredentialsWithResponse(ctx, realm, userID)
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

func (c *usersClient) DeleteUserCredential(
	ctx context.Context,
	realm, userID, credentialID string,
) (*Response, error) {
	res, err := c.client.DeleteAdminRealmsRealmUsersUserIdCredentialsCredentialIdWithResponse(
		ctx, realm, userID, credentialID)
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

func (c *usersClient) ExecuteActionsEmail(
	ctx context.Context,
	realm, userID string,
	actions []string,
) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmUsersUserIdExecuteActionsEmailWithResponse(
		ctx, realm, userID, nil, actions)
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

func (c *usersClient) SendVerifyEmail(ctx context.Context, realm, userID string) (*Response, error) {
	res, err := c.client.PutAdminRealmsRealmUsersUserIdSendVerifyEmailWithResponse(ctx, realm, userID, nil)
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

func (c *usersClient) ImpersonateUser(
	ctx context.Context,
	realm, userID string,
) (map[string]any, *Response, error) {
	res, err := c.client.PostAdminRealmsRealmUsersUserIdImpersonationWithResponse(ctx, realm, userID)
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
