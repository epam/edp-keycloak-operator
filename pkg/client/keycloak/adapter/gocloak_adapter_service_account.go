package adapter

import (
	"context"
	"fmt"
	"slices"

	"github.com/Nerzal/gocloak/v12"
)

func (a GoCloakAdapter) SyncServiceAccountRoles(realm, clientID string, realmRoles []string,
	clientRoles map[string][]string, addOnly bool) error {
	user, err := a.client.GetClientServiceAccount(context.Background(), a.token.AccessToken, realm, clientID)
	if err != nil {
		return fmt.Errorf("unable to get client service account: %w", err)
	}

	return a.SyncUserRoles(context.Background(), realm, *user.ID, realmRoles, clientRoles, addOnly)
}

func (a GoCloakAdapter) SyncServiceAccountGroups(realm, clientID string, groups []string, addOnly bool) error {
	user, err := a.client.GetClientServiceAccount(context.Background(), a.token.AccessToken, realm, clientID)
	if err != nil {
		return fmt.Errorf("unable to get client service account: %w", err)
	}

	return a.syncUserGroups(context.Background(), realm, *user.ID, groups, addOnly)
}

func doNotDeleteRealmRoleFromUser(ctx context.Context, token, realm, entityID string, roles []gocloak.Role) error {
	return nil
}

func doNotDeleteClientRoleFromUser(
	ctx context.Context,
	token, realm, clientID, groupID string,
	roles []gocloak.Role,
) error {
	return nil
}

func (a GoCloakAdapter) SetServiceAccountAttributes(
	realm, clientID string,
	attributes map[string][]string,
	addOnly bool,
) error {
	user, err := a.client.GetClientServiceAccount(context.Background(), a.token.AccessToken, realm, clientID)
	if err != nil {
		return fmt.Errorf("unable to get client service account: %w", err)
	}

	if user.Attributes == nil {
		user.Attributes = &map[string][]string{}
	}

	for k, v := range attributes {
		if addOnly {
			existingValues := (*user.Attributes)[k]

			for _, newValue := range v {
				if !slices.Contains(existingValues, newValue) {
					existingValues = append(existingValues, newValue)
				}
			}

			(*user.Attributes)[k] = existingValues
		} else {
			(*user.Attributes)[k] = v
		}
	}

	// If not addOnly, remove attributes that are not in the desired list
	if !addOnly {
		for existingKey := range *user.Attributes {
			if _, exists := attributes[existingKey]; !exists {
				delete(*user.Attributes, existingKey)
			}
		}
	}

	if err = a.client.UpdateUser(context.Background(), a.token.AccessToken, realm, *user); err != nil {
		return fmt.Errorf("unable to update service account user: %s: %w", clientID, err)
	}

	return nil
}
