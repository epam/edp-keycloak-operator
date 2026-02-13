package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type SyncClientRoles struct{}

func NewSyncClientRoles() *SyncClientRoles {
	return &SyncClientRoles{}
}

func (h *SyncClientRoles) Serve(
	ctx context.Context,
	group *keycloakApi.KeycloakRealmGroup,
	kClient *keycloakv2.KeycloakClient,
	groupCtx *GroupContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing group client roles")

	realm := groupCtx.RealmName
	groupID := groupCtx.GroupID

	roleMappings, _, err := kClient.Groups.GetRoleMappings(ctx, realm, groupID)
	if err != nil {
		return fmt.Errorf("unable to get role mappings for group %s: %w", groupID, err)
	}

	claimedClientRoles := make(map[string][]string, len(group.Spec.ClientRoles))
	for _, cr := range group.Spec.ClientRoles {
		claimedClientRoles[cr.ClientID] = cr.Roles
	}

	// Extract client mappings to avoid redundant API calls in syncOneClientRoles.
	var clientMappings map[string]keycloakv2.ClientMappingsRepresentation
	if roleMappings != nil && roleMappings.ClientMappings != nil {
		clientMappings = *roleMappings.ClientMappings
	}

	for clientIDStr, claimedRoleNames := range claimedClientRoles {
		if err := h.syncOneClientRoles(ctx, kClient, realm, groupID, clientIDStr, claimedRoleNames, clientMappings); err != nil {
			return fmt.Errorf("unable to sync client roles for client %q: %w", clientIDStr, err)
		}
	}

	// Remove roles for clients no longer in spec.
	if roleMappings != nil && roleMappings.ClientMappings != nil {
		for clientName, clientMapping := range *roleMappings.ClientMappings {
			if _, claimed := claimedClientRoles[clientName]; claimed {
				continue
			}

			if clientMapping.Mappings == nil || len(*clientMapping.Mappings) == 0 || clientMapping.Id == nil {
				continue
			}

			if _, err := kClient.Groups.DeleteClientRoleMappings(
				ctx, realm, groupID, *clientMapping.Id, *clientMapping.Mappings,
			); err != nil {
				return fmt.Errorf("unable to remove unclaimed client roles for %q: %w", clientName, err)
			}
		}
	}

	log.Info("Group client roles synced successfully")

	return nil
}

func (h *SyncClientRoles) syncOneClientRoles(
	ctx context.Context,
	kClient *keycloakv2.KeycloakClient,
	realm, groupID, clientIDStr string,
	claimedRoleNames []string,
	clientMappings map[string]keycloakv2.ClientMappingsRepresentation,
) error {
	// Try to get client UUID and current roles from the cached clientMappings.
	var clientUUID string

	var currentRoles []keycloakv2.RoleRepresentation

	if clientMapping, found := clientMappings[clientIDStr]; found {
		// Use cached data from GetRoleMappings call.
		if clientMapping.Id != nil {
			clientUUID = *clientMapping.Id
		}

		if clientMapping.Mappings != nil {
			currentRoles = *clientMapping.Mappings
		}
	} else {
		// Client not in mappings yet, resolve UUID.
		var err error

		clientUUID, err = resolveClientUUID(ctx, kClient, realm, clientIDStr)
		if err != nil {
			return err
		}
		// currentRoles remains empty (no roles currently assigned).
	}

	currentMap := make(map[string]keycloakv2.RoleRepresentation, len(currentRoles))

	for i, r := range currentRoles {
		if r.Name != nil {
			currentMap[*r.Name] = currentRoles[i]
		}
	}

	claimedSet := make(map[string]struct{}, len(claimedRoleNames))
	for _, rn := range claimedRoleNames {
		claimedSet[rn] = struct{}{}
	}

	var rolesToAdd []keycloakv2.RoleRepresentation

	for _, rn := range claimedRoleNames {
		if _, exists := currentMap[rn]; !exists {
			role, _, err := kClient.Clients.GetClientRole(ctx, realm, clientUUID, rn)
			if err != nil {
				return fmt.Errorf("unable to get client role %q: %w", rn, err)
			}

			if role == nil {
				return fmt.Errorf("client role %q not found for client %q in realm %q", rn, clientIDStr, realm)
			}

			rolesToAdd = append(rolesToAdd, *role)
		}
	}

	if len(rolesToAdd) > 0 {
		if _, err := kClient.Groups.AddClientRoleMappings(ctx, realm, groupID, clientUUID, rolesToAdd); err != nil {
			return fmt.Errorf("unable to add client role mappings: %w", err)
		}
	}

	var rolesToRemove []keycloakv2.RoleRepresentation

	for name, role := range currentMap {
		if _, claimed := claimedSet[name]; !claimed {
			rolesToRemove = append(rolesToRemove, role)
		}
	}

	if len(rolesToRemove) > 0 {
		if _, err := kClient.Groups.DeleteClientRoleMappings(ctx, realm, groupID, clientUUID, rolesToRemove); err != nil {
			return fmt.Errorf("unable to delete client role mappings: %w", err)
		}
	}

	return nil
}

// resolveClientUUID resolves a Keycloak client_id string to its internal UUID.
func resolveClientUUID(
	ctx context.Context,
	kClient *keycloakv2.KeycloakClient,
	realm, clientIDStr string,
) (string, error) {
	clients, _, err := kClient.Clients.GetClients(ctx, realm, &keycloakv2.GetClientsParams{
		ClientId: &clientIDStr,
	})
	if err != nil {
		return "", fmt.Errorf("unable to get clients: %w", err)
	}

	for _, c := range clients {
		if c.ClientId != nil && *c.ClientId == clientIDStr && c.Id != nil {
			return *c.Id, nil
		}
	}

	return "", fmt.Errorf("client %q not found in realm %q", clientIDStr, realm)
}
