package keycloakauthflow

import (
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type terminator struct {
	realmName, flowAlias string
	kClient              keycloak.Client
	log                  logr.Logger
}

func makeTerminator(realmName, flowAlias string, kClient keycloak.Client, log logr.Logger) *terminator {
	return &terminator{
		realmName: realmName,
		flowAlias: flowAlias,
		kClient:   kClient,
		log:       log,
	}
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}

func (t *terminator) DeleteResource() error {
	logger := t.log.WithValues("realm name", t.realmName, "flow alias", t.flowAlias)

	logger.Info("start deleting auth flow")
	if err := t.kClient.DeleteAuthFlow(t.realmName, t.flowAlias); err != nil {
		return errors.Wrap(err, "unable to delete auth flow")
	}

	logger.Info("deleting auth flow done")
	return nil
}
