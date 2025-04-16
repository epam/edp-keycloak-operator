package adapter

import (
	"context"
	"fmt"
	"slices"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
	keycloak_go_client "github.com/zmotso/keycloak-go-client"
)

type KeycloakUser struct {
	Username            string
	Enabled             bool
	EmailVerified       bool
	Email               string
	FirstName           string
	LastName            string
	RequiredUserActions []string
	Roles               []string
	Groups              []string
	Attributes          map[string]string
	Password            string
	IdentityProviders   *[]string
}

type UserRealmRoleMapping struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserGroupMapping struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (a GoCloakAdapter) SyncRealmUser(ctx context.Context, realmName string, userDto *KeycloakUser, addOnly bool) error {
	userID, err := a.createOrUpdateUser(ctx, realmName, userDto, addOnly)
	if err != nil {
		return err
	}

	if userDto.Password != "" {
		if err = a.setUserPassword(realmName, userID, userDto.Password); err != nil {
			return err
		}
	}

	if err = a.syncUserRoles(ctx, realmName, userID, userDto.Roles, addOnly); err != nil {
		return err
	}

	if err = a.syncUserGroups(ctx, realmName, userID, userDto.Groups, addOnly); err != nil {
		return err
	}

	if userDto.IdentityProviders != nil {
		return a.syncUserIdentityProviders(ctx, realmName, userID, userDto.Username, *userDto.IdentityProviders)
	}

	return nil
}

func (a GoCloakAdapter) createOrUpdateUser(ctx context.Context, realmName string, userDto *KeycloakUser, addOnly bool) (string, error) {
	user, err := a.GetUserByName(ctx, realmName, userDto.Username)
	if err != nil {
		if !IsErrNotFound(err) {
			return "", fmt.Errorf("unable to get user: %w", err)
		}

		kcUser := gocloak.User{
			Username:        &userDto.Username,
			Enabled:         &userDto.Enabled,
			EmailVerified:   &userDto.EmailVerified,
			FirstName:       &userDto.FirstName,
			LastName:        &userDto.LastName,
			RequiredActions: &userDto.RequiredUserActions,
			Email:           &userDto.Email,
		}

		if len(userDto.Attributes) > 0 {
			kcUser.Attributes = a.makeUserAttributes(&kcUser, userDto, addOnly)
		}

		var userID string

		userID, err = a.client.CreateUser(ctx, a.token.AccessToken, realmName, kcUser)
		if err != nil {
			return "", fmt.Errorf("unable to create user: %w", err)
		}

		return userID, nil
	}

	user.Username = &userDto.Username
	user.Enabled = &userDto.Enabled
	user.EmailVerified = &userDto.EmailVerified
	user.FirstName = &userDto.FirstName
	user.LastName = &userDto.LastName
	user.RequiredActions = &userDto.RequiredUserActions
	user.Email = &userDto.Email

	if len(userDto.Attributes) > 0 {
		user.Attributes = a.makeUserAttributes(user, userDto, addOnly)
	}

	if err = a.client.UpdateUser(ctx, a.token.AccessToken, realmName, *user); err != nil {
		return "", fmt.Errorf("unable to update user: %w", err)
	}

	return *user.ID, nil
}

func (a GoCloakAdapter) GetUserByName(ctx context.Context, realmName, username string) (*gocloak.User, error) {
	params := gocloak.GetUsersParams{
		Username: &username,
		Exact:    gocloak.BoolP(true),
	}

	users, err := a.client.GetUsers(ctx, a.token.AccessToken, realmName, params)
	if err != nil {
		return nil, fmt.Errorf("unable to get users: %w", err)
	}

	for _, user := range users {
		if user.Username != nil && *user.Username == username {
			return user, nil
		}
	}

	return nil, NotFoundError("user not found")
}

func (a GoCloakAdapter) syncUserGroups(ctx context.Context, realmName string, userID string, groups []string, addOnly bool) error {
	userGroups, err := a.GetUserGroupMappings(ctx, realmName, userID)
	if err != nil {
		return err
	}

	groupsToAdd := make([]string, 0, len(groups))

	for _, gn := range groups {
		if !slices.ContainsFunc(userGroups, func(mapping UserGroupMapping) bool {
			return mapping.Name == gn
		}) {
			groupsToAdd = append(groupsToAdd, gn)
		}
	}

	if len(groupsToAdd) > 0 {
		var kcGroups map[string]gocloak.Group

		kcGroups, err = a.getGroupsByNames(
			ctx,
			realmName,
			groupsToAdd,
		)
		if err != nil {
			return fmt.Errorf("unable to get groups: %w", err)
		}

		for _, gr := range kcGroups {
			if err = a.AddUserToGroup(ctx, realmName, userID, *gr.ID); err != nil {
				return fmt.Errorf("failed to add user to group %v: %w", gr.Name, err)
			}
		}
	}

	if !addOnly {
		for _, gr := range userGroups {
			if !slices.Contains(groups, gr.Name) {
				if err = a.RemoveUserFromGroup(ctx, realmName, userID, gr.ID); err != nil {
					return fmt.Errorf("unable to remove user from group: %w", err)
				}
			}
		}
	}

	return nil
}

