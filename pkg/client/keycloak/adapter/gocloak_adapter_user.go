package adapter

import (
	"context"

	"github.com/Nerzal/gocloak/v8"
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
}

func (a GoCloakAdapter) SyncRealmUser(realmName string, user *KeycloakUser) error {
	users, err := a.client.GetUsers(context.Background(), a.token.AccessToken, realmName, gocloak.GetUsersParams{
		Username: gocloak.StringP(user.Username),
	})

	if err != nil {
		return errors.Wrap(err, "unable to list users")
	}

	for _, usr := range users {
		if *usr.Username == user.Username {
			return ErrDuplicated("user already exists")
		}
	}

	usr := gocloak.User{
		Username:        &user.Username,
		Enabled:         &user.Enabled,
		EmailVerified:   &user.EmailVerified,
		FirstName:       &user.FirstName,
		LastName:        &user.LastName,
		RequiredActions: &user.RequiredUserActions,
		RealmRoles:      &user.Roles,
		Groups:          &user.Groups,
		Email:           &user.Email,
	}

	if user.Attributes != nil && len(user.Attributes) > 0 {
		attrs := make(map[string][]string)
		for k, v := range user.Attributes {
			attrs[k] = []string{v}
		}
		usr.Attributes = &attrs
	}

	if _, err := a.client.CreateUser(context.Background(), a.token.AccessToken, realmName, usr); err != nil {
		return errors.Wrap(err, "unable to create user")
	}

	for _, realmRole := range user.Roles {
		if err := a.AddRealmRoleToUser(realmName, user.Username, realmRole); err != nil {
			return errors.Wrap(err, "unable to add realm role to user")
		}
	}

	return nil
}
