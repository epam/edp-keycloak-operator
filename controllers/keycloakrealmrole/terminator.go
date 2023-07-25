package keycloakrealmrole

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName, realmRoleName string
	kClient                  keycloak.Client
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Start deleting keycloak realm role")

	if err := t.kClient.DeleteRealmRole(ctx, t.realmName, t.realmRoleName); err != nil {
		return fmt.Errorf("unable to delete realm role %w", err)
	}

	log.Info("Realm role has been deleted")

	return nil
}

func makeTerminator(realmName, realmRoleName string, kClient keycloak.Client) *terminator {
	return &terminator{
		realmRoleName: realmRoleName,
		realmName:     realmName,
		kClient:       kClient,
	}
}
