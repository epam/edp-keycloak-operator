package keycloakclientscope

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	realmName, scopeID string
	kClient            keycloak.Client
}

func makeTerminator(kClient keycloak.Client, realmName, scopeID string) *terminator {
	return &terminator{
		kClient:   kClient,
		realmName: realmName,
		scopeID:   scopeID,
	}
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	logger := ctrl.LoggerFrom(ctx).WithValues("realm name", t.realmName, "scope id", t.scopeID)

	logger.Info("Start deleting client scope")

	if err := t.kClient.DeleteClientScope(ctx, t.realmName, t.scopeID); err != nil {
		return fmt.Errorf("failed to delete client scope: %w", err)
	}

	logger.Info("Client scope has been deleted")

	return nil
}
