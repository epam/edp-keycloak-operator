package keycloakrealmcomponent

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

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
	log := t.log.WithValues("keycloak realm component name", t.componentName)
	log.Info("Start deleting keycloak realm component...")

	if err := t.kClient.DeleteComponent(ctx, t.realmName, t.componentName); err != nil {
		return errors.Wrap(err, "unable to delete realm component")
	}

	log.Info("realm component deletion done")
	return nil
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}
