package keycloakrealmrole

import (
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type terminator struct {
	realmName, realmRoleName string
	kClient                  keycloak.Client
	logger                   logr.Logger
}

func (t *terminator) DeleteResource() error {
	reqLog := t.logger.WithValues("keycloak realm role cr", t.realmRoleName)
	reqLog.Info("Start deleting keycloak client...")

	if err := t.kClient.DeleteRealmRole(t.realmName, t.realmRoleName); err != nil {
		return errors.Wrap(err, "unable to delete realm role")
	}
	reqLog.Info("realm role deletion done")

	return nil
}

func makeTerminator(realmName, realmRoleName string, kClient keycloak.Client, logger logr.Logger) *terminator {
	return &terminator{
		realmRoleName: realmRoleName,
		realmName:     realmName,
		kClient:       kClient,
		logger:        logger,
	}
}
