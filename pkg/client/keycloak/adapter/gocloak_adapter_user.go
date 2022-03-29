package adapter

import (
	"context"
	"fmt"

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

	for _, roleName := range user.Roles {
		if err := a.AddRealmRoleToUser(ctx, realmName, user.Username, roleName); err != nil {
			return errors.Wrap(err, "unable to add realm role to user")
		}
	}

	return nil
}

func (a GoCloakAdapter) setUserParams(ctx context.Context, realmName string, keycloakUser *gocloak.User,
	userCR *KeycloakUser, addOnly bool) error {

	if len(userCR.Groups) > 0 {
		if addOnly && keycloakUser.Groups != nil && len(*keycloakUser.Groups) > 0 {
			userCR.Groups = append(userCR.Groups, *keycloakUser.Groups...)
		}

		for _, gr := range userCR.Groups {
			if _, err := a.getGroup(realmName, gr); err != nil {
				return ErrNotFound(fmt.Sprintf("group [%s] not found", gr))
			}
		}

		keycloakUser.Groups = &userCR.Groups
	}

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
