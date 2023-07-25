package keycloakrealm

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName string
	kClient   keycloak.Client
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx).WithValues("keycloak_realm", t.realmName)
	log.Info("Start deleting keycloak realm")

	if err := t.kClient.DeleteRealm(ctx, t.realmName); err != nil {
		return fmt.Errorf("failed to delete keycloak realm: %w", err)
	}

	log.Info("Realm has been deleted")

	return nil
}

func makeTerminator(realmName string, kClient keycloak.Client) *terminator {
	return &terminator{
		realmName: realmName,
		kClient:   kClient,
	}
}
