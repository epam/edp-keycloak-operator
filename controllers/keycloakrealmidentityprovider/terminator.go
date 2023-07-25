package keycloakrealmidentityprovider

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName string
	idpAlias  string
	kClient   keycloak.Client
}

func makeTerminator(realmName, idpAlias string, kClient keycloak.Client) *terminator {
	return &terminator{
		realmName: realmName,
		idpAlias:  idpAlias,
		kClient:   kClient,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("keycloak_realm_idp_alias", t.idpAlias)
	log.Info("Start deleting keycloak realm idp")

	if err := t.kClient.DeleteIdentityProvider(ctx, t.realmName, t.idpAlias); err != nil {
		return fmt.Errorf("unable to delete realm idp %w", err)
	}

	log.Info("Realm idp has been deleted")

	return nil
}
