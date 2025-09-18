package keycloakrealmgroup

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	kClient keycloak.Client
	realmName,
	groupName string
	preserveResourcesOnDeletion bool
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("realm_name", t.realmName, "group_name", t.groupName)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting group")

	if err := t.kClient.DeleteGroup(ctx, t.realmName, t.groupName); err != nil {
		return fmt.Errorf("unable to delete group %w", err)
	}

	log.Info("Group has been deleted")

	return nil
}

func makeTerminator(kClient keycloak.Client, realmName, groupName string, preserveResourcesOnDeletion bool) *terminator {
	return &terminator{
		kClient:                     kClient,
		realmName:                   realmName,
		groupName:                   groupName,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}
