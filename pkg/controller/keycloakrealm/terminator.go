package keycloakrealm

import (
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/pkg/errors"
)

type terminator struct {
	realmName string
	kClient   keycloak.Client
}

func (t *terminator) DeleteResource() error {
	reqLog := log.WithValues("keycloak realm cr", t.realmName)
	reqLog.Info("Start deleting keycloak realm...")

	if err := t.kClient.DeleteRealm(t.realmName); err != nil {
		return errors.Wrap(err, "unable to delete realm")
	}

	reqLog.Info("client deletion done")

	return nil
}

func makeTerminator(realmName string, kClient keycloak.Client) *terminator {
	return &terminator{
		realmName: realmName,
		kClient:   kClient,
	}
}
