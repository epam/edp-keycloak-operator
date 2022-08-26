package adapter

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v10"
	"github.com/pkg/errors"
)

func (a GoCloakAdapter) AddDefaultScopeToClient(ctx context.Context, realmName, clientName string, scopes []ClientScope) error {
	log := a.log.WithValues("clientName", clientName, logKeyRealm, realmName)
	log.Info("Start add Client Scopes to client...")

	clientID, err := a.GetClientID(clientName, realmName)
	if err != nil {
		return errors.Wrap(err, "error during GetClientId")
	}

	existingScopes, err := a.client.GetClientsDefaultScopes(ctx, a.token.AccessToken, realmName, clientID)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to get existing client scope for client %s", clientName))
	}

	existingScopesMap := make(map[string]*gocloak.ClientScope)

	for _, s := range existingScopes {
		if s != nil {
			existingScopesMap[*s.ID] = s
		}
	}

	for _, scope := range scopes {
		if _, ok := existingScopesMap[scope.ID]; ok {
			continue
		}

		err := a.client.AddDefaultScopeToClient(ctx, a.token.AccessToken, realmName, clientID, scope.ID)
		if err != nil {
			a.log.Error(err, fmt.Sprintf("failed link scope %s to client %s", scope.Name, clientName))
		}
	}

	log.Info("End add Client Scopes to client...")

	return nil
}
