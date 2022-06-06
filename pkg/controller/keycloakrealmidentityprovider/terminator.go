package keycloakrealmidentityprovider

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName string
	idpAlias  string
	kClient   keycloak.Client
	log       logr.Logger
}

func makeTerminator(realmName, idpAlias string, kClient keycloak.Client, log logr.Logger) *terminator {
	return &terminator{
		realmName: realmName,
		idpAlias:  idpAlias,
		kClient:   kClient,
		log:       log,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := t.log.WithValues("keycloak realm idp alias", t.idpAlias)
	log.Info("Start deleting keycloak realm idp...")

	if err := t.kClient.DeleteIdentityProvider(ctx, t.realmName, t.idpAlias); err != nil {
		return errors.Wrap(err, "unable to delete realm idp")
	}

	log.Info("realm idp deletion done")
	return nil
}

func (t *terminator) GetLogger() logr.Logger {
	return t.log
}
