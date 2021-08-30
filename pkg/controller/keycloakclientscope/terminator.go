package keycloakclientscope

import (
	"context"

	"github.com/go-logr/logr"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/pkg/errors"
)

type terminator struct {
	realmName, scopeID string
	kClient            keycloak.Client
	ctx                context.Context
	log                logr.Logger
}

func makeTerminator(ctx context.Context, kClient keycloak.Client, realmName, scopeID string, log logr.Logger) *terminator {
	return &terminator{
		ctx:       ctx,
		kClient:   kClient,
		realmName: realmName,
		scopeID:   scopeID,
		log:       log,
	}
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}

func (t *terminator) DeleteResource() error {
	logger := t.log.WithValues("realm name", t.realmName, "scope id", t.scopeID)

	logger.Info("start deleting client scope")
	if err := t.kClient.DeleteClientScope(t.ctx, t.realmName, t.scopeID); err != nil {
		return errors.Wrap(err, "unable to delete client scope")
	}

	logger.Info("done deleting client scope")
	return nil
}
