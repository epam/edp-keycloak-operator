package keycloakclient

import (
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	pkgErrors "github.com/pkg/errors"
)

type terminator struct {
	clientID, realmName string
	kClient             keycloak.Client
}

func makeTerminator(clientID, realmName string, kClient keycloak.Client) *terminator {
	return &terminator{clientID: clientID, realmName: realmName, kClient: kClient}
}

func (t *terminator) DeleteResource() error {
	reqLog := log.WithValues("keycloak client id", t.clientID)
	reqLog.Info("Start deleting keycloak client...")

	if err := t.kClient.DeleteClient(t.clientID, t.realmName); err != nil {
		return pkgErrors.Wrap(err, "unable to delete kk client")
	}

	reqLog.Info("client deletion done")

	return nil
}
