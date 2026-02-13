package keycloakrealmgroup

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type terminator struct {
	kClient                     *keycloakv2.KeycloakClient
	realmName                   string
	groupID                     string
	groupName                   string
	preserveResourcesOnDeletion bool
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("realm_name", t.realmName, "group_name", t.groupName, "group_id", t.groupID)

	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	if t.groupID == "" {
		log.Info("Group ID is empty, skipping deletion")
		return nil
	}

	log.Info("Start deleting group")

	if _, err := t.kClient.Groups.DeleteGroup(ctx, t.realmName, t.groupID); err != nil {
		if keycloakv2.IsNotFound(err) {
			log.Info("Group not found, skipping deletion")

			return nil
		}

		return fmt.Errorf("unable to delete group: %w", err)
	}

	log.Info("Group has been deleted")

	return nil
}

func makeTerminator(
	kClient *keycloakv2.KeycloakClient,
	realmName, groupID, groupName string,
	preserveResourcesOnDeletion bool,
) *terminator {
	return &terminator{
		kClient:                     kClient,
		realmName:                   realmName,
		groupID:                     groupID,
		groupName:                   groupName,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}
