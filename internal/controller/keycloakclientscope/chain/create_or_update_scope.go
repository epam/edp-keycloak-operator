package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakv2 "github.com/epam/edp-keycloak-operator/pkg/client/keycloakv2"
)

type CreateOrUpdateScope struct {
	kClientV2 *keycloakv2.KeycloakClient
}

func NewCreateOrUpdateScope(kClientV2 *keycloakv2.KeycloakClient) *CreateOrUpdateScope {
	return &CreateOrUpdateScope{kClientV2: kClientV2}
}

func (h *CreateOrUpdateScope) Serve(
	ctx context.Context,
	scope *keycloakApi.KeycloakClientScope,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating client scope")

	scopesClient := h.kClientV2.ClientScopes
	spec := scope.Spec

	existingScope, err := h.findScopeByName(ctx, realmName, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to find client scope by name: %w", err)
	}

	attrs := spec.Attributes
	desc := spec.Description
	protocol := spec.Protocol

	if existingScope == nil {
		resp, err := scopesClient.CreateClientScope(ctx, realmName, keycloakv2.ClientScopeRepresentation{
			Name:        &spec.Name,
			Protocol:    &protocol,
			Description: &desc,
			Attributes:  &attrs,
		})
		if err != nil {
			return fmt.Errorf("failed to create client scope: %w", err)
		}

		scope.Status.ID = keycloakv2.GetResourceIDFromResponse(resp)
	} else {
		if existingScope.Id != nil {
			scope.Status.ID = *existingScope.Id
		}

		_, err := scopesClient.UpdateClientScope(ctx, realmName, scope.Status.ID, keycloakv2.ClientScopeRepresentation{
			Name:        &spec.Name,
			Protocol:    &protocol,
			Description: &desc,
			Attributes:  &attrs,
		})
		if err != nil {
			return fmt.Errorf("failed to update client scope: %w", err)
		}
	}

	log.Info("Client scope has been synced")

	return nil
}

func (h *CreateOrUpdateScope) findScopeByName(
	ctx context.Context,
	realmName, scopeName string,
) (*keycloakv2.ClientScopeRepresentation, error) {
	scopes, _, err := h.kClientV2.ClientScopes.GetClientScopes(ctx, realmName)
	if err != nil {
		return nil, fmt.Errorf("failed to get client scopes: %w", err)
	}

	for i := range scopes {
		if scopes[i].Name != nil && *scopes[i].Name == scopeName {
			return &scopes[i], nil
		}
	}

	return nil, nil
}
