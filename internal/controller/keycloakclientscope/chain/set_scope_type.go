package chain

import (
	"context"
	"fmt"
	"slices"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

type SetScopeType struct {
	kClient *keycloakapi.KeycloakClient
}

func NewSetScopeType(kClient *keycloakapi.KeycloakClient) *SetScopeType {
	return &SetScopeType{kClient: kClient}
}

func (h *SetScopeType) Serve(
	ctx context.Context,
	scope *keycloakApi.KeycloakClientScope,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Setting client scope type")

	scopesClient := h.kClient.ClientScopes
	scopeID := scope.Status.ID
	scopeType := scope.GetType()

	if scopeType != keycloakApi.KeycloakClientScopeTypeDefault &&
		scopeType != keycloakApi.KeycloakClientScopeTypeOptional &&
		scopeType != keycloakApi.KeycloakClientScopeTypeNone {
		return fmt.Errorf("invalid client scope type: %s", scopeType)
	}

	inDefault, err := h.scopeInRealmList(ctx, realmName, scopeID, scopesClient.GetRealmDefaultClientScopes)
	if err != nil {
		return fmt.Errorf("failed to get realm default client scopes: %w", err)
	}

	inOptional, err := h.scopeInRealmList(ctx, realmName, scopeID, scopesClient.GetRealmOptionalClientScopes)
	if err != nil {
		return fmt.Errorf("failed to get realm optional client scopes: %w", err)
	}

	wantDefault := scopeType == keycloakApi.KeycloakClientScopeTypeDefault
	wantOptional := scopeType == keycloakApi.KeycloakClientScopeTypeOptional

	if inDefault && !wantDefault {
		if _, err := scopesClient.RemoveRealmDefaultClientScope(ctx, realmName, scopeID); err != nil && !keycloakapi.IsNotFound(err) {
			return fmt.Errorf("failed to remove scope from default list: %w", err)
		}
	}

	if inOptional && !wantOptional {
		if _, err := scopesClient.RemoveRealmOptionalClientScope(ctx, realmName, scopeID); err != nil && !keycloakapi.IsNotFound(err) {
			return fmt.Errorf("failed to remove scope from optional list: %w", err)
		}
	}

	if !inDefault && wantDefault {
		if _, err := scopesClient.AddRealmDefaultClientScope(ctx, realmName, scopeID); err != nil {
			return fmt.Errorf("failed to add scope to default list: %w", err)
		}
	}

	if !inOptional && wantOptional {
		if _, err := scopesClient.AddRealmOptionalClientScope(ctx, realmName, scopeID); err != nil {
			return fmt.Errorf("failed to add scope to optional list: %w", err)
		}
	}

	log.Info("Client scope type has been set")

	return nil
}

func (h *SetScopeType) scopeInRealmList(
	ctx context.Context,
	realmName, scopeID string,
	list func(ctx context.Context, realm string) ([]keycloakapi.ClientScopeRepresentation, *keycloakapi.Response, error),
) (bool, error) {
	scopes, _, err := list(ctx, realmName)
	if err != nil {
		return false, err
	}

	return slices.ContainsFunc(scopes, func(s keycloakapi.ClientScopeRepresentation) bool {
		return s.Id != nil && *s.Id == scopeID
	}), nil
}
