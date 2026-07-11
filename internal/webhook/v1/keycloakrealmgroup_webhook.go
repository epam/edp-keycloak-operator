package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	keycloakAlpha "github.com/epam/edp-keycloak-operator/api/v1alpha1"
)

// keycloakrealmgrouplog is for logging in this package.
var keycloakrealmgrouplog = logf.Log.WithName("keycloakrealmgroup-resource")

// SetupKeycloakRealmGroupWebhookWithManager registers the webhook for KeycloakRealmGroup in the manager.
func SetupKeycloakRealmGroupWebhookWithManager(mgr ctrl.Manager, k8sClient client.Client) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&keycloakApi.KeycloakRealmGroup{}).
		WithValidator(NewKeycloakRealmGroupCustomValidator(k8sClient)).
		Complete()
}

// +kubebuilder:webhook:path=/validate-v1-edp-epam-com-v1-keycloakrealmgroup,mutating=false,failurePolicy=fail,sideEffects=None,groups=v1.edp.epam.com,resources=keycloakrealmgroups,verbs=create;update,versions=v1,name=vkeycloakrealmgroup-v1.kb.io,admissionReviewVersions=v1

// KeycloakRealmGroupCustomValidator validates the KeycloakRealmGroup resource on create and update.
type KeycloakRealmGroupCustomValidator struct {
	k8sclient client.Client
}

func NewKeycloakRealmGroupCustomValidator(k8sClient client.Client) *KeycloakRealmGroupCustomValidator {
	return &KeycloakRealmGroupCustomValidator{
		k8sclient: k8sClient,
	}
}

var _ webhook.CustomValidator = &KeycloakRealmGroupCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealmGroup.
func (v *KeycloakRealmGroupCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	group, ok := obj.(*keycloakApi.KeycloakRealmGroup)
	if !ok {
		return nil, fmt.Errorf("expected a KeycloakRealmGroup object but got %T", obj)
	}

	keycloakrealmgrouplog.Info("Validation for KeycloakRealmGroup upon creation", "name", group.GetName())

	return nil, v.validateGroupUniqueness(ctx, group)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealmGroup.
// Unlike name fields on other resources, spec.name and spec.parentGroup are mutable,
// so an update can move the group into a colliding slot and must be re-validated.
func (v *KeycloakRealmGroupCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	group, ok := newObj.(*keycloakApi.KeycloakRealmGroup)
	if !ok {
		return nil, fmt.Errorf("expected a KeycloakRealmGroup object but got %T", newObj)
	}

	oldGroup, ok := oldObj.(*keycloakApi.KeycloakRealmGroup)
	if !ok {
		return nil, fmt.Errorf("expected a KeycloakRealmGroup object but got %T", oldObj)
	}

	keycloakrealmgrouplog.Info("Validation for KeycloakRealmGroup upon update", "name", group.GetName())

	// Only re-validate when the update changes the group's identity. Identity-preserving
	// updates (finalizer removal, labels, other spec fields) must not be blocked: otherwise
	// duplicates that predate this webhook could never be reconciled or even deleted.
	if oldGroup.Spec.Name == group.Spec.Name && sameGroupTarget(oldGroup, group) {
		return nil, nil
	}

	return nil, v.validateGroupUniqueness(ctx, group)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealmGroup.
func (v *KeycloakRealmGroupCustomValidator) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// validateGroupUniqueness checks that no other KeycloakRealmGroup resource resolves to the
// same Keycloak group. A group's identity is (resolved realm, parent group, name): a top-level
// group is identified by its name within the realm, a child group by its name under its parent.
// Without this check two resources would silently adopt the same Keycloak group ID and
// fight over its children on every reconciliation.
func (v *KeycloakRealmGroupCustomValidator) validateGroupUniqueness(
	ctx context.Context,
	group *keycloakApi.KeycloakRealmGroup,
) error {
	existingGroups := &keycloakApi.KeycloakRealmGroupList{}
	if err := v.k8sclient.List(ctx, existingGroups); err != nil {
		return fmt.Errorf("failed to list KeycloakRealmGroup resources: %w", err)
	}

	for i := range existingGroups.Items {
		existing := &existingGroups.Items[i]

		isSameResource := existing.Namespace == group.Namespace && existing.Name == group.Name

		if existing.Spec.Name == group.Spec.Name && sameGroupTarget(existing, group) && !isSameResource {
			return fmt.Errorf(
				"group name %q is already used by KeycloakRealmGroup %s/%s (%s in realm %s)",
				group.Spec.Name,
				existing.Namespace,
				existing.Name,
				formatGroupPlacement(group),
				formatResolvedRealm(group),
			)
		}
	}

	return nil
}

func sameGroupTarget(a, b *keycloakApi.KeycloakRealmGroup) bool {
	if !sameRealmInstance(a, b) {
		return false
	}

	aParent, bParent := a.Spec.ParentGroup, b.Spec.ParentGroup

	if (aParent == nil) != (bParent == nil) {
		return false
	}

	if aParent == nil {
		return true
	}

	// ParentGroup is a namespace-local CR reference, so the same parent name in
	// different namespaces refers to two distinct parent groups.
	return aParent.Name == bParent.Name && a.Namespace == b.Namespace
}

// sameRealmInstance reports whether two KeycloakRealmGroup resources target the same realm.
//
// The namespaced KeycloakRealm kind is resolved within the resource's own namespace, so its
// identity is (namespace, name). The cluster-scoped ClusterKeycloakRealm kind is resolved
// cluster-wide, so its identity is the name alone.
func sameRealmInstance(a, b *keycloakApi.KeycloakRealmGroup) bool {
	if normalizeRealmKind(a.Spec.RealmRef.Kind) != normalizeRealmKind(b.Spec.RealmRef.Kind) ||
		a.Spec.RealmRef.Name != b.Spec.RealmRef.Name {
		return false
	}

	if normalizeRealmKind(b.Spec.RealmRef.Kind) == keycloakApi.KeycloakRealmKind {
		return a.Namespace == b.Namespace
	}

	return true
}

// normalizeRealmKind maps an unset RealmRef.Kind to its CRD default so that resources
// created before structural defaulting are compared consistently.
func normalizeRealmKind(kind string) string {
	if kind == "" {
		return keycloakApi.KeycloakRealmKind
	}

	return kind
}

func formatGroupPlacement(group *keycloakApi.KeycloakRealmGroup) string {
	if group.Spec.ParentGroup != nil {
		return fmt.Sprintf("under parent group %q", group.Spec.ParentGroup.Name)
	}

	return "top-level"
}

// formatResolvedRealm returns a human-readable identity of the realm a KeycloakRealmGroup
// targets: Kind/namespace/name for the namespaced KeycloakRealm kind, Kind/name for
// the cluster-scoped ClusterKeycloakRealm kind.
func formatResolvedRealm(group *keycloakApi.KeycloakRealmGroup) string {
	kind := normalizeRealmKind(group.Spec.RealmRef.Kind)
	if kind == keycloakAlpha.ClusterKeycloakRealmKind {
		return fmt.Sprintf("%s/%s", kind, group.Spec.RealmRef.Name)
	}

	return fmt.Sprintf("%s/%s/%s", kind, group.Namespace, group.Spec.RealmRef.Name)
}
