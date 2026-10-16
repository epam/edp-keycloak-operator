package chain

import (
	"context"
	"errors"
	"fmt"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
)

// ErrParentNotReady marks a parent that is expected to appear: the parent CR is missing, has
// no status.id yet, or its component is not in Keycloak yet. Callers requeue instead of failing.
var ErrParentNotReady = errors.New("parent not ready")

// maxParentDepth bounds the parentRef chain walk. Keycloak component trees are two or three
// levels deep (provider, mapper); anything longer is treated as a misconfiguration.
const maxParentDepth = 10

// ResolvedParent is the Keycloak parent a component is attached to.
type ResolvedParent struct {
	// ID is the Keycloak parent ID: the realm internal ID for a realm parent,
	// the parent component ID for a component parent.
	ID string
	// CR is the parent KeycloakRealmComponent; nil when the parent is the realm.
	CR *keycloakApi.KeycloakRealmComponent
}

// ResolveParent resolves spec.parentRef of component to its Keycloak parent.
// An absent parentRef resolves to the component's own realm.
// A KeycloakRealm parentRef must name spec.realmRef.
// A KeycloakRealmComponent parentRef names the parent CR in the component's namespace;
// the parent must target the same realm, have status.id set, and exist in Keycloak.
func ResolveParent(
	ctx context.Context,
	k8sClient client.Client,
	kClient *keycloakapi.KeycloakClient,
	component *keycloakApi.KeycloakRealmComponent,
	realmName string,
) (ResolvedParent, error) {
	ref := component.Spec.ParentRef
	if ref == nil {
		return resolveRealmParent(ctx, kClient, realmName)
	}

	switch parentRefKind(ref) {
	case keycloakApi.KeycloakRealmKind:
		if ref.Name != component.Spec.RealmRef.Name {
			return ResolvedParent{}, fmt.Errorf(
				"parentRef %q does not match spec.realmRef %q: a component can only be attached to its own realm",
				ref.Name, component.Spec.RealmRef.Name,
			)
		}

		return resolveRealmParent(ctx, kClient, realmName)

	case keycloakApi.KeycloakRealmComponentKind:
		return resolveComponentParent(ctx, k8sClient, kClient, component, realmName)

	default:
		return ResolvedParent{}, fmt.Errorf("parent kind %s is not supported", ref.Kind)
	}
}

// parentRefKind returns the parentRef kind; an empty kind is the CRD default KeycloakRealm.
func parentRefKind(ref *keycloakApi.ParentComponent) string {
	if ref.Kind == "" {
		return keycloakApi.KeycloakRealmKind
	}

	return ref.Kind
}

func resolveRealmParent(
	ctx context.Context,
	kClient *keycloakapi.KeycloakClient,
	realmName string,
) (ResolvedParent, error) {
	kcRealm, _, err := kClient.Realms.GetRealm(ctx, realmName)
	if err != nil {
		return ResolvedParent{}, fmt.Errorf("unable to get realm %q from Keycloak: %w", realmName, err)
	}

	if kcRealm == nil || kcRealm.Id == nil || *kcRealm.Id == "" {
		return ResolvedParent{}, fmt.Errorf("realm %q has no ID in Keycloak", realmName)
	}

	return ResolvedParent{ID: *kcRealm.Id}, nil
}

func resolveComponentParent(
	ctx context.Context,
	k8sClient client.Client,
	kClient *keycloakapi.KeycloakClient,
	component *keycloakApi.KeycloakRealmComponent,
	realmName string,
) (ResolvedParent, error) {
	name := component.Spec.ParentRef.Name

	if name == component.Name {
		return ResolvedParent{}, fmt.Errorf("parentRef %q must not reference the component itself", name)
	}

	parent := &keycloakApi.KeycloakRealmComponent{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: component.Namespace}, parent); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ResolvedParent{}, fmt.Errorf("%w: parent component %q not found", ErrParentNotReady, name)
		}

		return ResolvedParent{}, fmt.Errorf("unable to get parent component %q: %w", name, err)
	}

	if parent.DeletionTimestamp != nil {
		return ResolvedParent{}, fmt.Errorf("parent component %q is being deleted", name)
	}

	if err := rejectParentCycle(ctx, k8sClient, component, parent); err != nil {
		return ResolvedParent{}, err
	}

	sameRealm, err := helper.SameRealmTarget(ctx, k8sClient, component, parent)
	if err != nil {
		return ResolvedParent{}, fmt.Errorf("unable to compare realms of parent component %q: %w", name, err)
	}

	if !sameRealm {
		return ResolvedParent{}, fmt.Errorf("parent component %q belongs to a different realm", name)
	}

	if parent.Status.ID == "" {
		return ResolvedParent{}, fmt.Errorf("%w: parent component %q has no ID in status", ErrParentNotReady, name)
	}

	kcParent, _, err := kClient.RealmComponents.GetComponent(ctx, realmName, parent.Status.ID)
	if err != nil && !keycloakapi.IsNotFound(err) {
		return ResolvedParent{}, fmt.Errorf("unable to get parent component %q from Keycloak: %w", name, err)
	}

	if err != nil || kcParent == nil {
		return ResolvedParent{}, fmt.Errorf(
			"%w: parent component %q (ID %s) not found in Keycloak", ErrParentNotReady, name, parent.Status.ID,
		)
	}

	return ResolvedParent{ID: parent.Status.ID, CR: parent}, nil
}

// rejectParentCycle follows parentRef from parent upwards and fails when the chain returns to
// component or to a CR already visited, or exceeds maxParentDepth. A missing ancestor ends the
// walk: that ancestor's own reconcile reports it.
func rejectParentCycle(
	ctx context.Context,
	k8sClient client.Client,
	component *keycloakApi.KeycloakRealmComponent,
	parent *keycloakApi.KeycloakRealmComponent,
) error {
	seen := map[string]struct{}{component.Name: {}}
	current := parent

	for range maxParentDepth {
		if _, visited := seen[current.Name]; visited {
			return fmt.Errorf("parentRef %q forms a cycle through %q", component.Spec.ParentRef.Name, current.Name)
		}

		seen[current.Name] = struct{}{}

		ref := current.Spec.ParentRef
		if ref == nil || parentRefKind(ref) != keycloakApi.KeycloakRealmComponentKind {
			return nil
		}

		next := &keycloakApi.KeycloakRealmComponent{}
		if err := k8sClient.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: current.Namespace}, next); err != nil {
			if k8sErrors.IsNotFound(err) {
				return nil
			}

			return fmt.Errorf("unable to follow parentRef of %q: %w", current.Name, err)
		}

		current = next
	}

	return fmt.Errorf("parentRef %q chain exceeds %d levels", component.Spec.ParentRef.Name, maxParentDepth)
}
