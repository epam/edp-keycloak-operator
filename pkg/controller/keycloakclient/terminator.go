package keycloakclient

import (
	"github.com/epam/keycloak-operator/pkg/client/keycloak"
	"github.com/go-logr/logr"
	pkgErrors "github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type terminator struct {
	clientID, realmName string
	kClient             keycloak.Client
	log                 logr.Logger
}

func makeTerminator(clientID, realmName string, kClient keycloak.Client) *terminator {
	return &terminator{
		clientID:  clientID,
		realmName: realmName,
		kClient:   kClient,
		log:       ctrl.Log.WithName("keycloak-client-terminator"),
	}
}

func (t *terminator) DeleteResource() error {
	log := t.log.WithValues("keycloak client id", t.clientID)
	log.Info("Start deleting keycloak client...")

	if err := t.kClient.DeleteClient(t.clientID, t.realmName); err != nil {
		return pkgErrors.Wrap(err, "unable to delete kk client")
	}

	log.Info("client deletion done")

	return nil
}
