package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// SyncMembers adds the resolved members to the group.
// With the full strategy, it also removes the group members that are not listed.
type SyncMembers struct {
	kClient *keycloakapi.KeycloakClient
}

func NewSyncMembers(kClient *keycloakapi.KeycloakClient) *SyncMembers {
	return &SyncMembers{kClient: kClient}
}

func (h *SyncMembers) Serve(
	ctx context.Context,
	membership *keycloakApi.KeycloakRealmGroupMembership,
	realmName string,
	membershipCtx *MembershipContext,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Syncing group members")

	current, err := listGroupMembers(ctx, h.kClient, realmName, membershipCtx.GroupID)
	if err != nil {
		return err
	}

	desired := make(map[string]struct{}, len(membershipCtx.Resolved))

	for _, m := range membershipCtx.Resolved {
		if _, ok := desired[m.UserID]; ok {
			continue
		}

		desired[m.UserID] = struct{}{}

		if _, ok := current[m.UserID]; ok {
			continue
		}

		log.Info("Adding user to group", "member", memberLabel(m.Member), "userID", m.UserID)

		if _, err := h.kClient.Users.AddUserToGroup(ctx, realmName, m.UserID, membershipCtx.GroupID); err != nil {
			return fmt.Errorf("unable to add user %s to group: %w", memberLabel(m.Member), err)
		}
	}

	if !membership.IsStrategyFull() {
		log.Info("Group members synced successfully (add-only)")

		return nil
	}

	for userID, username := range current {
		if _, keep := desired[userID]; keep {
			continue
		}

		log.Info("Removing user from group", "username", username, "userID", userID)

		if _, err := h.kClient.Users.RemoveUserFromGroup(ctx, realmName, userID, membershipCtx.GroupID); err != nil {
			if keycloakapi.IsNotFound(err) {
				continue
			}

			return fmt.Errorf("unable to remove user %q from group: %w", username, err)
		}
	}

	log.Info("Group members synced successfully")

	return nil
}
