package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

// SyncSubGroups handles the deprecated SubGroups field for backward compatibility.
// Deprecated: Use ParentGroup approach instead.
type SyncSubGroups struct{}

func NewSyncSubGroups() *SyncSubGroups {
	return &SyncSubGroups{}
}

func (h *SyncSubGroups) Serve(
	ctx context.Context,
	group *keycloakApi.KeycloakRealmGroup,
	kClient *keycloakv2.KeycloakClient,
	groupCtx *GroupContext,
) error {
	if len(group.Spec.SubGroups) == 0 {
		return nil
	}

	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing deprecated SubGroups field")

	realm := groupCtx.RealmName
	groupID := groupCtx.GroupID

	currentChildren, _, err := kClient.Groups.GetChildGroups(ctx, realm, groupID, nil)
	if err != nil {
		return fmt.Errorf("unable to get child groups: %w", err)
	}

	currentMap := make(map[string]keycloakv2.GroupRepresentation, len(currentChildren))

	for i, c := range currentChildren {
		if c.Name != nil {
			currentMap[*c.Name] = currentChildren[i]
		}
	}

	claimedSet := make(map[string]struct{}, len(group.Spec.SubGroups))
	for _, sg := range group.Spec.SubGroups {
		claimedSet[sg] = struct{}{}
	}

	for _, claimed := range group.Spec.SubGroups {
		if _, exists := currentMap[claimed]; !exists {
			subGroup, _, err := kClient.Groups.FindGroupByName(ctx, realm, claimed)
			if err != nil {
				return fmt.Errorf("unable to find subgroup %q: %w", claimed, err)
			}

			if subGroup == nil {
				return fmt.Errorf("subgroup %q not found in realm %q", claimed, realm)
			}

			if _, err := kClient.Groups.CreateChildGroup(ctx, realm, groupID, *subGroup); err != nil {
				return fmt.Errorf("unable to add subgroup %q: %w", claimed, err)
			}
		}
	}

	// Detach subgroups that are no longer claimed.
	// Creating a group at the top level detaches it from its parent.
	for name, child := range currentMap {
		if _, claimed := claimedSet[name]; !claimed {
			if _, err := kClient.Groups.CreateGroup(ctx, realm, child); err != nil {
				return fmt.Errorf("unable to detach subgroup %q: %w", name, err)
			}
		}
	}

	log.Info("SubGroups synced successfully")

	return nil
}
