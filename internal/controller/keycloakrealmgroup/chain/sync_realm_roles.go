package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type SyncRealmRoles struct{}

func NewSyncRealmRoles() *SyncRealmRoles {
	return &SyncRealmRoles{}
}

func (h *SyncRealmRoles) Serve(
	ctx context.Context,
	group *keycloakApi.KeycloakRealmGroup,
	kClient *keycloakv2.KeycloakClient,
	groupCtx *GroupContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing group realm roles")

	realm := groupCtx.RealmName
	groupID := groupCtx.GroupID

	currentRoles, _, err := kClient.Groups.GetRealmRoleMappings(ctx, realm, groupID)
	if err != nil {
		return fmt.Errorf("unable to get realm role mappings for group %s: %w", groupID, err)
	}

	currentMap := make(map[string]keycloakv2.RoleRepresentation, len(currentRoles))

	for i, r := range currentRoles {
		if r.Name != nil {
			currentMap[*r.Name] = currentRoles[i]
		}
	}

	claimedSet := make(map[string]struct{}, len(group.Spec.RealmRoles))
	for _, r := range group.Spec.RealmRoles {
		claimedSet[r] = struct{}{}
	}

	var rolesToAdd []keycloakv2.RoleRepresentation

	for _, claimedName := range group.Spec.RealmRoles {
		if _, exists := currentMap[claimedName]; !exists {
			role, _, err := kClient.Roles.GetRealmRole(ctx, realm, claimedName)
			if err != nil {
				return fmt.Errorf("unable to get realm role %q: %w", claimedName, err)
			}

			if role == nil {
				return fmt.Errorf("realm role %q not found in realm %q", claimedName, realm)
			}

			rolesToAdd = append(rolesToAdd, *role)
		}
	}

	if len(rolesToAdd) > 0 {
		if _, err := kClient.Groups.AddRealmRoleMappings(ctx, realm, groupID, rolesToAdd); err != nil {
			return fmt.Errorf("unable to add realm role mappings: %w", err)
		}
	}

	var rolesToRemove []keycloakv2.RoleRepresentation

	for name, role := range currentMap {
		if _, claimed := claimedSet[name]; !claimed {
			rolesToRemove = append(rolesToRemove, role)
		}
	}

	if len(rolesToRemove) > 0 {
		if _, err := kClient.Groups.DeleteRealmRoleMappings(ctx, realm, groupID, rolesToRemove); err != nil {
			return fmt.Errorf("unable to delete realm role mappings: %w", err)
		}
	}

	log.Info("Group realm roles synced successfully")

	return nil
}
