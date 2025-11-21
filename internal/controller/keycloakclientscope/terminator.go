package keycloakclientscope

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

type terminator struct {
	realmName, scopeID          string
	kClient                     keycloak.Client
	preserveResourcesOnDeletion bool
}

func makeTerminator(kClient keycloak.Client, realmName, scopeID string, preserveResourcesOnDeletion bool) *terminator {
	return &terminator{
		kClient:                     kClient,
		realmName:                   realmName,
		scopeID:                     scopeID,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("realm name", t.realmName, "scope id", t.scopeID)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting client scope")

	if err := t.kClient.DeleteClientScope(ctx, t.realmName, t.scopeID); err != nil {
		if adapter.IsErrNotFound(err) {
			log.Info("Client scope not found, skipping deletion.")

			return nil
		}

		return fmt.Errorf("failed to delete client scope: %w", err)
	}

	log.Info("Client scope has been deleted")

	return nil
}
