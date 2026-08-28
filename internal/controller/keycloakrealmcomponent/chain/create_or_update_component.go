package chain

import (
	"context"
	"fmt"
	"slices"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
	"github.com/epam/edp-keycloak-operator/pkg/secretref"
)

// CreateOrUpdateComponent creates or updates a realm component in Keycloak.
type CreateOrUpdateComponent struct {
	k8sClient       client.Client
	kClient         *keycloakapi.KeycloakClient
	secretRefClient SecretRefClient
}

func NewCreateOrUpdateComponent(
	k8sClient client.Client,
	kClient *keycloakapi.KeycloakClient,
	secretRefClient SecretRefClient,
) *CreateOrUpdateComponent {
	return &CreateOrUpdateComponent{
		k8sClient:       k8sClient,
		kClient:         kClient,
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
	rawCfg := spec.Config

	config := make(keycloakapi.MultivaluedHashMapStringString, len(spec.Config))

	for k, v := range spec.Config {
		copied := make([]string, len(v))
		copy(copied, v)
		config[k] = copied
	}

	if err := h.secretRefClient.MapComponentConfigSecretsRefs(ctx, config, component.Namespace); err != nil {
		return fmt.Errorf("unable to map config secrets: %w", err)
	}

	newHash := secretref.ConfigSecretsHash(rawCfg, config)

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

	existing, err := h.kClient.RealmComponents.FindComponentByName(ctx, realmName, spec.Name)
	if err != nil {
		return fmt.Errorf("failed to find component by name: %w", err)
	}

	if existing == nil {
		resp, err := h.kClient.RealmComponents.CreateComponent(ctx, realmName, repr)
		if err != nil {
			return fmt.Errorf("failed to create realm component: %w", err)
		}

		component.Status.ID = keycloakapi.GetResourceIDFromResponse(resp)

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
// representation. Config keys are excluded from comparison when Keycloak masks them: either
// the raw spec value is a secret ref, or the fetched value is already the mask sentinel.
// ParentId is compared only when the operator manages it (desired.ParentId set): Keycloak
// auto-fills it with the realm's internal ID for components created without a parentRef.
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

	if desired.ParentId != nil && ptr.Deref(existing.ParentId, "") != *desired.ParentId {
		return false
	}

	existingConfig := ptr.Deref(existing.Config, nil)
	desiredConfig := ptr.Deref(desired.Config, nil)

	comparableConfig := make(map[string][]string, len(desiredConfig))

	for k, v := range desiredConfig {
		if secretref.HasAnySecretRef(rawCfg[k]) || slices.Contains(existingConfig[k], keycloakapi.MaskedSecretValue) {
			continue
		}

		comparableConfig[k] = v
	}

	return maputil.ContainsSubsetMulti(existingConfig, comparableConfig)
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

		kcRealm, _, err := h.kClient.Realms.GetRealm(ctx, parentRealm.Spec.RealmName)
		if err != nil {
			return "", fmt.Errorf("unable to get parent realm from Keycloak: %w", err)
		}

		if kcRealm.Id == nil || *kcRealm.Id == "" {
			return "", fmt.Errorf("parent realm ID is empty")
		}

		return *kcRealm.Id, nil

	case keycloakApi.KeycloakRealmComponentKind:
		parentComponent, err := h.kClient.RealmComponents.FindComponentByName(ctx, realmName, component.Spec.ParentRef.Name)
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
