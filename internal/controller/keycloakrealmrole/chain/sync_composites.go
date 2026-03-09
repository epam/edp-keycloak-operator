package chain

import (
	"context"
	"fmt"
	"maps"
	"slices"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type SyncComposites struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewSyncComposites(kClientV2 *keycloakv2.KeycloakClient) *SyncComposites {
	return &SyncComposites{kClientV2: kClientV2}
}

func (h *SyncComposites) Serve(
	ctx context.Context,
	role *keycloakApi.KeycloakRealmRole,
	realmName string,
	_ *RoleContext,
) error {
	if !role.Spec.Composite {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing composite roles")

	spec := role.Spec
	rolesClient := h.kClientV2.Roles
	clientsClient := h.kClientV2.Clients

	currentComposites, _, err := rolesClient.GetRealmRoleComposites(ctx, realmName, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to get current composites: %w", err)
	}

	stale := make(map[string]keycloakv2.RoleRepresentation, len(currentComposites))

	for _, cr := range currentComposites {
		key := *cr.Name
		if cr.ClientRole != nil && *cr.ClientRole && cr.ContainerId != nil {
			key = fmt.Sprintf("%s-%s", *cr.ContainerId, *cr.Name)
		}

		stale[key] = cr
	}

	rolesToAdd := make([]keycloakv2.RoleRepresentation, 0, len(spec.Composites))

	for _, composite := range spec.Composites {
		name := composite.Name
		if _, ok := stale[name]; ok {
			delete(stale, name)
			continue
		}

		realmRole, _, err := rolesClient.GetRealmRole(ctx, realmName, name)
		if err != nil {
			return fmt.Errorf("failed to get realm role %s: %w", name, err)
		}

		rolesToAdd = append(rolesToAdd, *realmRole)
	}

	for clientName, composites := range spec.CompositesClientRoles {
		clients, _, err := clientsClient.GetClients(ctx, realmName, &keycloakv2.GetClientsParams{
			ClientId: &clientName,
		})
		if err != nil {
			return fmt.Errorf("failed to get client %s: %w", clientName, err)
		}

		if len(clients) == 0 || clients[0].Id == nil {
			return fmt.Errorf("client %s not found", clientName)
		}

		clientUUID := *clients[0].Id

		for _, composite := range composites {
			roleName := composite.Name
			mapKey := fmt.Sprintf("%s-%s", clientUUID, roleName)

			if _, ok := stale[mapKey]; ok {
				delete(stale, mapKey)
				continue
			}

			clientRole, _, err := clientsClient.GetClientRole(ctx, realmName, clientUUID, roleName)
			if err != nil {
				return fmt.Errorf("failed to get client role %s: %w", roleName, err)
			}

			rolesToAdd = append(rolesToAdd, *clientRole)
		}
	}

	if len(rolesToAdd) > 0 {
		if _, err := rolesClient.AddRealmRoleComposites(ctx, realmName, spec.Name, rolesToAdd); err != nil {
			return fmt.Errorf("failed to add composite roles: %w", err)
		}
	}

	if len(stale) > 0 {
		staleRoles := slices.Collect(maps.Values(stale))
		if _, err := rolesClient.DeleteRealmRoleComposites(ctx, realmName, spec.Name, staleRoles); err != nil {
			return fmt.Errorf("failed to delete stale composite roles: %w", err)
		}
	}

	log.Info("Composite roles synced successfully")

	return nil
}
