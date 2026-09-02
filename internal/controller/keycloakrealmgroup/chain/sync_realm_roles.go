package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type SyncRealmRoles struct{}

func NewSyncRealmRoles() *SyncRealmRoles {
	return &SyncRealmRoles{}
}

func (h *SyncRealmRoles) Serve(
	ctx context.Context,
	group *keycloakApi.KeycloakRealmGroup,
	kClient *keycloakapi.KeycloakClient,
	groupCtx *GroupContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing group realm roles")

	// nil: not managed, leave Keycloak alone. Empty non-nil: managed, clear every mapping.
	// KeycloakRealmGroup has no reconciliationStrategy; this is the only opt-out.
	if group.Spec.RealmRoles == nil {
		log.Info("Realm roles are not managed by the resource, skipping")

		return nil
	}

	desiredRoleNames := *group.Spec.RealmRoles

	realm := groupCtx.RealmName
	groupID := groupCtx.GroupID

	currentRoles, _, err := kClient.Groups.GetRealmRoleMappings(ctx, realm, groupID)
	if err != nil {
		return fmt.Errorf("unable to get realm role mappings for group %s: %w", groupID, err)
	}

	currentMap := make(map[string]keycloakapi.RoleRepresentation, len(currentRoles))

	for i, r := range currentRoles {
		if r.Name != nil {
			currentMap[*r.Name] = currentRoles[i]
		}
	}

	claimedSet := make(map[string]struct{}, len(desiredRoleNames))
	for _, r := range desiredRoleNames {
		claimedSet[r] = struct{}{}
	}

	var rolesToAdd []keycloakapi.RoleRepresentation

	for _, claimedName := range desiredRoleNames {
		if _, exists := currentMap[claimedName]; !exists {
			role, _, err := kClient.Roles.GetRealmRole(ctx, realm, claimedName)
			if err != nil {
				return fmt.Errorf("unable to get realm role %q: %w", claimedName, err)
			}

			mappable, err := keycloakapi.RequireRealmRoleWithID(role, realm, claimedName)
			if err != nil {
				return err
			}

			rolesToAdd = append(rolesToAdd, mappable)
		}
	}

	if len(rolesToAdd) > 0 {
		if _, err := kClient.Groups.AddRealmRoleMappings(ctx, realm, groupID, rolesToAdd); err != nil {
			return fmt.Errorf("unable to add realm role mappings: %w", err)
		}
	}

	var rolesToRemove []keycloakapi.RoleRepresentation

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
