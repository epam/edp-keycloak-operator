package chain

import (
	"context"
	"fmt"

	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

// CreateOrUpdateComponent creates or updates a realm component in Keycloak.
type CreateOrUpdateComponent struct {
	kClient         *keycloakapi.KeycloakClient
	secretRefClient SecretRefClient
	parentID        string
}

func NewCreateOrUpdateComponent(
	kClient *keycloakapi.KeycloakClient,
	secretRefClient SecretRefClient,
	parentID string,
) *CreateOrUpdateComponent {
	return &CreateOrUpdateComponent{
		kClient:         kClient,
		secretRefClient: secretRefClient,
		parentID:        parentID,
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
	rawCfg := spec.Config

	config := make(keycloakapi.MultivaluedHashMapStringString, len(spec.Config))

	for k, v := range spec.Config {
		copied := make([]string, len(v))
		copy(copied, v)
		config[k] = copied
	}

	secretVersions, err := h.secretRefClient.MapComponentConfigSecretsRefs(ctx, "spec.config", config, component.Namespace)
	if err != nil {
		return fmt.Errorf("unable to map config secrets: %w", err)
	}

	newHash := secretref.ValuesHash(secretVersions)

	repr := keycloakapi.ComponentRepresentation{
		Name:         &spec.Name,
		ProviderId:   &spec.ProviderID,
		ProviderType: &spec.ProviderType,
		Config:       &config,
		ParentId:     ptr.To(h.parentID),
	}

	existing, err := h.findExisting(ctx, component, realmName)
	if err != nil {
		return err
	}

	if existing == nil {
		resp, err := h.kClient.RealmComponents.CreateComponent(ctx, realmName, repr)
		if err != nil {
			return fmt.Errorf("failed to create realm component: %w", err)
		}

		component.Status.ID = keycloakapi.GetResourceIDFromResponse(resp)
		if component.Status.ID == "" {
			return fmt.Errorf("realm component %q created but Location header missing or empty", spec.Name)
		}

		log.Info("Realm component created")
	} else {
		if existing.Id != nil {
			component.Status.ID = *existing.Id
			repr.Id = existing.Id
		}

		needsUpdate := helper.SpecChanged(component.Generation, component.Status.ObservedGeneration) ||
			newHash != component.Status.ConfigSecretsHash ||
			!componentMatchesSpec(existing, repr, rawCfg)

		if needsUpdate {
			if _, err := h.kClient.RealmComponents.UpdateComponent(ctx, realmName, component.Status.ID, repr); err != nil {
				return fmt.Errorf("failed to update realm component: %w", err)
			}

			log.Info("Realm component updated")
		}
	}

	// Stamped even when the update is skipped; if the controller's status write fails, the
	// generation check forces the next reconcile to re-run.
	component.Status.ConfigSecretsHash = newHash

	return nil
}

// componentMatchesSpec reports whether the fetched component already matches the desired
// representation.
func componentMatchesSpec(
	existing *keycloakapi.ComponentRepresentation,
	desired keycloakapi.ComponentRepresentation,
	rawCfg map[string][]string,
) bool {
	if ptr.Deref(existing.Name, "") != ptr.Deref(desired.Name, "") ||
		ptr.Deref(existing.ProviderId, "") != ptr.Deref(desired.ProviderId, "") ||
		ptr.Deref(existing.ProviderType, "") != ptr.Deref(desired.ProviderType, "") {
		return false
	}

	if ptr.Deref(existing.ParentId, "") != ptr.Deref(desired.ParentId, "") {
		return false
	}

	return secretref.ConfigMatchesMulti(ptr.Deref(existing.Config, nil), ptr.Deref(desired.Config, nil), rawCfg)
}

// findExisting returns the Keycloak component the CR manages, or nil if none exists.
// status.id wins when it still exists, whatever its current name, parent or providerType.
// Otherwise the component is matched by exact spec.name and spec.providerType under the resolved parent.
func (h *CreateOrUpdateComponent) findExisting(
	ctx context.Context,
	component *keycloakApi.KeycloakRealmComponent,
	realmName string,
) (*keycloakapi.ComponentRepresentation, error) {
	if id := component.Status.ID; id != "" {
		existing, _, err := h.kClient.RealmComponents.GetComponent(ctx, realmName, id)
		if err != nil && !keycloakapi.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get component %s: %w", id, err)
		}

		if err == nil && existing != nil {
			return existing, nil
		}
	}

	return findComponentUnderParent(ctx, h.kClient, realmName, component.Spec.Name, h.parentID, component.Spec.ProviderType)
}

// findComponentUnderParent returns the component with exactly name and providerType under parentID, or nil.
// Keycloak stores top-level components with the realm ID as parentId, so the realm ID selects them.
// Component names are not unique in Keycloak; providerType separates components of different types.
func findComponentUnderParent(
	ctx context.Context,
	kClient *keycloakapi.KeycloakClient,
	realmName, name, parentID, providerType string,
) (*keycloakapi.ComponentRepresentation, error) {
	components, _, err := kClient.RealmComponents.GetComponents(ctx, realmName, &keycloakapi.GetComponentsParams{
		Name:   &name,
		Parent: &parentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find component %q under parent %s: %w", name, parentID, err)
	}

	for i := range components {
		if ptr.Deref(components[i].Name, "") == name &&
			ptr.Deref(components[i].ProviderType, "") == providerType {
			return &components[i], nil
		}
	}

	return nil, nil
}
