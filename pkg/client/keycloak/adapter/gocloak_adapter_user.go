package adapter

import (
	"context"

	"github.com/Nerzal/gocloak/v10"
	"github.com/pkg/errors"
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
}

type UserRealmRoleMapping struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserGroupMapping struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (a GoCloakAdapter) SyncRealmUser(ctx context.Context, realmName string, user *KeycloakUser, addOnly bool) error {
	users, err := a.client.GetUsers(ctx, a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: gocloak.StringP(user.Username),
	})
	if err != nil {
		return errors.Wrap(err, "unable to list users")
	}

	var keycloakUser gocloak.User
	for _, usr := range users {
		if *usr.Username == user.Username {
			keycloakUser = *usr
			break
		}
	}

	if keycloakUser.ID == nil {
		keycloakUser = gocloak.User{
			Username:        &user.Username,
			Enabled:         &user.Enabled,
			EmailVerified:   &user.EmailVerified,
			FirstName:       &user.FirstName,
			LastName:        &user.LastName,
			RequiredActions: &user.RequiredUserActions,
			Email:           &user.Email,
		}
	}

	if err := a.setUserParams(ctx, realmName, &keycloakUser, user, addOnly); err != nil {
		return errors.Wrap(err, "unable to set user params")
	}

	if err := a.syncUserRoles(ctx, realmName, *keycloakUser.ID, user, addOnly); err != nil {
		return errors.Wrap(err, "unable to sync user roles")
	}

	if err := a.syncUserGroups(ctx, realmName, *keycloakUser.ID, user, addOnly); err != nil {
		return errors.Wrap(err, "unable to sync user group")
	}

	return nil
}

func (a GoCloakAdapter) syncUserGroups(ctx context.Context, realmName string, userID string, user *KeycloakUser, addOnly bool) error {
	if !addOnly {
		if err := a.clearUserGroups(ctx, realmName, userID); err != nil {
			return errors.Wrap(err, "unable to clear user groups")
		}
	}

	groups, err := a.client.GetGroups(ctx, a.token.AccessToken, realmName, gocloak.GetGroupsParams{})
	if err != nil {
		return errors.Wrap(err, "unable to get realm groups")
	}

	groupDict := make(map[string]string)
	for _, gr := range groups {
		groupDict[*gr.Name] = *gr.ID
	}

	for _, gr := range user.Groups {
		groupID, ok := groupDict[gr]
		if !ok {
			return errors.Errorf("group %s not found", gr)
		}

		if err := a.AddUserToGroup(ctx, realmName, userID, groupID); err != nil {
			return errors.Wrap(err, "unable to add user to group")
		}
	}

	return nil
}

func (a GoCloakAdapter) syncUserRoles(ctx context.Context, realmName string, userID string, user *KeycloakUser, addOnly bool) error {
	if !addOnly {
		if err := a.clearUserRealmRoles(ctx, realmName, userID); err != nil {
			return errors.Wrap(err, "unable to clear realm roles")
		}
	}

	for _, roleName := range user.Roles {
		if err := a.AddRealmRoleToUser(ctx, realmName, user.Username, roleName); err != nil {
			return errors.Wrap(err, "unable to add realm role to user")
		}
	}

	return nil
}

func (a GoCloakAdapter) GetUserRealmRoleMappings(ctx context.Context, realmName string, userID string) ([]UserRealmRoleMapping, error) {
	var roles []UserRealmRoleMapping

	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realmName,
		"id":    userID,
	}).SetResult(&roles).Get(a.basePath + getUserRealmRoleMappings)

	if err := a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "unable to get realm role mappings")
	}

	return roles, nil
}

func (a GoCloakAdapter) GetUserGroupMappings(ctx context.Context, realmName string, userID string) ([]UserGroupMapping, error) {
	var groups []UserGroupMapping

	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm": realmName,
		"id":    userID,
	}).SetResult(&groups).Get(a.basePath + getUserGroupMappings)

	if err := a.checkError(err, rsp); err != nil {
		return nil, errors.Wrap(err, "unable to get group mappings")
	}

	return groups, nil
}

