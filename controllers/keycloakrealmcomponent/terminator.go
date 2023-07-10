package keycloakrealmcomponent

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName     string
	componentName string
	kClient       keycloak.Client
	log           logr.Logger
}

func makeTerminator(realmName, componentName string, kClient keycloak.Client, log logr.Logger) *terminator {
	return &terminator{
		realmName:     realmName,
		componentName: componentName,
		kClient:       kClient,
		log:           log,
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

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}
