package keycloakrealm

import (
	"github.com/epmd-edp/keycloak-operator/pkg/client/keycloak"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type terminator struct {
	realmName string
	kClient   keycloak.Client
	log       logr.Logger
}

func (t *terminator) DeleteResource() error {
	log := t.log.WithValues("keycloak realm cr", t.realmName)
	log.Info("Start deleting keycloak realm...")

	if err := t.kClient.DeleteRealm(t.realmName); err != nil {
		return errors.Wrap(err, "unable to delete realm")
	}

	log.Info("client deletion done")

	return nil
}

func makeTerminator(realmName string, kClient keycloak.Client) *terminator {
	return &terminator{
		realmName: realmName,
		kClient:   kClient,
		log:       ctrl.Log.WithName("keycloak-realm-terminator"),
	}
}
