package keycloakrealmrole

import (
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type terminator struct {
	realmName, realmRoleName string
	kClient                  keycloak.Client
	log                      logr.Logger
}

func (t *terminator) DeleteResource() error {
	log := t.log.WithValues("keycloak realm role cr", t.realmRoleName)
	log.Info("Start deleting keycloak client...")

	if err := t.kClient.DeleteRealmRole(t.realmName, t.realmRoleName); err != nil {
		return errors.Wrap(err, "unable to delete realm role")
	}
	log.Info("realm role deletion done")

	return nil
}

func makeTerminator(realmName, realmRoleName string, kClient keycloak.Client) *terminator {
	return &terminator{
		realmRoleName: realmRoleName,
		realmName:     realmName,
		kClient:       kClient,
		log:           ctrl.Log.WithName("keycloak-realm-role-terminator"),
	}
}
