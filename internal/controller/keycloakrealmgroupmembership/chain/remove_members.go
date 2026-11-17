package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/objectmeta"
)

// RemoveMembers removes the members listed in the spec from the group on deletion.
// Members added by other means are left alone, whatever the strategy.
type RemoveMembers struct {
	kClient *keycloakapi.KeycloakClient
}

func NewRemoveMembers(kClient *keycloakapi.KeycloakClient) *RemoveMembers {
	return &RemoveMembers{kClient: kClient}
}

func (h *RemoveMembers) ServeRequest(
	ctx context.Context,
	membership *keycloakApi.KeycloakRealmGroupMembership,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start removing group members")

	if objectmeta.PreserveResourcesOnDeletion(membership) {
		log.Info("Preserve resources on deletion, skipping")

		return nil
	}

	groupID, err := findGroupID(ctx, h.kClient, realmName, membership.Spec.Group)
	if err != nil {
		if keycloakapi.IsNotFound(err) {
			log.Info("Group not found, skipping")

			return nil
		}

		return err
	}

	for _, member := range membership.Spec.Members {
		userID, err := findUser(ctx, h.kClient, realmName, member)
		if err != nil {
			if keycloakapi.IsNotFound(err) {
				continue
			}

			return err
		}

		if _, err := h.kClient.Users.RemoveUserFromGroup(ctx, realmName, userID, groupID); err != nil {
			if keycloakapi.IsNotFound(err) {
				continue
			}

			return fmt.Errorf("unable to remove user %s from group: %w", memberLabel(member), err)
		}
	}

	log.Info("Group members removed successfully")

	return nil
}
