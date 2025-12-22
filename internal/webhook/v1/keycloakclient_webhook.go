package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

// log is for logging in this package.
var keycloakclientlog = logf.Log.WithName("keycloakclient-resource")

// SetupKeycloakClientWebhookWithManager registers the webhook for KeycloakClient in the manager.
func SetupKeycloakClientWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&keycloakApi.KeycloakClient{}).
		WithDefaulter(&KeycloakClientCustomDefaulter{}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-v1-edp-epam-com-v1-keycloakclient,mutating=true,failurePolicy=fail,sideEffects=None,groups=v1.edp.epam.com,resources=keycloakclients,verbs=create;update,versions=v1,name=mkeycloakclient-v1.kb.io,admissionReviewVersions=v1

// KeycloakClientCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind KeycloakClient when those are created or updated.
type KeycloakClientCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &KeycloakClientCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind KeycloakClient.
func (d *KeycloakClientCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	keycloakclient, ok := obj.(*keycloakApi.KeycloakClient)

	if !ok {
		return fmt.Errorf("expected a KeycloakClient object but got %T", obj)
	}

	keycloakclientlog.Info("Defaulting for KeycloakClient", "name", keycloakclient.GetName())

	updated := d.applyDefaults(keycloakclient)

	if updated {
		keycloakclientlog.Info("Applied defaults to KeycloakClient", "name", keycloakclient.GetName())
	}

	return nil
}

const (
	ClientAttributeLogoutRedirectUris         = "post.logout.redirect.uris"
	ClientAttributeLogoutRedirectUrisDefValue = "+"
)

// applyDefaults applies default values to KeycloakClient.
func (r *KeycloakClientCustomDefaulter) applyDefaults(keycloakClient *keycloakApi.KeycloakClient) bool {
	if keycloakClient.Spec.Attributes == nil {
		keycloakClient.Spec.Attributes = make(map[string]string)
	}

	updated := false

	if _, ok := keycloakClient.Spec.Attributes[ClientAttributeLogoutRedirectUris]; !ok {
		// set default value for logout redirect uris to "+" is required for correct logout from keycloak
		keycloakClient.Spec.Attributes[ClientAttributeLogoutRedirectUris] = ClientAttributeLogoutRedirectUrisDefValue
		updated = true
	}

	if keycloakClient.Spec.WebOrigins == nil && keycloakClient.Spec.WebUrl != "" {
		keycloakClient.Spec.WebOrigins = []string{
			keycloakClient.Spec.WebUrl,
		}

		updated = true
	}

	if migrated := r.migrateClientRoles(keycloakClient); migrated {
		updated = true
	}

	if keycloakClient.Spec.ServiceAccount != nil {
		if migrated := r.migrateServiceAccountAttributes(keycloakClient); migrated {
			updated = true
		}
	}

	return updated
}

// migrateClientRoles migrates ClientRoles to ClientRolesV2 format.
// This function converts the old string-based client roles to the new ClientRole struct format.
// It only performs migration if ClientRolesV2 is empty and ClientRoles is not empty.
func (r *KeycloakClientCustomDefaulter) migrateClientRoles(keycloakClient *keycloakApi.KeycloakClient) bool {
	if len(keycloakClient.Spec.ClientRolesV2) == 0 && len(keycloakClient.Spec.ClientRoles) > 0 {
		keycloakclientlog.Info("Migrating ClientRoles to ClientRolesV2",
			"name", keycloakClient.GetName(),
			"roleCount", len(keycloakClient.Spec.ClientRoles))

		// Convert string-based roles to ClientRole structs
		for _, roleName := range keycloakClient.Spec.ClientRoles {
			clientRole := keycloakApi.ClientRole{
				Name: roleName,
				// Composite field is left empty as it wasn't available in the old format
			}
			keycloakClient.Spec.ClientRolesV2 = append(keycloakClient.Spec.ClientRolesV2, clientRole)
		}

		// Keep the original ClientRoles field for backward compatibility
		// keycloakClient.Spec.ClientRoles remains unchanged

		return true
	}

	return false
}

// migrateServiceAccountAttributes migrates Attributes to AttributesV2 format.
// This function converts the old string-based attributes to the new []string format.
// It only performs migration if AttributesV2 is empty and Attributes is not empty.
func (r *KeycloakClientCustomDefaulter) migrateServiceAccountAttributes(keycloakClient *keycloakApi.KeycloakClient) bool {
	if len(keycloakClient.Spec.ServiceAccount.AttributesV2) == 0 && len(keycloakClient.Spec.ServiceAccount.Attributes) > 0 {
		keycloakclientlog.Info("Migrating ServiceAccount.Attributes to AttributesV2",
			"name", keycloakClient.GetName(),
			"attributeCount", len(keycloakClient.Spec.ServiceAccount.Attributes))

		keycloakClient.Spec.ServiceAccount.AttributesV2 = make(map[string][]string, len(keycloakClient.Spec.ServiceAccount.Attributes))

		// Convert string bases attributes to []string
		for attr, value := range keycloakClient.Spec.ServiceAccount.Attributes {
			keycloakClient.Spec.ServiceAccount.AttributesV2[attr] = []string{value}
		}

		// Keep the original Attributes field for backward compatibility
		// keycloakClient.Spec.ServiceAccount.Attributes remains unchanged

		return true
	}

	return false
}
