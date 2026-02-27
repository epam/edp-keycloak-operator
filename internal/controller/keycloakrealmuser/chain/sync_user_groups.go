package chain

import (
	"context"
	"fmt"
	"strings"

	"github.com/Nerzal/gocloak/v12"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type SyncUserGroups struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewSyncUserGroups(kClientV2 *keycloakv2.KeycloakClient) *SyncUserGroups {
	return &SyncUserGroups{kClientV2: kClientV2}
}

// resolvedGroup holds a Keycloak group identity resolved from a spec entry.
type resolvedGroup struct {
	id   string
	name string
	path string
}

// label returns a human-readable identifier for the group (path if available, otherwise name).
func (g resolvedGroup) label() string {
	if g.path != "" {
		return g.path
	}

	return g.name
}

func (h *SyncUserGroups) Serve(
	ctx context.Context,
	user *keycloakApi.KeycloakRealmUser,
	_ keycloak.Client,
	realm *gocloak.RealmRepresentation,
	userCtx *UserContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing user groups")

	if len(user.Spec.Groups) == 0 && user.IsReconciliationStrategyAddOnly() {
		log.Info("No groups specified (add-only), skipping")
		return nil
	}

	realmName := ptr.Deref(realm.Realm, "")

	currentGroups, _, err := h.kClientV2.Users.GetUserGroups(ctx, realmName, userCtx.UserID)
	if err != nil {
		return fmt.Errorf("unable to get user groups: %w", err)
	}

	desired := make([]resolvedGroup, 0, len(user.Spec.Groups))

	for _, g := range user.Spec.Groups {
		if strings.HasPrefix(g, "/") {
			grp, _, err := h.kClientV2.Groups.GetGroupByPath(ctx, realmName, g)
			if err != nil {
				return fmt.Errorf("unable to get group by path %q: %w", g, err)
			}

			if grp == nil || grp.Id == nil {
				return fmt.Errorf("group not found by path %q", g)
			}

			desired = append(desired, resolvedGroup{id: *grp.Id, path: g})
		} else {
			grp, _, err := h.kClientV2.Groups.FindGroupByName(ctx, realmName, g)
			if err != nil {
				return fmt.Errorf("unable to find group by name %q: %w", g, err)
			}

			if grp == nil || grp.Id == nil {
				return fmt.Errorf("group not found by name %q", g)
			}

			desired = append(desired, resolvedGroup{id: *grp.Id, name: g})
		}
	}

	// build lookup of current group IDs
	currentByID := make(map[string]keycloakv2.GroupRepresentation, len(currentGroups))

	for _, cg := range currentGroups {
		if cg.Id != nil {
			currentByID[*cg.Id] = cg
		}
	}

	// add missing groups
	for _, dg := range desired {
		if _, exists := currentByID[dg.id]; !exists {
			log.V(1).Info("Adding user to group", "group", dg.label(), "groupID", dg.id)

			if _, err := h.kClientV2.Users.AddUserToGroup(ctx, realmName, userCtx.UserID, dg.id); err != nil {
				return fmt.Errorf("unable to add user to group %q (id %s): %w", dg.label(), dg.id, err)
			}
		}
	}

	if user.IsReconciliationStrategyAddOnly() {
		log.Info("User groups synced successfully (add-only)")
		return nil
	}

	// build lookup of desired IDs for removal check
	desiredIDs := make(map[string]struct{}, len(desired))
	for _, dg := range desired {
		desiredIDs[dg.id] = struct{}{}
	}

	// remove groups no longer desired
	for id, cg := range currentByID {
		if _, keep := desiredIDs[id]; !keep {
			groupLabel := id
			if cg.Name != nil {
				groupLabel = *cg.Name
			}

			log.V(1).Info("Removing user from group", "group", groupLabel, "groupID", id)

			if _, err := h.kClientV2.Users.RemoveUserFromGroup(ctx, realmName, userCtx.UserID, id); err != nil {
				return fmt.Errorf("unable to remove user from group %q (id %s): %w", groupLabel, id, err)
			}
		}
	}

	log.Info("User groups synced successfully")

	return nil
}
