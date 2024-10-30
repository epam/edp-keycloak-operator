package chain

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/Nerzal/gocloak/v12"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloak/adapter"
)

const resourceLogKey = "resource"

type ProcessResources struct {
	keycloakApiClient keycloak.Client
}

func NewProcessResources(keycloakApiClient keycloak.Client) *ProcessResources {
	return &ProcessResources{keycloakApiClient: keycloakApiClient}
}

func (h *ProcessResources) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string) error {
	log := ctrl.LoggerFrom(ctx)

	if keycloakClient.Spec.Authorization == nil {
		log.Info("Authorization settings are not specified")
		return nil
	}

	clientID, err := h.keycloakApiClient.GetClientID(keycloakClient.Spec.ClientId, realmName)
	if err != nil {
		return fmt.Errorf("failed to get client id: %w", err)
	}

	existingResources, err := h.keycloakApiClient.GetResources(ctx, realmName, clientID)
	if err != nil {
		return fmt.Errorf("failed to get resources: %w", err)
	}

	for i := 0; i < len(keycloakClient.Spec.Authorization.Resources); i++ {
		log.Info("Processing resource", resourceLogKey, keycloakClient.Spec.Authorization.Resources[i].Name)

		var resourceRepresentation *gocloak.ResourceRepresentation

		if resourceRepresentation, err = h.toResourceRepresentation(ctx, &keycloakClient.Spec.Authorization.Resources[i], clientID, realmName); err != nil {
			return fmt.Errorf("failed to convert resource: %w", err)
		}

		existingResource, ok := existingResources[keycloakClient.Spec.Authorization.Resources[i].Name]
		if ok {
			resourceRepresentation.ID = existingResource.ID
			if err = h.keycloakApiClient.UpdateResource(ctx, realmName, clientID, *resourceRepresentation); err != nil {
				return fmt.Errorf("failed to update resource: %w", err)
			}

			log.Info("Resource updated", resourceLogKey, keycloakClient.Spec.Authorization.Resources[i].Name)

			delete(existingResources, keycloakClient.Spec.Authorization.Resources[i].Name)

			continue
		}

		if _, err = h.keycloakApiClient.CreateResource(ctx, realmName, clientID, *resourceRepresentation); err != nil {
			return fmt.Errorf("failed to create resource: %w", err)
		}

		log.Info("Resource created", resourceLogKey, keycloakClient.Spec.Authorization.Resources[i].Name)
	}

	if err = h.deleteResources(ctx, existingResources, realmName, clientID); err != nil {
		return err
	}

	return nil
}

func (h *ProcessResources) deleteResources(ctx context.Context, existingResources map[string]gocloak.ResourceRepresentation, realmName string, clientID string) error {
	log := ctrl.LoggerFrom(ctx)

	for name := range existingResources {
		if name == "Default Resource" {
			continue
		}

		if err := h.keycloakApiClient.DeleteResource(ctx, realmName, clientID, *existingResources[name].ID); err != nil {
			if !adapter.IsErrNotFound(err) {
				return fmt.Errorf("failed to delete resource: %w", err)
			}
		}

		log.Info("Resource deleted", resourceLogKey, name)
	}

	return nil
}

// toResourceRepresentation converts keycloakApi.Resource to gocloak.ResourceRepresentation.
func (h *ProcessResources) toResourceRepresentation(ctx context.Context, resource *keycloakApi.Resource, clientID, realm string) (*gocloak.ResourceRepresentation, error) {
	keycloakResource := getBaseResourceRepresentation(resource)

	if err := h.mapScopes(ctx, resource, keycloakResource, realm, clientID); err != nil {
		return nil, fmt.Errorf("failed to map scopes: %w", err)
	}

	return keycloakResource, nil
}

func (h *ProcessResources) mapScopes(
	ctx context.Context,
	resource *keycloakApi.Resource,
	keycloakResource *gocloak.ResourceRepresentation,
	realm,
	clientID string,
) error {
	if len(resource.Scopes) == 0 {
		keycloakResource.Scopes = &[]gocloak.ScopeRepresentation{}

		return nil
	}

	existingScopes, err := h.keycloakApiClient.GetScopes(ctx, realm, clientID)
	if err != nil {
		return fmt.Errorf("failed to get scopes: %w", err)
	}

	resourceScopes := make([]gocloak.ScopeRepresentation, 0, len(resource.Scopes))

	for _, r := range resource.Scopes {
		existingScope, ok := existingScopes[r]
		if !ok {
			return fmt.Errorf("scope %s does not exist", r)
		}

		if existingScope.ID == nil {
			return fmt.Errorf("scope %s does not have ID", r)
		}

		resourceScopes = append(resourceScopes, existingScope)
	}

	keycloakResource.Scopes = &resourceScopes

	return nil
}

func getBaseResourceRepresentation(resource *keycloakApi.Resource) *gocloak.ResourceRepresentation {
	r := &gocloak.ResourceRepresentation{
		Name:               &resource.Name,
		DisplayName:        &resource.DisplayName,
		Type:               &resource.Type,
		IconURI:            &resource.IconURI,
		OwnerManagedAccess: &resource.OwnerManagedAccess,
	}

	uris := slices.Clone(resource.URIs)
	r.URIs = &uris

	attributes := make(map[string][]string, len(resource.Attributes))
	maps.Copy(attributes, resource.Attributes)

	return r
}
