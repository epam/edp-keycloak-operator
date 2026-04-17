package chain

import (
	"context"
	"fmt"
	"maps"
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

const resourceLogKey = "resource"

type ProcessResources struct {
	kClient   *keycloakapi.APIClient
	k8sClient client.Client
}

func NewProcessResources(kClient *keycloakapi.APIClient, k8sClient client.Client) *ProcessResources {
	return &ProcessResources{kClient: kClient, k8sClient: k8sClient}
}

func (h *ProcessResources) Serve(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, realmName string, clientCtx *ClientContext) error {
	log := ctrl.LoggerFrom(ctx)

	if keycloakClient.Spec.Authorization == nil {
		log.Info("Authorization settings are not specified")
		return nil
	}

	clientUUID := clientCtx.ClientUUID

	resourcesList, _, err := h.kClient.Authorization.GetResources(ctx, realmName, clientUUID)
	if err != nil {
		h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization resources: %s", err.Error()))

		return fmt.Errorf("failed to get resources: %w", err)
	}

	existingResources := maputil.SliceToMapSelf(resourcesList, func(r keycloakapi.ResourceRepresentation) (string, bool) {
		return *r.Name, r.Name != nil
	})

	for i := 0; i < len(keycloakClient.Spec.Authorization.Resources); i++ {
		log.Info("Processing resource", resourceLogKey, keycloakClient.Spec.Authorization.Resources[i].Name)

		var resourceRepresentation keycloakapi.ResourceRepresentation

		if resourceRepresentation, err = h.toResourceRepresentation(ctx, &keycloakClient.Spec.Authorization.Resources[i], clientUUID, realmName); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization resources: %s", err.Error()))

			return fmt.Errorf("failed to convert resource: %w", err)
		}

		existingResource, ok := existingResources[keycloakClient.Spec.Authorization.Resources[i].Name]
		if ok {
			if existingResource.UnderscoreId == nil {
				h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization resources: resource %s has no ID", keycloakClient.Spec.Authorization.Resources[i].Name))

				return fmt.Errorf("existing resource %s has no ID", keycloakClient.Spec.Authorization.Resources[i].Name)
			}

			if _, err = h.kClient.Authorization.UpdateResource(ctx, realmName, clientUUID, *existingResource.UnderscoreId, resourceRepresentation); err != nil {
				h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization resources: %s", err.Error()))

				return fmt.Errorf("failed to update resource: %w", err)
			}

			log.Info("Resource updated", resourceLogKey, keycloakClient.Spec.Authorization.Resources[i].Name)

			delete(existingResources, keycloakClient.Spec.Authorization.Resources[i].Name)

			continue
		}

		if _, _, err = h.kClient.Authorization.CreateResource(ctx, realmName, clientUUID, resourceRepresentation); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization resources: %s", err.Error()))

			return fmt.Errorf("failed to create resource: %w", err)
		}

		log.Info("Resource created", resourceLogKey, keycloakClient.Spec.Authorization.Resources[i].Name)
	}

	if keycloakClient.Spec.ReconciliationStrategy != keycloakApi.ReconciliationStrategyAddOnly {
		if err = h.deleteResources(ctx, existingResources, realmName, clientUUID); err != nil {
			h.setFailureCondition(ctx, keycloakClient, fmt.Sprintf("Failed to sync authorization resources: %s", err.Error()))

			return err
		}
	}

	h.setSuccessCondition(ctx, keycloakClient, "Authorization resources synchronized")

	return nil
}

func (h *ProcessResources) setFailureCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAuthorizationResourcesSynced,
		metav1.ConditionFalse,
		ReasonKeycloakAPIError,
		message,
	); err != nil {
		log.Error(err, "Failed to set failure condition")
	}
}

func (h *ProcessResources) setSuccessCondition(ctx context.Context, keycloakClient *keycloakApi.KeycloakClient, message string) {
	log := ctrl.LoggerFrom(ctx)

	if err := SetCondition(
		ctx, h.k8sClient, keycloakClient,
		ConditionAuthorizationResourcesSynced,
		metav1.ConditionTrue,
		ReasonAuthorizationResourcesSynced,
		message,
	); err != nil {
		log.Error(err, "Failed to set success condition")
	}
}

func (h *ProcessResources) deleteResources(ctx context.Context, existingResources map[string]keycloakapi.ResourceRepresentation, realmName string, clientUUID string) error {
	log := ctrl.LoggerFrom(ctx)

	for name := range existingResources {
		if name == "Default Resource" {
			continue
		}

		r := existingResources[name]
		if r.UnderscoreId == nil {
			continue
		}

		if _, err := h.kClient.Authorization.DeleteResource(ctx, realmName, clientUUID, *r.UnderscoreId); err != nil {
			if !keycloakapi.IsNotFound(err) {
				return fmt.Errorf("failed to delete resource: %w", err)
			}
		}

		log.Info("Resource deleted", resourceLogKey, name)
	}

	return nil
}

// toResourceRepresentation converts keycloakApi.Resource to keycloakapi.ResourceRepresentation.
func (h *ProcessResources) toResourceRepresentation(ctx context.Context, resource *keycloakApi.Resource, clientUUID, realm string) (keycloakapi.ResourceRepresentation, error) {
	keycloakResource := getBaseResourceRepresentation(resource)

	if err := h.mapScopes(ctx, resource, &keycloakResource, realm, clientUUID); err != nil {
		return keycloakapi.ResourceRepresentation{}, fmt.Errorf("failed to map scopes: %w", err)
	}

	return keycloakResource, nil
}

func (h *ProcessResources) mapScopes(
	ctx context.Context,
	resource *keycloakApi.Resource,
	keycloakResource *keycloakapi.ResourceRepresentation,
	realm,
	clientUUID string,
) error {
	if len(resource.Scopes) == 0 {
		emptyScopes := []keycloakapi.ScopeRepresentation{}
		keycloakResource.Scopes = &emptyScopes

		return nil
	}

	scopesList, _, err := h.kClient.Authorization.GetScopes(ctx, realm, clientUUID)
	if err != nil {
		return fmt.Errorf("failed to get scopes: %w", err)
	}

	existingScopes := maputil.SliceToMapSelf(scopesList, func(s keycloakapi.ScopeRepresentation) (string, bool) {
		return *s.Name, s.Name != nil
	})

	resourceScopes := make([]keycloakapi.ScopeRepresentation, 0, len(resource.Scopes))

	for _, r := range resource.Scopes {
		existingScope, ok := existingScopes[r]
		if !ok {
			return fmt.Errorf("scope %s does not exist", r)
		}

		if existingScope.Id == nil {
			return fmt.Errorf("scope %s does not have ID", r)
		}

		resourceScopes = append(resourceScopes, existingScope)
	}

	keycloakResource.Scopes = &resourceScopes

	return nil
}

func getBaseResourceRepresentation(resource *keycloakApi.Resource) keycloakapi.ResourceRepresentation {
	r := keycloakapi.ResourceRepresentation{
		Name:               &resource.Name,
		DisplayName:        &resource.DisplayName,
		Type:               &resource.Type,
		IconUri:            &resource.IconURI,
		OwnerManagedAccess: &resource.OwnerManagedAccess,
	}

	uris := slices.Clone(resource.URIs)
	r.Uris = &uris

	attributes := make(map[string][]string, len(resource.Attributes))
	maps.Copy(attributes, resource.Attributes)

	if len(attributes) > 0 {
		r.Attributes = &attributes
	}

	return r
}
