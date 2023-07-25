package keycloakrealmcomponent

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName     string
	componentName string
	kClient       keycloak.Client
}

func makeTerminator(realmName, componentName string, kClient keycloak.Client) *terminator {
	return &terminator{
		realmName:     realmName,
		componentName: componentName,
		kClient:       kClient,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start deleting KeycloakRealmComponent")

	if err := t.kClient.DeleteComponent(ctx, t.realmName, t.componentName); err != nil {
		return fmt.Errorf("unable to delete realm component %w", err)
	}

	log.Info("KeycloakRealmComponent deletion done")

	return nil
}
