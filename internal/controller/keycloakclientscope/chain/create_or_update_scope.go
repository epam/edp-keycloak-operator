package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

type CreateOrUpdateScope struct {
	kClient *keycloakapi.KeycloakClient
}

func NewCreateOrUpdateScope(kClient *keycloakapi.KeycloakClient) *CreateOrUpdateScope {
	return &CreateOrUpdateScope{kClient: kClient}
}

func (h *CreateOrUpdateScope) Serve(
	ctx context.Context,
	scope *keycloakApi.KeycloakClientScope,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating client scope")

	scopesClient := h.kClient.ClientScopes
	spec := scope.Spec

	existingScope, err := h.findScopeByName(ctx, realmName, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to find client scope by name: %w", err)
	}

	attrs := spec.Attributes
	desc := spec.Description
	protocol := spec.Protocol

	if existingScope == nil {
		resp, err := scopesClient.CreateClientScope(ctx, realmName, keycloakapi.ClientScopeRepresentation{
			Name:        &spec.Name,
			Protocol:    &protocol,
			Description: &desc,
			Attributes:  &attrs,
		})
		if err != nil {
			return fmt.Errorf("failed to create client scope: %w", err)
		}

		scope.Status.ID = keycloakapi.GetResourceIDFromResponse(resp)
	} else {
		if existingScope.Id != nil {
			scope.Status.ID = *existingScope.Id
		}

		if helper.SpecChanged(scope.Generation, scope.Status.ObservedGeneration) || !clientScopeMatchesSpec(existingScope, &spec) {
			_, err := scopesClient.UpdateClientScope(ctx, realmName, scope.Status.ID, keycloakapi.ClientScopeRepresentation{
				Name:        &spec.Name,
				Protocol:    &protocol,
				Description: &desc,
				Attributes:  &attrs,
			})
			if err != nil {
				return fmt.Errorf("failed to update client scope: %w", err)
			}
		}
	}

	log.Info("Client scope has been synced")

	return nil
}

func clientScopeMatchesSpec(existing *keycloakapi.ClientScopeRepresentation, spec *keycloakApi.KeycloakClientScopeSpec) bool {
	return ptr.Deref(existing.Name, "") == spec.Name &&
		ptr.Deref(existing.Protocol, "") == spec.Protocol &&
		ptr.Deref(existing.Description, "") == spec.Description &&
		maputil.ContainsSubset(ptr.Deref(existing.Attributes, nil), spec.Attributes)
}

func (h *CreateOrUpdateScope) findScopeByName(
	ctx context.Context,
	realmName, scopeName string,
) (*keycloakapi.ClientScopeRepresentation, error) {
	scopes, _, err := h.kClient.ClientScopes.GetClientScopes(ctx, realmName)
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
