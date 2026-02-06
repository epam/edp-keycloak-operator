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

// +kubebuilder:webhook:path=/validate-v1-edp-epam-com-v1-keycloakrealm,mutating=false,failurePolicy=fail,sideEffects=None,groups=v1.edp.epam.com,resources=keycloakrealms,verbs=create,versions=v1,name=vkeycloakrealm-v1.kb.io,admissionReviewVersions=v1

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

var _ webhook.CustomValidator = &KeycloakRealmCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealm.
func (v *KeycloakRealmCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	keycloakrealm, ok := obj.(*keycloakApi.KeycloakRealm)
	if !ok {
		return nil, fmt.Errorf("expected a KeycloakRealm object but got %T", obj)
	}

	keycloakrealmlog.Info("Validation for KeycloakRealm upon creation", "name", keycloakrealm.GetName())

	// Check if the combination of RealmName and KeycloakRef is unique across all KeycloakRealm resources in the cluster.
	existingKeycloakRealms := &keycloakApi.KeycloakRealmList{}
	if err := v.k8sclient.List(ctx, existingKeycloakRealms); err != nil {
		return nil, fmt.Errorf("failed to list KeycloakRealm resources: %w", err)
	}

	for _, existingRealm := range existingKeycloakRealms.Items {
		isSameResource := existingRealm.Namespace == keycloakrealm.Namespace && existingRealm.Name == keycloakrealm.Name

		isSameKeycloakInstance := existingRealm.Spec.KeycloakRef.Kind == keycloakrealm.Spec.KeycloakRef.Kind &&
			existingRealm.Spec.KeycloakRef.Name == keycloakrealm.Spec.KeycloakRef.Name

		if existingRealm.Spec.RealmName == keycloakrealm.Spec.RealmName && isSameKeycloakInstance && !isSameResource {
			return nil, fmt.Errorf(
				"realm name %s is already in use by another KeycloakRealm resource (%s/%s) for Keycloak instance %s/%s",
				keycloakrealm.Spec.RealmName,
				existingRealm.Namespace,
				existingRealm.Name,
				keycloakrealm.Spec.KeycloakRef.Kind,
				keycloakrealm.Spec.KeycloakRef.Name,
			)
		}
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealm.
func (v *KeycloakRealmCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type KeycloakRealm.
func (v *KeycloakRealmCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
