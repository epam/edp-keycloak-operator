package keycloakclient

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	clientID, realmName         string
	kClient                     keycloak.Client
	preserveResourcesOnDeletion bool
}

func makeTerminator(clientID, realmName string, kClient keycloak.Client, preserveResourcesOnDeletion bool) *terminator {
	return &terminator{
		clientID:                    clientID,
		realmName:                   realmName,
		kClient:                     kClient,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("client_id", t.clientID)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting keycloak client")

	if err := t.kClient.DeleteClient(ctx, t.clientID, t.realmName); err != nil {
		return fmt.Errorf("failed to delete keycloak client: %w", err)
	}

	log.Info("Keycloak client has been deleted")

	return nil
}
