package keycloakrealmuser

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

type terminator struct {
	kClient             keycloak.Client
	log                 logr.Logger
	realmName, userName string
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	if err := t.kClient.DeleteRealmUser(ctx, t.realmName, t.userName); err != nil {
		return errors.Wrap(err, "unable to delete realm user")
	}

	return nil
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}

func makeTerminator(realmName, userName string, kClient keycloak.Client, log logr.Logger) *terminator {
	return &terminator{
		kClient:   kClient,
		log:       log,
		realmName: realmName,
		userName:  userName,
	}
}
