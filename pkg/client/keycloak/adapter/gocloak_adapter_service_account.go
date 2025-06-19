package adapter

import (
	"context"

	"github.com/Nerzal/gocloak/v12"
	"github.com/pkg/errors"
)

func (a GoCloakAdapter) SyncServiceAccountRoles(realm, clientID string, realmRoles []string,
	clientRoles map[string][]string, addOnly bool) error {
	user, err := a.client.GetClientServiceAccount(context.Background(), a.token.AccessToken, realm, clientID)
	if err != nil {
		return errors.Wrap(err, "unable to get client service account")
	}

	return a.SyncUserRoles(context.Background(), realm, *user.ID, realmRoles, clientRoles, addOnly)
}

func (a GoCloakAdapter) SyncServiceAccountGroups(realm, clientID string, groups []string, addOnly bool) error {
	user, err := a.client.GetClientServiceAccount(context.Background(), a.token.AccessToken, realm, clientID)
	if err != nil {
		return errors.Wrap(err, "unable to get client service account")
	}

	return a.syncUserGroups(context.Background(), realm, *user.ID, groups, addOnly)
}

func doNotDeleteRealmRoleFromUser(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
	return nil
}

func doNotDeleteClientRoleFromUser(ctx context.Context, token, realm, clientID, groupID string, roles []gocloak.Role) error {
	return nil
}

func (a GoCloakAdapter) SetServiceAccountAttributes(realm, clientID string, attributes map[string]string,
	addOnly bool) error {
	user, err := a.client.GetClientServiceAccount(context.Background(), a.token.AccessToken, realm, clientID)
	if err != nil {
		return errors.Wrap(err, "unable to get client service account")
	}

	svcAttributes := make(map[string][]string)
	if addOnly && user.Attributes != nil {
		svcAttributes = *user.Attributes
	}

	for k, v := range attributes {
		svcAttributes[k] = []string{v}
	}

	user.Attributes = &svcAttributes

	if err := a.client.UpdateUser(context.Background(), a.token.AccessToken, realm, *user); err != nil {
		return errors.Wrapf(err, "unable to update service account user: %s", clientID)
	}

	return nil
}
