package keycloakrealmrolebatch

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

type terminator struct {
	client                      client.Client
	childRoles                  []keycloakApi.KeycloakRealmRole
	preserveResourcesOnDeletion bool
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting keycloak realm role batch")

	for i := range t.childRoles {
		if err := t.client.Delete(ctx, &t.childRoles[i]); err != nil {
			return fmt.Errorf("unable to delete realm role %w", err)
		}
	}

	log.Info("Realm role batch has been deleted")

	return nil
}

func makeTerminator(client client.Client, childRoles []keycloakApi.KeycloakRealmRole, preserveResourcesOnDeletion bool) *terminator {
	return &terminator{
		client:                      client,
		childRoles:                  childRoles,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}