func (a GoCloakAdapter) syncUserRoles(ctx context.Context, realmName string, userID string, roles []string, addOnly bool) error {
	if !addOnly {
		if err := a.clearUserRealmRoles(ctx, realmName, userID); err != nil {
			return errors.Wrap(err, "unable to clear realm roles")
		}
	}

	realmRoles, err := a.client.GetRealmRoles(ctx, a.token.AccessToken, realmName, gocloak.GetRoleParams{
		Max:                 gocloak.IntP(100),
		BriefRepresentation: gocloak.BoolP(true),
	})
	if err != nil {
		return fmt.Errorf("unable to get realm roles: %w", err)
	}

	realmRolesDict := make(map[string]gocloak.Role, len(realmRoles))
	for _, role := range realmRoles {
		realmRolesDict[*role.Name] = *role
	}

	kcRoles := make([]gocloak.Role, 0, len(roles))

	for _, roleName := range roles {
		role, ok := realmRolesDict[roleName]
		if !ok {
			return errors.Errorf("realm role %s not found", roleName)
		}

		kcRoles = append(kcRoles, role)
	}

	if err = a.client.AddRealmRoleToUser(ctx, a.token.AccessToken, realmName, userID, kcRoles); err != nil {
		return fmt.Errorf("unable to add realm roles to user: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) GetUserRealmRoleMappings(ctx context.Context, realmName string, userID string) ([]UserRealmRoleMapping, error) {
	var roles []UserRealmRoleMapping

	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    userID,
		}).
		SetResult(&roles).
		Get(a.buildPath(getUserRealmRoleMappings))

	if err = a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "unable to get realm role mappings")
	}

	return roles, nil
}

func (a GoCloakAdapter) GetUserGroupMappings(ctx context.Context, realmName string, userID string) ([]UserGroupMapping, error) {
	var groups []UserGroupMapping

	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    userID,
		}).
		SetResult(&groups).
		Get(a.buildPath(getUserGroupMappings))

	if err = a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "unable to get group mappings")
	}

	return groups, nil
}

func (a GoCloakAdapter) RemoveUserFromGroup(ctx context.Context, realmName, userID, groupID string) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			"userID":              userID,
			"groupID":             groupID,
		}).
		Delete(a.buildPath(manageUserGroups))

	if err = a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to remove user from group")
	}

	return nil
}

func (a GoCloakAdapter) AddUserToGroup(ctx context.Context, realmName, userID, groupID string) error {
	rsp, err := a.startRestyRequest().
		SetContext(ctx).
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			"userID":              userID,
			"groupID":             groupID,
		}).
		SetBody(map[string]string{
			"groupId":             groupID,
			keycloakApiParamRealm: realmName,
			"userId":              userID,
		}).
		Put(a.buildPath(manageUserGroups))

	if err = a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to add user to group")
	}

	return nil
}

func (a GoCloakAdapter) UpdateUsersProfile(
	ctx context.Context,
	realm string,
	userProfile keycloak_go_client.UserProfileConfig,
) (*keycloak_go_client.UserProfileConfig, error) {
	httpClient := a.client.RestyClient().GetClient()

	cl, err := keycloak_go_client.NewClient(a.basePath, keycloak_go_client.WithToken(a.token.AccessToken), keycloak_go_client.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create keycloak_go_client client: %w", err)
	}

	profile, res, err := cl.Users.UpdateUsersProfile(ctx, realm, userProfile)
	if err = checkHttpResp(res, err); err != nil {
		return nil, err
	}

	return profile, nil
}

func (a GoCloakAdapter) GetUsersProfile(
	ctx context.Context,
	realm string,
) (*keycloak_go_client.UserProfileConfig, error) {
	httpClient := a.client.RestyClient().GetClient()

	cl, err := keycloak_go_client.NewClient(a.basePath, keycloak_go_client.WithToken(a.token.AccessToken), keycloak_go_client.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create keycloak_go_client client: %w", err)
	}

	profile, res, err := cl.Users.GetUsersProfile(ctx, realm)
	if err = checkHttpResp(res, err); err != nil {
		return nil, err
	}

	return profile, nil
}