func (a GoCloakAdapter) RemoveUserFromGroup(ctx context.Context, realmName, userID, groupID string) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm":   realmName,
		"userID":  userID,
		"groupID": groupID,
	}).Delete(a.basePath + manageUserGroups)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to remove user from group")
	}

	return nil
}

func (a GoCloakAdapter) AddUserToGroup(ctx context.Context, realmName, userID, groupID string) error {
	rsp, err := a.startRestyRequest().SetContext(ctx).SetPathParams(map[string]string{
		"realm":   realmName,
		"userID":  userID,
		"groupID": groupID,
	}).SetBody(map[string]string{
		"groupId": groupID,
		"realm":   realmName,
		"userId":  userID,
	}).Put(a.basePath + manageUserGroups)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to add user to group")
	}

	return nil
}

func (a GoCloakAdapter) clearUserGroups(ctx context.Context, realmName, userID string) error {
	groups, err := a.GetUserGroupMappings(ctx, realmName, userID)
	if err != nil {
		return errors.Wrap(err, "unable to get user groups")
	}

	for _, gr := range groups {
		if err := a.RemoveUserFromGroup(ctx, realmName, userID, gr.ID); err != nil {
			return errors.Wrap(err, "unable to remove user from group")
		}
	}

	return nil
}

func (a GoCloakAdapter) clearUserRealmRoles(ctx context.Context, realmName string, userID string) error {
	roles, err := a.GetUserRealmRoleMappings(ctx, realmName, userID)
	if err != nil {
		return errors.Wrap(err, "unable to get user realm role map")
	}

	goRoles := make([]gocloak.Role, 0, len(roles))
	for _, r := range roles {
		goRoles = append(goRoles, gocloak.Role{ID: &r.ID, Name: &r.Name})
	}

	if err := a.client.DeleteRealmRoleFromUser(ctx, a.token.AccessToken, realmName, userID, goRoles); err != nil {
		return errors.Wrap(err, "unable to delete realm role from user")
	}

	return nil
}

func (a GoCloakAdapter) setUserParams(ctx context.Context, realmName string, keycloakUser *gocloak.User,
	userCR *KeycloakUser, addOnly bool) error {
	if userCR.Attributes != nil && len(userCR.Attributes) > 0 {
		attrs := make(map[string][]string)
		for k, v := range userCR.Attributes {
			attrs[k] = []string{v}
		}

		if addOnly && keycloakUser.Attributes != nil && len(*keycloakUser.Attributes) > 0 {
			for k, v := range *keycloakUser.Attributes {
				attrs[k] = v
			}
		}

		keycloakUser.Attributes = &attrs
	}

	if keycloakUser.ID != nil {
		if err := a.client.UpdateUser(ctx, a.token.AccessToken, realmName, *keycloakUser); err != nil {
			return errors.Wrap(err, "unable to update user")
		}

		return nil
	}

	userID, err := a.client.CreateUser(ctx, a.token.AccessToken, realmName, *keycloakUser)
	if err != nil {
		return errors.Wrap(err, "unable to create user")
	}

	if userCR.Password != "" {
		if err := a.setUserPassword(realmName, userID, userCR.Password); err != nil {
			return errors.Wrapf(err, "unable to set user password, user id: %s", userID)
		}
	}
	keycloakUser.ID = &userID

	return nil
}

func (a GoCloakAdapter) setUserPassword(realmName, userID, password string) error {
	rsp, err := a.startRestyRequest().SetPathParams(map[string]string{
		"realm": realmName,
		"id":    userID,
	}).SetBody(map[string]interface{}{
		"temporary": true,
		"type":      "password",
		"value":     password,
	}).Put(a.basePath + setRealmUserPassword)

	if err := a.checkError(err, rsp); err != nil {
		return errors.Wrap(err, "unable to set user password")
	}

	return nil
}
