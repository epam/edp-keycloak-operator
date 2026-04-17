package chain

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakapi "github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"k8s.io/apimachinery/pkg/types"
)

// CreateOrUpdateComponent creates or updates a realm component in Keycloak.
type CreateOrUpdateComponent struct {
	k8sClient       client.Client
	kClientV2       *keycloakapi.APIClient
	secretRefClient SecretRefClient
}

func NewCreateOrUpdateComponent(
	k8sClient client.Client,
	kClientV2 *keycloakapi.APIClient,
	secretRefClient SecretRefClient,
) *CreateOrUpdateComponent {
	return &CreateOrUpdateComponent{
		k8sClient:       k8sClient,
		kClientV2:       kClientV2,
		secretRefClient: secretRefClient,
	}
}

func (h *CreateOrUpdateComponent) Serve(
	ctx context.Context,
	component *keycloakApi.KeycloakRealmComponent,
	realmName string,
) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating or updating realm component")

	spec := component.Spec

	config := make(keycloakapi.MultivaluedHashMapStringString, len(spec.Config))

	for k, v := range spec.Config {
		copied := make([]string, len(v))
		copy(copied, v)
		config[k] = copied
	}

	if err := h.secretRefClient.MapComponentConfigSecretsRefs(ctx, config, component.Namespace); err != nil {
		return fmt.Errorf("unable to map config secrets: %w", err)
	}

	parentID, err := h.resolveParentID(ctx, component, realmName)
	if err != nil {
		return fmt.Errorf("unable to resolve parent ID: %w", err)
	}

	repr := keycloakapi.ComponentRepresentation{
		Name:         &spec.Name,
		ProviderId:   &spec.ProviderID,
		ProviderType: &spec.ProviderType,
		Config:       &config,
	}

	if parentID != "" {
		repr.ParentId = &parentID
	}

	existing, err := h.kClientV2.RealmComponents.FindComponentByName(ctx, realmName, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to find component by name: %w", err)
	}

	if existing == nil {
		resp, err := h.kClientV2.RealmComponents.CreateComponent(ctx, realmName, repr)
		if err != nil {
			return fmt.Errorf("failed to create realm component: %w", err)
		}

		component.Status.ID = keycloakapi.GetResourceIDFromResponse(resp)

		log.Info("Realm component created")

		return nil
	}

	if existing.Id != nil {
		component.Status.ID = *existing.Id
		repr.Id = existing.Id
	}

	if _, err := h.kClientV2.RealmComponents.UpdateComponent(ctx, realmName, component.Status.ID, repr); err != nil {
		return fmt.Errorf("failed to update realm component: %w", err)
	}

	log.Info("Realm component updated")

	return nil
}

func (h *CreateOrUpdateComponent) resolveParentID(
	ctx context.Context,
	component *keycloakApi.KeycloakRealmComponent,
	realmName string,
) (string, error) {
	if component.Spec.ParentRef == nil {
		return "", nil
	}

	switch component.Spec.ParentRef.Kind {
	case keycloakApi.KeycloakRealmKind:
		parentRealm := &keycloakApi.KeycloakRealm{}
		if err := h.k8sClient.Get(ctx, types.NamespacedName{
			Name:      component.Spec.ParentRef.Name,
			Namespace: component.Namespace,
		}, parentRealm); err != nil {
			return "", fmt.Errorf("unable to get parent realm: %w", err)
		}

		kcRealm, _, err := h.kClientV2.Realms.GetRealm(ctx, parentRealm.Spec.RealmName)
		if err != nil {
			return "", fmt.Errorf("unable to get parent realm from Keycloak: %w", err)
		}

		if kcRealm.Id == nil || *kcRealm.Id == "" {
			return "", fmt.Errorf("parent realm ID is empty")
		}

		return *kcRealm.Id, nil

	case keycloakApi.KeycloakRealmComponentKind:
		parentComponent, err := h.kClientV2.RealmComponents.FindComponentByName(ctx, realmName, component.Spec.ParentRef.Name)
		if err != nil {
			return "", fmt.Errorf("unable to find parent component: %w", err)
		}

		if parentComponent == nil {
			return "", fmt.Errorf("parent component %q not found", component.Spec.ParentRef.Name)
		}

		if parentComponent.Id == nil {
			return "", fmt.Errorf("parent component %q has no ID", component.Spec.ParentRef.Name)
		}

		return *parentComponent.Id, nil

	default:
		return "", fmt.Errorf("parent kind %s is not supported", component.Spec.ParentRef.Kind)
	}
}
