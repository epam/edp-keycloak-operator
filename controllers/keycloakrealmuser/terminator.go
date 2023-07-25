package keycloakrealmuser

import (
	"context"
	"fmt"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
)

type terminator struct {
	kClient             keycloak.Client
	realmName, userName string
}

func (t *terminator) DeleteResource(ctx context.Context) error {
	if err := t.kClient.DeleteRealmUser(ctx, t.realmName, t.userName); err != nil {
		return fmt.Errorf("unable to delete realm user %w", err)
	}

	return nil
}

func makeTerminator(realmName, userName string, kClient keycloak.Client) *terminator {
	return &terminator{
		kClient:   kClient,
		realmName: realmName,
		userName:  userName,
	}
}
