package adapter

import (
	"context"

	"github.com/Nerzal/gocloak/v10"
	"github.com/pkg/errors"
)

func (a GoCloakAdapter) syncEntityRealmRoles(entityID string, realm string, claimedRealmRoles []string,
	currentRealmRoles *[]gocloak.Role,
	addRoleFunc func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error,
	delRoleFunc func(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error) error {

	currentRealmRoleMap, claimedRoleMap := make(map[string]gocloak.Role), make(map[string]struct{})
	if currentRealmRoles != nil {
		for i, currentRole := range *currentRealmRoles {
			currentRealmRoleMap[*currentRole.Name] = (*currentRealmRoles)[i]
		}
	}
	for _, r := range claimedRealmRoles {
		claimedRoleMap[r] = struct{}{}
	}

	realmRolesToAdd := make([]gocloak.Role, 0, len(claimedRealmRoles))
	for _, r := range claimedRealmRoles {
		if _, ok := currentRealmRoleMap[r]; !ok {
			role, err := a.client.GetRealmRole(context.Background(), a.token.AccessToken, realm, r)
			if err != nil {
				return errors.Wrapf(err, "unable to get realm role, realm: %s, role: %s", realm, r)
			}
			realmRolesToAdd = append(realmRolesToAdd, *role)
		}
	}
	if len(realmRolesToAdd) > 0 {
		if err := addRoleFunc(context.Background(), a.token.AccessToken, realm, entityID,
			realmRolesToAdd); err != nil {
			return errors.Wrapf(err, "unable to add realm roles to entity, realm: %s, entity id: %s, roles: %v",
				realm, entityID, realmRolesToAdd)
		}
	}
	realmRolesToDelete := make([]gocloak.Role, 0, len(currentRealmRoleMap))
	for currentRoleName, role := range currentRealmRoleMap {
		if _, ok := claimedRoleMap[currentRoleName]; !ok {
			realmRolesToDelete = append(realmRolesToDelete, role)
		}
	}
	if len(realmRolesToDelete) > 0 {
		if err := delRoleFunc(context.Background(), a.token.AccessToken, realm, entityID,
			realmRolesToDelete); err != nil {
			return errors.Wrapf(err, "unable to delete realm roles from group, realm: %s, entity id: %s, roles: %v",
				realm, entityID, realmRolesToDelete)
		}
	}
	return nil
}

func (a GoCloakAdapter) syncOneEntityClientRole(realm, entityID, clientID string, claimedRoles []string,
	currentRoles map[string]*gocloak.ClientMappingsRepresentation,
	addRoleFunc func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error,
	delRoleFunc func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error) error {

	CID, err := a.GetClientID(clientID, realm)
	if err != nil {
		return errors.Wrapf(err, "unable to get client id, realm: %s, clientID %s", realm, clientID)
	}

	currentClientRoles, claimedClientRoles := make(map[string]*gocloak.Role), make(map[string]struct{})
	if client, ok := currentRoles[clientID]; ok && client != nil && client.Mappings != nil {
		for k, role := range *client.Mappings {
			currentClientRoles[*role.Name] = &(*client.Mappings)[k]
		}
	}
	for _, claimedRole := range claimedRoles {
		claimedClientRoles[claimedRole] = struct{}{}
	}

	rolesToAdd := make([]gocloak.Role, 0, len(claimedRoles))
	for k := range claimedClientRoles {
		if _, ok := currentClientRoles[k]; !ok {
			role, err := a.client.GetClientRole(context.Background(), a.token.AccessToken, realm, CID, k)
			if err != nil {
				return errors.Wrapf(err, "unable to get client role, realm: %s, clientID: %s, role: %s", realm,
					CID, k)
			}
			rolesToAdd = append(rolesToAdd, *role)
		}
	}
	if len(rolesToAdd) > 0 {
		if err := addRoleFunc(context.Background(), a.token.AccessToken, realm, CID, entityID,
			rolesToAdd); err != nil {
			return errors.Wrapf(err, "unable to add realm role to entity, realm: %s, clientID: %s, entityID: %s",
				realm, CID, entityID)
		}
	}

	rolesToDelete := make([]gocloak.Role, 0, len(currentClientRoles))
	for k, v := range currentClientRoles {
		if _, ok := claimedClientRoles[k]; !ok {
			rolesToDelete = append(rolesToDelete, *v)
		}
	}
	if len(rolesToDelete) > 0 {
		if err := delRoleFunc(context.Background(), a.token.AccessToken, realm, CID, entityID,
			rolesToDelete); err != nil {
			return errors.Wrapf(err, "unable to del client role from entity, realm: %s, clientID: %s, entityID: %s",
				realm, CID, entityID)
		}
	}

	return nil
}

func (a GoCloakAdapter) syncEntityClientRoles(realm, entityID string, claimedRoles map[string][]string,
	currentRoles map[string]*gocloak.ClientMappingsRepresentation,
	addRoleFunc func(ctx context.Context, token, realm, clientID, entityID string, roles []gocloak.Role) error,
	delRoleFunc func(ctx context.Context, token, realm, clientID, groupID string, roles []gocloak.Role) error) error {

	for clientID, roles := range claimedRoles {
		if err := a.syncOneEntityClientRole(realm, entityID, clientID, roles, currentRoles, addRoleFunc,
			delRoleFunc); err != nil {
			return errors.Wrap(err, "error during syncOneEntityClientRole")
		}
	}

	for clientName, client := range currentRoles {
		if _, ok := claimedRoles[clientName]; !ok && client.Mappings != nil {
			rolesToDelete := make([]gocloak.Role, 0, len(currentRoles))
			for _, role := range *client.Mappings {
				rolesToDelete = append(rolesToDelete, role)
			}

			if len(rolesToDelete) > 0 {
				if err := delRoleFunc(context.Background(), a.token.AccessToken, realm,
					*client.ID, entityID, rolesToDelete); err != nil {
					return errors.Wrap(err, "unable to delete client role from user")
				}
			}
		}
	}

	return nil
}
