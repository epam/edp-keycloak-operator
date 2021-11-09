package keycloakrealm

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type terminator struct {
	realmName string
	kClient   keycloak.Client
	log       logr.Logger
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := t.log.WithValues("keycloak realm cr", t.realmName)
	log.Info("Start deleting keycloak realm...")

	if err := t.kClient.DeleteRealm(ctx, t.realmName); err != nil {
		return errors.Wrap(err, "unable to delete realm")
	}

	log.Info("realm deletion done")
	return nil
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}

func makeTerminator(realmName string, kClient keycloak.Client, log logr.Logger) *terminator {
	return &terminator{
		realmName: realmName,
		kClient:   kClient,
		log:       log,
	}
}
