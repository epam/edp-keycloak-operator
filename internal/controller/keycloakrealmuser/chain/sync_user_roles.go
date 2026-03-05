package chain

import (
	"context"
	"fmt"
	"slices"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type SyncUserRoles struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewSyncUserRoles(kClientV2 *keycloakv2.KeycloakClient) *SyncUserRoles {
	return &SyncUserRoles{kClientV2: kClientV2}
}

func (h *SyncUserRoles) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	realmName string,
	userCtx *UserContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing user roles")

	addOnly := user.IsReconciliationStrategyAddOnly()

	if err := h.syncRealmRoles(ctx, realmName, userCtx.UserID, user.Spec.Roles, addOnly); err != nil {
		return fmt.Errorf("unable to sync user realm roles: %w", err)
	}

	if err := h.syncClientRoles(ctx, realmName, userCtx.UserID, user.Spec.ClientRoles, addOnly); err != nil {
		return fmt.Errorf("unable to sync user client roles: %w", err)
	}

	log.Info("User roles synced successfully")

	return nil
}

func (h *SyncUserRoles) syncRealmRoles(
	ctx context.Context,
	realmName, userID string,
	desiredRoleNames []string,
	addOnly bool,
) error {
	currentRoles, _, err := h.kClientV2.Users.GetUserRealmRoleMappings(ctx, realmName, userID)
	if err != nil {
		return fmt.Errorf("unable to get user realm role mappings: %w", err)
	}

	currentByName := make(map[string]keycloakv2.RoleRepresentation, len(currentRoles))

	for _, r := range currentRoles {
		if r.Name != nil {
			currentByName[*r.Name] = r
		}
	}

	// Add missing roles
	toAdd := make([]keycloakv2.RoleRepresentation, 0, len(desiredRoleNames))

	for _, roleName := range desiredRoleNames {
		if _, exists := currentByName[roleName]; exists {
			continue
		}

		role, _, err := h.kClientV2.Roles.GetRealmRole(ctx, realmName, roleName)
		if err != nil {
			return fmt.Errorf("unable to get realm role %q: %w", roleName, err)
		}

		toAdd = append(toAdd, *role)
	}

	if len(toAdd) > 0 {
		if _, err := h.kClientV2.Users.AddUserRealmRoles(ctx, realmName, userID, toAdd); err != nil {
			return fmt.Errorf("unable to add realm roles to user: %w", err)
		}
	}

	if addOnly {
		return nil
	}

	// Remove extra roles
	var toRemove []keycloakv2.RoleRepresentation

	for _, r := range currentRoles {
		if r.Name != nil && !slices.Contains(desiredRoleNames, *r.Name) {
			toRemove = append(toRemove, r)
		}
	}

	if len(toRemove) > 0 {
		if _, err := h.kClientV2.Users.DeleteUserRealmRoles(ctx, realmName, userID, toRemove); err != nil {
			return fmt.Errorf("unable to delete realm roles from user: %w", err)
		}
	}

	return nil
}

func (h *SyncUserRoles) syncClientRoles(
	ctx context.Context,
	realmName, userID string,
	clientRoles []keycloakApi.UserClientRole,
	addOnly bool,
) error {
	for _, cr := range clientRoles {
		clients, _, err := h.kClientV2.Clients.GetClients(ctx, realmName, &keycloakv2.GetClientsParams{ClientId: &cr.ClientID})
		if err != nil {
			return fmt.Errorf("unable to get client %q: %w", cr.ClientID, err)
		}

		if len(clients) == 0 || clients[0].Id == nil {
			return fmt.Errorf("client %q not found", cr.ClientID)
		}

		clientUUID := *clients[0].Id

		currentRoles, _, err := h.kClientV2.Users.GetUserClientRoleMappings(ctx, realmName, userID, clientUUID)
		if err != nil {
			return fmt.Errorf("unable to get client role mappings for client %q: %w", cr.ClientID, err)
		}

		currentByName := make(map[string]keycloakv2.RoleRepresentation, len(currentRoles))

		for _, r := range currentRoles {
			if r.Name != nil {
				currentByName[*r.Name] = r
			}
		}

		// Add missing roles
		var toAdd []keycloakv2.RoleRepresentation

		for _, roleName := range cr.Roles {
			if _, exists := currentByName[roleName]; exists {
				continue
			}

			role, _, err := h.kClientV2.Clients.GetClientRole(ctx, realmName, clientUUID, roleName)
			if err != nil {
				return fmt.Errorf("unable to get client role %q for client %q: %w", roleName, cr.ClientID, err)
			}

			toAdd = append(toAdd, *role)
		}

		if len(toAdd) > 0 {
			if _, err := h.kClientV2.Users.AddUserClientRoles(ctx, realmName, userID, clientUUID, toAdd); err != nil {
				return fmt.Errorf("unable to add client roles to user for client %q: %w", cr.ClientID, err)
			}
		}

		if addOnly {
			continue
		}

		// Remove extra roles
		var toRemove []keycloakv2.RoleRepresentation

		for _, r := range currentRoles {
			if r.Name != nil && !slices.Contains(cr.Roles, *r.Name) {
				toRemove = append(toRemove, r)
			}
		}

		if len(toRemove) > 0 {
			if _, err := h.kClientV2.Users.DeleteUserClientRoles(ctx, realmName, userID, clientUUID, toRemove); err != nil {
				return fmt.Errorf("unable to delete client roles from user for client %q: %w", cr.ClientID, err)
			}
		}
	}

	return nil
}