func checkHttpResp(res *keycloak_go_client.Response, err error) error {
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	if res == nil || res.HTTPResponse == nil {
		return errors.New("empty response")
	}

	const maxStatusCodesSuccess = 399

	if res.HTTPResponse.StatusCode > maxStatusCodesSuccess {
		return errors.Errorf("status: %s, body: %s", res.HTTPResponse.Status, res.Body)
	}

	return nil
}

func (a GoCloakAdapter) clearUserRealmRoles(ctx context.Context, realmName string, userID string) error {
	roles, err := a.GetUserRealmRoleMappings(ctx, realmName, userID)
	if err != nil {
		return errors.Wrap(err, "unable to get user realm role map")
	}

	goRoles := make([]gocloak.Role, 0, len(roles))
	for i := range roles {
		goRoles = append(goRoles, gocloak.Role{ID: &roles[i].ID, Name: &roles[i].Name})
	}

	if err := a.client.DeleteRealmRoleFromUser(ctx, a.token.AccessToken, realmName, userID, goRoles); err != nil {
		return errors.Wrap(err, "unable to delete realm role from user")
	}

	return nil
}

func (a GoCloakAdapter) setUserPassword(realmName, userID, password string) error {
	rsp, err := a.startRestyRequest().
		SetPathParams(map[string]string{
			keycloakApiParamRealm: realmName,
			keycloakApiParamId:    userID,
		}).
		SetBody(map[string]interface{}{
			"temporary": false,
			"type":      "password",
			"value":     password,
		}).
		Put(a.buildPath(setRealmUserPassword))

	if err = a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to set user password")
	}

	return nil
}

func (a GoCloakAdapter) makeUserAttributes(keycloakUser *gocloak.User, userCR *KeycloakUser, addOnly bool) *map[string][]string {
	attrs := make(map[string][]string)
	for k, v := range userCR.Attributes {
		attrs[k] = []string{v}
	}

	if addOnly && keycloakUser.Attributes != nil && len(*keycloakUser.Attributes) > 0 {
		for k, v := range *keycloakUser.Attributes {
			attrs[k] = v
		}
	}

	return &attrs
}

func (a GoCloakAdapter) syncUserIdentityProviders(
	ctx context.Context,
	realmName,
	userID,
	userName string,
	providers []string,
) error {
	existingProviders, err := a.getExistingIdentityProviders(ctx, realmName, userID)
	if err != nil {
		return err
	}

	if err := a.addMissingIdentityProviders(ctx, realmName, userID, userName, providers, existingProviders); err != nil {
		return err
	}

	return a.removeExtraIdentityProviders(ctx, realmName, userID, providers, existingProviders)
}

func (a GoCloakAdapter) getExistingIdentityProviders(
	ctx context.Context,
	realmName, userID string,
) (map[string]struct{}, error) {
	existingIdentities, err := a.client.GetUserFederatedIdentities(ctx, a.token.AccessToken, realmName, userID)
	if err != nil {
		return nil, fmt.Errorf("unable to get user identity providers: %w", err)
	}

	existingProviders := make(map[string]struct{}, len(existingIdentities))

	for _, identity := range existingIdentities {
		if identity.IdentityProvider != nil {
			existingProviders[*identity.IdentityProvider] = struct{}{}
		}
	}

	return existingProviders, nil
}

func (a GoCloakAdapter) addMissingIdentityProviders(
	ctx context.Context,
	realmName, userID, userName string,
	providers []string,
	existingProviders map[string]struct{},
) error {
	for _, provider := range providers {
		if _, exists := existingProviders[provider]; exists {
			continue
		}

		federatedIdentity := gocloak.FederatedIdentityRepresentation{
			IdentityProvider: &provider,
			UserID:           &userID,
			UserName:         &userName,
		}

		if err := a.client.CreateUserFederatedIdentity(ctx, a.token.AccessToken, realmName, userID, provider, federatedIdentity); err != nil {
			return fmt.Errorf("unable to add user to identity provider %s: %w", provider, err)
		}
	}

	return nil
}

func (a GoCloakAdapter) removeExtraIdentityProviders(
	ctx context.Context,
	realmName, userID string,
	providers []string,
	existingProviders map[string]struct{},
) error {
	for existingProvider := range existingProviders {
		if !slices.Contains(providers, existingProvider) {
			if err := a.client.DeleteUserFederatedIdentity(ctx, a.token.AccessToken, realmName, userID, existingProvider); err != nil {
				return fmt.Errorf("unable to remove user from identity provider %s: %w", existingProvider, err)
			}
		}
	}

	return nil
}
