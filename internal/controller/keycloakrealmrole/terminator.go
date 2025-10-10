package keycloakrealmrole

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type terminator struct {
	realmName, realmRoleName    string
	kClient                     keycloak.Client
	preserveResourcesOnDeletion bool
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting keycloak realm role")

	if err := t.kClient.DeleteRealmRole(ctx, t.realmName, t.realmRoleName); err != nil {
		if adapter.IsErrNotFound(err) {
			log.Info("Realm role not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("unable to delete realm role %w", err)
	}

	log.Info("Realm role has been deleted")

	return nil
}

func makeTerminator(realmName, realmRoleName string, kClient keycloak.Client, preserveResourcesOnDeletion bool) *terminator {
	return &terminator{
		realmRoleName:               realmRoleName,
		realmName:                   realmName,
		kClient:                     kClient,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}
