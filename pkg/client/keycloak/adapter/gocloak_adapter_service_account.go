package adapter

import (
	"context"

	"github.com/pkg/errors"
)

func (a GoCloakAdapter) SyncServiceAccountRoles(realm, clientID string, realmRoles []string,
	clientRoles map[string][]string) error {
	user, err := a.client.GetClientServiceAccount(context.Background(), a.token.AccessToken, realm, clientID)
	if err != nil {
		return errors.Wrap(err, "unable to get client service account")
	}

	roleMappings, err := a.client.GetRoleMappingByUserID(context.Background(), a.token.AccessToken, realm, *user.ID)
	if err != nil {
		return errors.Wrap(err, "error during GetRoleMappingByUserID")
	}

	if err := a.syncEntityRealmRoles(*user.ID, realm, realmRoles, roleMappings.RealmMappings,
		a.client.AddRealmRoleToUser, a.client.DeleteRealmRoleFromUser); err != nil {
		return errors.Wrap(err, "unable to sync service account realm roles")
	}

	if err := a.syncEntityClientRoles(realm, *user.ID, clientRoles, roleMappings.ClientMappings,
		a.client.AddClientRoleToUser, a.client.DeleteClientRoleFromUser); err != nil {
		return errors.Wrap(err, "unable to sync service account client roles")
	}

	return nil
}
