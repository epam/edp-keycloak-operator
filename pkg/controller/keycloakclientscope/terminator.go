package keycloakclientscope

import (
	"context"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/pkg/errors"
)

type terminator struct {
	realmName, scopeID string
	kClient            keycloak.Client
	ctx                context.Context
}

func makeTerminator(ctx context.Context, kClient keycloak.Client, realmName, scopeID string) *terminator {
	return &terminator{
		ctx:       ctx,
		kClient:   kClient,
		realmName: realmName,
		scopeID:   scopeID,
	}
}

func (t *terminator) DeleteResource() error {
	if err := t.kClient.DeleteClientScope(t.ctx, t.realmName, t.scopeID); err != nil {
		return errors.Wrap(err, "unable to delete client scope")
	}

	return nil
}
