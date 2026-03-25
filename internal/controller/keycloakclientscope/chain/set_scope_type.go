package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type SetScopeType struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewSetScopeType(kClientV2 *keycloakv2.KeycloakClient) *SetScopeType {
	return &SetScopeType{kClientV2: kClientV2}
}

func (h *SetScopeType) Serve(
	ctx context.Context,
	scope *keycloakApi.KeycloakClientScope,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Setting client scope type")

	scopesClient := h.kClientV2.ClientScopes
	scopeID := scope.Status.ID
	scopeType := scope.GetType()

	switch scopeType {
	case keycloakApi.KeycloakClientScopeTypeDefault:
		if _, err := scopesClient.RemoveRealmOptionalClientScope(ctx, realmName, scopeID); err != nil && !keycloakv2.IsNotFound(err) {
			return fmt.Errorf("failed to remove scope from optional list: %w", err)
		}

		if _, err := scopesClient.AddRealmDefaultClientScope(ctx, realmName, scopeID); err != nil {
			return fmt.Errorf("failed to add scope to default list: %w", err)
		}

	case keycloakApi.KeycloakClientScopeTypeOptional:
		if _, err := scopesClient.RemoveRealmDefaultClientScope(ctx, realmName, scopeID); err != nil && !keycloakv2.IsNotFound(err) {
			return fmt.Errorf("failed to remove scope from default list: %w", err)
		}

		if _, err := scopesClient.AddRealmOptionalClientScope(ctx, realmName, scopeID); err != nil {
			return fmt.Errorf("failed to add scope to optional list: %w", err)
		}

	case keycloakApi.KeycloakClientScopeTypeNone:
		if _, err := scopesClient.RemoveRealmDefaultClientScope(ctx, realmName, scopeID); err != nil && !keycloakv2.IsNotFound(err) {
			return fmt.Errorf("failed to remove scope from default list: %w", err)
		}

		if _, err := scopesClient.RemoveRealmOptionalClientScope(ctx, realmName, scopeID); err != nil && !keycloakv2.IsNotFound(err) {
			return fmt.Errorf("failed to remove scope from optional list: %w", err)
		}

	default:
		return fmt.Errorf("invalid client scope type: %s", scopeType)
	}

	log.Info("Client scope type has been set")

	return nil
}
