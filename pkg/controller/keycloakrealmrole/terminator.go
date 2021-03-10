package keycloakrealmrole

import (
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/pkg/errors"
)

type terminator struct {
	realmName, realmRoleName string
	kClient                  keycloak.Client
}

func (t *terminator) DeleteResource() error {
	reqLog := log.WithValues("keycloak realm role cr", t.realmRoleName)
	reqLog.Info("Start deleting keycloak client...")

	if err := t.kClient.DeleteRealmRole(t.realmName, t.realmRoleName); err != nil {
		return errors.Wrap(err, "unable to delete realm role")
	}
	reqLog.Info("realm role deletion done")

	return nil
}

func makeTerminator(realmName, realmRoleName string, kClient keycloak.Client) *terminator {
	return &terminator{
		realmRoleName: realmRoleName,
		realmName:     realmName,
		kClient:       kClient,
	}
}
