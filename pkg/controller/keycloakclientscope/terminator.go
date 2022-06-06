package keycloakclientscope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName, scopeID string
	kClient            keycloak.Client
	log                logr.Logger
}

func makeTerminator(kClient keycloak.Client, realmName, scopeID string, log logr.Logger) *terminator {
	return &terminator{
		kClient:   kClient,
		realmName: realmName,
		scopeID:   scopeID,
		log:       log,
	}
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	logger := t.log.WithValues("realm name", t.realmName, "scope id", t.scopeID)

	logger.Info("start deleting client scope")
	if err := t.kClient.DeleteClientScope(ctx, t.realmName, t.scopeID); err != nil {
		return errors.Wrap(err, "unable to delete client scope")
	}

	logger.Info("done deleting client scope")
	return nil
}
