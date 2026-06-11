package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// keycloakrealmlog is for logging in this package.
var keycloakrealmlog = logf.Log.WithName("keycloakrealm-resource")

// SetupKeycloakRealmWebhookWithManager registers the webhook for KeycloakRealm in the manager.
func SetupKeycloakRealmWebhookWithManager(mgr ctrl.Manager, k8sClient client.Client) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&keycloakApi.KeycloakRealm{}).
		WithValidator(NewKeycloakRealmCustomValidator(k8sClient)).
		Complete()
}

// +kubebuilder:webhook:path=/validate-v1-edp-epam-com-v1-keycloakrealm,mutating=false,failurePolicy=fail,sideEffects=None,groups=v1.edp.epam.com,resources=keycloakrealms,verbs=create;update,versions=v1,name=vkeycloakrealm-v1.kb.io,admissionReviewVersions=v1

// KeycloakRealmCustomValidator struct is responsible for validating the KeycloakRealm resource
// when it is created, updated, or deleted.
type KeycloakRealmCustomValidator struct {
	k8sclient client.Client
}

func NewKeycloakRealmCustomValidator(k8sclient client.Client) *KeycloakRealmCustomValidator {
	return &KeycloakRealmCustomValidator{
		k8sclient: k8sclient,
	}
}

// duplicateInternationalizationWarning returns an admission warning when both deprecated
// spec.themes.internationalizationEnabled and canonical spec.localization.internationalizationEnabled are set.
//
//nolint:staticcheck // intentional comparison with deprecated themes.internationalizationEnabled for this warning
func duplicateInternationalizationWarning(realm *keycloakApi.KeycloakRealm) admission.Warnings {
	t := realm.Spec.Themes
	l := realm.Spec.Localization

	if t == nil || l == nil {
		return nil
	}

	if t.InternationalizationEnabled == nil || l.InternationalizationEnabled == nil {
		return nil
	}

	return admission.Warnings{
		"Both spec.themes.internationalizationEnabled and spec.localization.internationalizationEnabled are set; " +
			"spec.localization wins (canonical). spec.themes.internationalizationEnabled is deprecated — remove it.",
	}
}

var _ webhook.CustomValidator = &KeycloakRealmCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealm.
func (v *KeycloakRealmCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	keycloakrealm, ok := obj.(*keycloakApi.KeycloakRealm)
	if !ok {
		return nil, fmt.Errorf("expected a KeycloakRealm object but got %T", obj)
	}

	keycloakrealmlog.Info("Validation for KeycloakRealm upon creation", "name", keycloakrealm.GetName())

	// Check that the combination of RealmName and the resolved Keycloak instance is
	// unique across KeycloakRealm resources. Uniqueness must be enforced per target
	// Keycloak instance, not per KeycloakRef name string.
	existingKeycloakRealms := &keycloakApi.KeycloakRealmList{}
	if err := v.k8sclient.List(ctx, existingKeycloakRealms); err != nil {
		return nil, fmt.Errorf("failed to list KeycloakRealm resources: %w", err)
	}

	for _, existingRealm := range existingKeycloakRealms.Items {
		isSameResource := existingRealm.Namespace == keycloakrealm.Namespace && existingRealm.Name == keycloakrealm.Name

		if existingRealm.Spec.RealmName == keycloakrealm.Spec.RealmName &&
			sameKeycloakInstance(&existingRealm, keycloakrealm) && !isSameResource {
			return nil, fmt.Errorf(
				"realm name %q is already used by KeycloakRealm %s/%s targeting %s",
				keycloakrealm.Spec.RealmName,
				existingRealm.Namespace,
				existingRealm.Name,
				formatResolvedKeycloakInstance(keycloakrealm),
			)
		}
	}

	return duplicateInternationalizationWarning(keycloakrealm), nil
}

// formatResolvedKeycloakInstance returns a human-readable identity for the Keycloak
// instance resolved from a KeycloakRealm reference. For the namespaced Keycloak kind
// this includes the realm namespace (Kind/namespace/name); for ClusterKeycloak it is
// cluster-scoped (Kind/name).
func formatResolvedKeycloakInstance(realm *keycloakApi.KeycloakRealm) string {
	ref := realm.Spec.KeycloakRef
	if ref.Kind == keycloakApi.KeycloakKind {
		return fmt.Sprintf("%s/%s/%s", ref.Kind, realm.Namespace, ref.Name)
	}

	return fmt.Sprintf("%s/%s", ref.Kind, ref.Name)
}

// sameKeycloakInstance reports whether two KeycloakRealm resources target the same
// Keycloak instance.
//
// The namespaced Keycloak kind is resolved within the realm's own namespace, so its
// instance identity is (namespace, name): the same KeycloakRef name in two different
// namespaces refers to two distinct Keycloak instances. The cluster-scoped
// ClusterKeycloak kind is resolved cluster-wide, so its identity is the name alone.
func sameKeycloakInstance(a, b *keycloakApi.KeycloakRealm) bool {
	if a.Spec.KeycloakRef.Kind != b.Spec.KeycloakRef.Kind ||
		a.Spec.KeycloakRef.Name != b.Spec.KeycloakRef.Name {
		return false
	}

	if b.Spec.KeycloakRef.Kind == keycloakApi.KeycloakKind {
		return a.Namespace == b.Namespace
	}

	return true
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealm.
func (v *KeycloakRealmCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	keycloakrealm, ok := newObj.(*keycloakApi.KeycloakRealm)
	if !ok {
		return nil, fmt.Errorf("expected a KeycloakRealm object but got %T", newObj)
	}

	return duplicateInternationalizationWarning(keycloakrealm), nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealm.
func (v *KeycloakRealmCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
