package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
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
	kClient *keycloakapi.KeycloakClient,
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

	currentMap := make(map[string]keycloakapi.GroupRepresentation, len(currentChildren))

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
				if keycloakapi.IsNotFound(err) {
					// The name search only matches top-level groups: a group nested under
					// another parent is reported as not found and cannot be claimed here.
					return fmt.Errorf(
						"subgroup %q not found among top-level groups of realm %q; "+
							"if it is nested under another parent group, use spec.parentGroup on that group's resource instead of the deprecated spec.subGroups field",
						claimed, realm,
					)
				}

				return fmt.Errorf("unable to find subgroup %q: %w", claimed, err)
			}

			// GroupsClient guarantees a non-nil group and id on success; the generated mock is a
			// second implementation of the same interface and does not. Guard before every use.
			if subGroup == nil {
				return fmt.Errorf("subgroup %q not found in realm %q", claimed, realm)
			}

			if subGroup.Id == nil {
				return fmt.Errorf("subgroup %q has no id", claimed)
			}

			// Claiming a group that is itself managed by another KeycloakRealmGroup resource is
			// the documented behavior of this deprecated field: the move keeps the group ID, so
			// the owning resource still resolves it via status.ID afterwards.
			//
			// CreateChildGroup moves an existing group, so a nested match would be taken from
			// its current parent. Unreachable through Keycloak: the search above returns only
			// top-level roots. Kept for an alternate GroupsClient or a changed server response.
			if subGroup.ParentId != nil && *subGroup.ParentId != groupID {
				return fmt.Errorf(
					"subgroup %q is already nested under a different parent group (id %s); "+
						"use spec.parentGroup instead of the deprecated spec.subGroups field to manage it",
					claimed, *subGroup.ParentId,
				)
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
