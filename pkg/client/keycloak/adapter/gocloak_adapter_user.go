package adapter

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/Nerzal/gocloak/v12"
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
	ClientRoles         map[string][]string
	Groups              []string
	Attributes          map[string][]string
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

func (a GoCloakAdapter) SyncRealmUser(
	ctx context.Context,
	realmName string,
	userDto *KeycloakUser,
	addOnly bool,
) error {
	userID, err := a.createOrUpdateUser(ctx, realmName, userDto, addOnly)
	if err != nil {
		return err
	}

	if userDto.Password != "" {
		if err = a.setUserPassword(realmName, userID, userDto.Password); err != nil {
			return err
		}
	}

	if err = a.SyncUserRoles(ctx, realmName, userID, userDto.Roles, userDto.ClientRoles, addOnly); err != nil {
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

func (a GoCloakAdapter) createOrUpdateUser(
	ctx context.Context,
	realmName string,
	userDto *KeycloakUser,
	addOnly bool,
) (string, error) {
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

func (a GoCloakAdapter) syncUserGroups(
	ctx context.Context,
	realmName string,
	userID string,
	groups []string,
	addOnly bool,
) error {
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

// SyncUserRoles syncs user realm and client roles.
func (a GoCloakAdapter) SyncUserRoles(
	ctx context.Context,
	realm, userID string,
	realmRoles []string,
	clientRoles map[string][]string,
	addOnly bool,
) error {
	roleMappings, err := a.client.GetRoleMappingByUserID(ctx, a.token.AccessToken, realm, userID)
	if err != nil {
		return fmt.Errorf("error during GetRoleMappingByUserID: %w", err)
	}

	deleteRealmRoleFunc := a.client.DeleteRealmRoleFromUser
	if addOnly {
		deleteRealmRoleFunc = doNotDeleteRealmRoleFromUser
	}

	if err := a.syncEntityRealmRoles(
		userID,
		realm,
		realmRoles,
		roleMappings.RealmMappings,
		a.client.AddRealmRoleToUser,
		deleteRealmRoleFunc,
	); err != nil {
		return fmt.Errorf("unable to sync user realm roles: %w", err)
	}

	deleteClientRoleFromUserFunc := a.client.DeleteClientRoleFromUser
	if addOnly {
		deleteClientRoleFromUserFunc = doNotDeleteClientRoleFromUser
	}

	if err := a.syncEntityClientRoles(
		realm,
		userID,
		clientRoles,
		roleMappings.ClientMappings,
		a.client.AddClientRoleToUser,
		deleteClientRoleFromUserFunc,
	); err != nil {
		return fmt.Errorf("unable to sync user client roles: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) GetUserRealmRoleMappings(
	ctx context.Context,
	realmName string,
	userID string,
) ([]UserRealmRoleMapping, error) {
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
		return nil, fmt.Errorf("unable to get realm role mappings: %w", err)
	}

	return roles, nil
}

func (a GoCloakAdapter) GetUserGroupMappings(
	ctx context.Context,
	realmName string,
	userID string,
) ([]UserGroupMapping, error) {
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
		return nil, fmt.Errorf("unable to get group mappings: %w", err)
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
		return fmt.Errorf("unable to remove user from group: %w", err)
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
		return fmt.Errorf("unable to add user to group: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) UpdateUsersProfile(
	ctx context.Context,
	realm string,
	userProfile keycloak_go_client.UserProfileConfig,
) (*keycloak_go_client.UserProfileConfig, error) {
	httpClient := a.client.RestyClient().GetClient()

	cl, err := keycloak_go_client.NewClient(
		a.buildPath(""),
		keycloak_go_client.WithToken(a.token.AccessToken),
		keycloak_go_client.WithHTTPClient(httpClient),
	)
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

	cl, err := keycloak_go_client.NewClient(
		a.buildPath(""),
		keycloak_go_client.WithToken(a.token.AccessToken),
		keycloak_go_client.WithHTTPClient(httpClient),
	)
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
		return fmt.Errorf("status: %s, body: %s", res.HTTPResponse.Status, res.Body)
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
		return fmt.Errorf("unable to set user password: %w", err)
	}

	return nil
}

func (a GoCloakAdapter) makeUserAttributes(
	keycloakUser *gocloak.User,
	userCR *KeycloakUser,
	addOnly bool,
) *map[string][]string {
	if keycloakUser.Attributes == nil {
		keycloakUser.Attributes = &map[string][]string{}
	}

	attributes := make(map[string][]string)
	for k, v := range *keycloakUser.Attributes {
		attributes[k] = v
	}

	for k, v := range userCR.Attributes {
		if addOnly {
			existingValues := attributes[k]

			for _, newValue := range v {
				if !slices.Contains(existingValues, newValue) {
					existingValues = append(existingValues, newValue)
				}
			}

			attributes[k] = existingValues
		} else {
			attributes[k] = v
		}
	}

	// If not addOnly, remove attributes that are not in the desired list
	if !addOnly {
		for existingKey := range attributes {
			if _, exists := userCR.Attributes[existingKey]; !exists {
				delete(attributes, existingKey)
			}
		}
	}

	return &attributes
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

		exists, err := a.IdentityProviderExists(ctx, realmName, provider)
		if err != nil {
			return fmt.Errorf("unable to check if identity provider exists: %w", err)
		}

		if !exists {
			return fmt.Errorf("identity provider %s does not exist", provider)
		}

		federatedIdentity := gocloak.FederatedIdentityRepresentation{
			IdentityProvider: &provider,
			UserID:           &userID,
			UserName:         &userName,
		}

		if err := a.client.CreateUserFederatedIdentity(
			ctx,
			a.token.AccessToken,
			realmName,
			userID,
			provider,
			federatedIdentity,
		); err != nil {
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
			if err := a.client.DeleteUserFederatedIdentity(
				ctx,
				a.token.AccessToken,
				realmName,
				userID,
				existingProvider,
			); err != nil {
				return fmt.Errorf("unable to remove user from identity provider %s: %w", existingProvider, err)
			}
		}
	}

	return nil
}
