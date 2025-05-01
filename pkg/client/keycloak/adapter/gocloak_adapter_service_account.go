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

	roleMappings, err := a.client.GetRoleMappingByUserID(context.Background(), a.token.AccessToken, realm, *user.ID)
	if err != nil {
		return errors.Wrap(err, "error during GetRoleMappingByUserID")
	}

	deleteRealmRoleFunc := a.client.DeleteRealmRoleFromUser
	if addOnly {
		deleteRealmRoleFunc = doNotDeleteRealmRoleFromUser
	}

	if err := a.syncEntityRealmRoles(*user.ID, realm, realmRoles, roleMappings.RealmMappings,
		a.client.AddRealmRoleToUser, deleteRealmRoleFunc); err != nil {
		return errors.Wrap(err, "unable to sync service account realm roles")
	}

	deleteClientRoleFromUserFunc := a.client.DeleteClientRoleFromUser
	if addOnly {
		deleteClientRoleFromUserFunc = doNotDeleteClientRoleFromUser
	}

	if err := a.syncEntityClientRoles(realm, *user.ID, clientRoles, roleMappings.ClientMappings,
		a.client.AddClientRoleToUser, deleteClientRoleFromUserFunc); err != nil {
		return errors.Wrap(err, "unable to sync service account client roles")
	}

	return nil
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

func (a GoCloakAdapter) SetServiceAccountGroups(realm, clientID string, groups []string, addOnly bool) error {
	user, err := a.client.GetClientServiceAccount(context.Background(), a.token.AccessToken, realm, clientID)
	if err != nil {
		return errors.Wrap(err, "unable to get client service account")
	}

	svcGroups := make(map[string]struct{})
	if addOnly && user.Groups != nil {
		for _, group := range *user.Groups {
			svcGroups[group] = struct{}{}
		}
	}

	for _, group := range groups {
		svcGroups[group] = struct{}{}
	}

	var newGroups []string
	for group := range svcGroups {
		newGroups = append(newGroups, group)
	}

	user.Groups = &newGroups

	if err = a.client.UpdateUser(context.Background(), a.token.AccessToken, realm, *user); err != nil {
		return errors.Wrapf(err, "unable to update service account user: %s", clientID)
	}

	return nil
}
