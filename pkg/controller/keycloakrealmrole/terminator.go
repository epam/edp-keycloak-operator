package keycloakrealmrole

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName, realmRoleName string
	kClient                  keycloak.Client
	log                      logr.Logger
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := t.log.WithValues("keycloak realm role cr", t.realmRoleName)
	log.Info("Start deleting keycloak realm role...")

	if err := t.kClient.DeleteRealmRole(ctx, t.realmName, t.realmRoleName); err != nil {
		return errors.Wrap(err, "unable to delete realm role")
	}

	log.Info("realm role deletion done")
	return nil
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}

func makeTerminator(realmName, realmRoleName string, kClient keycloak.Client, log logr.Logger) *terminator {
	return &terminator{
		realmRoleName: realmRoleName,
		realmName:     realmName,
		kClient:       kClient,
		log:           log,
	}
}
