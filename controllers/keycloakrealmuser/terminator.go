package keycloakrealmuser

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	kClient                     keycloak.Client
	realmName, userName         string
	preserveResourcesOnDeletion bool
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting keycloak realm user")

	if err := t.kClient.DeleteRealmUser(ctx, t.realmName, t.userName); err != nil {
		return fmt.Errorf("unable to delete realm user %w", err)
	}

	log.Info("Realm user has been deleted")

	return nil
}

func makeTerminator(realmName, userName string, kClient keycloak.Client, preserveResourcesOnDeletion bool) *terminator {
	return &terminator{
		kClient:                     kClient,
		realmName:                   realmName,
		userName:                    userName,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}
