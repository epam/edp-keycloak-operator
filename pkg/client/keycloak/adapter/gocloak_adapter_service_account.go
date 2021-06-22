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

func (a GoCloakAdapter) SetServiceAccountAttributes(realm, clientID string, attributes map[string]string) error {
	user, err := a.client.GetClientServiceAccount(context.Background(), a.token.AccessToken, realm, clientID)
	if err != nil {
		return errors.Wrap(err, "unable to get client service account")
	}

	svcAttributes := make(map[string][]string)
	for k, v := range attributes {
		svcAttributes[k] = []string{v}
	}

	user.Attributes = &svcAttributes

	if err := a.client.UpdateUser(context.Background(), a.token.AccessToken, realm, *user); err != nil {
		return errors.Wrapf(err, "unable to update service account user: %s", clientID)
	}

	return nil
}
