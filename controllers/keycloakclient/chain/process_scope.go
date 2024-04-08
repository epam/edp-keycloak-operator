package chain

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

const scopeLogKey = "scope"

type ProcessScope struct {
	keycloakApiClient keycloak.Client
}

func NewProcessScope(keycloakApi keycloak.Client) *ProcessScope {
	return &ProcessScope{keycloakApiClient: keycloakApi}
}

func (h *ProcessScope) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	if keycloakClient.Spec.Authorization == nil {
		log.Info("Authorization settings are not specified")
		return nil
	}

	clientID, err := h.keycloakApiClient.GetClientID(keycloakClient.Spec.ClientId, realmName)
	if err != nil {
		return fmt.Errorf("failed to get client id: %w", err)
	}

	existingScopes, err := h.keycloakApiClient.GetScopes(ctx, realmName, clientID)
	if err != nil {
		return fmt.Errorf("failed to get scopes: %w", err)
	}

	for _, scope := range keycloakClient.Spec.Authorization.Scopes {
		log.Info("Processing scope", scopeLogKey, scope)

		_, ok := existingScopes[scope]
		if ok {
			log.Info("Scope already exists")
			delete(existingScopes, scope)

			continue
		}

		if _, err = h.keycloakApiClient.CreateScope(ctx, realmName, clientID, scope); err != nil {
			return fmt.Errorf("failed to create scope: %w", err)
		}

		log.Info("Scope created", scopeLogKey, scope)

		delete(existingScopes, scope)
	}

	if err = h.deleteScopes(ctx, existingScopes, realmName, clientID); err != nil {
		return err
	}

	return nil
}

func (h *ProcessScope) deleteScopes(ctx context.Context, existingScopes map[string]gocloak.ScopeRepresentation, realmName string, clientID string) error {
	log := ctrl.LoggerFrom(ctx)

	for name := range existingScopes {
		if err := h.keycloakApiClient.DeleteScope(ctx, realmName, clientID, *existingScopes[name].ID); err != nil {
			if !adapter.IsErrNotFound(err) {
				return fmt.Errorf("failed to delete scope: %w", err)
			}
		}

		log.Info("Scope deleted", scopeLogKey, name)
	}

	return nil
}
