package keycloakrealmrole

import (
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type terminator struct {
	realmName, realmRoleName string
	kClient                  keycloak.Client
	log                      logr.Logger
}

func (t *terminator) DeleteResource() error {
	log := t.log.WithValues("keycloak realm role cr", t.realmRoleName)
	log.Info("Start deleting keycloak realm role...")

	if err := t.kClient.DeleteRealmRole(t.realmName, t.realmRoleName); err != nil {
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
