package keycloakrealmgroup

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type terminator struct {
	kClient                     *keycloakapi.APIClient
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
		if keycloakapi.IsNotFound(err) {
			log.Info("Group not found, skipping deletion")

			return nil
		}

		return fmt.Errorf("unable to delete group: %w", err)
	}

	log.Info("Group has been deleted")

	return nil
}

func makeTerminator(
	kClient *keycloakapi.APIClient,
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
