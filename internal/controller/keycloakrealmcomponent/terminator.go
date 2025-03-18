package keycloakrealmcomponent

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName                   string
	componentName               string
	kClient                     keycloak.Client
	preserveResourcesOnDeletion bool
}

func makeTerminator(realmName, componentName string, kClient keycloak.Client, preserveResourcesOnDeletion bool) *terminator {
	return &terminator{
		realmName:                   realmName,
		componentName:               componentName,
		kClient:                     kClient,
		preserveResourcesOnDeletion: preserveResourcesOnDeletion,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	if t.preserveResourcesOnDeletion {
		log.Info("PreserveResourcesOnDeletion is enabled, skipping deletion.")
		return nil
	}

	log.Info("Start deleting KeycloakRealmComponent")

	if err := t.kClient.DeleteComponent(ctx, t.realmName, t.componentName); err != nil {
		return fmt.Errorf("unable to delete realm component %w", err)
	}

	log.Info("KeycloakRealmComponent deletion done")

	return nil
}
