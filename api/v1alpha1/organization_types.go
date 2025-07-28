package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// KeycloakOrganizationSpec defines the desired state of Organization.
type KeycloakOrganizationSpec struct {
	// Name is the unique name of the organization.
	// The name should be unique across Organizations.
	// +required
	Name string `json:"name"`

	// Alias is the unique alias for the organization.
	// The alias should be unique across Organizations.
	// +required
	Alias string `json:"alias"`

	// Domains is a list of email domains associated with the organization.
	// Each domain should be unique across Organizations.
	// +required
	// +kubebuilder:validation:MinItems=1
	Domains []string `json:"domains"`

	// RedirectURL is the optional redirect URL for the organization.
	// +optional
	RedirectURL string `json:"redirectUrl,omitempty"`

	// Description is an optional description of the organization.
	// +optional
	Description string `json:"description,omitempty"`

	// Attributes is a map of custom attributes for the organization.
	// +optional
	// +nullable
	Attributes map[string][]string `json:"attributes,omitempty"`

	// IdentityProviders is a list of identity providers associated with the organization.
	// One identity provider can't be assigned to multiple organizations.
	// +optional
	// +nullable
	IdentityProviders []OrgIdentityProvider `json:"identityProviders,omitempty"`

	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`
}

// OrgIdentityProvider defines an identity provider for an organization.
type OrgIdentityProvider struct {
	// Alias is the unique identifier for the identity provider within the organization.
	// +required
	Alias string `json:"alias"`
}

// KeycloakOrganizationStatus defines the observed state of Organization.
type KeycloakOrganizationStatus struct {
	// Value contains the current reconciliation status.
	// +optional
	Value string `json:"value,omitempty"`

	// OrganizationID is the unique identifier of the organization in Keycloak.
	// +optional
	OrganizationID string `json:"organizationId,omitempty"`

	// Error is the error message if the reconciliation failed.
	// +optional
	Error string `json:"error,omitempty"`
}

func (in *KeycloakOrganizationStatus) SetOK() {
	in.Value = common.StatusOK
	in.Error = ""
}

func (in *KeycloakOrganizationStatus) SetError(err string) {
	in.Value = common.StatusError
	in.Error = err
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconciliation status"
// +kubebuilder:printcolumn:name="Organization ID",type="string",JSONPath=".status.organizationId",description="Keycloak organization ID"
// +kubebuilder:printcolumn:name="Realm",type="string",JSONPath=".spec.realmName",description="Keycloak realm name"
// +kubebuilder:printcolumn:name="Keycloak",type="string",JSONPath=".spec.keycloakRef.name",description="Keycloak instance name"

// KeycloakOrganization is the Schema for the organizations API.
type KeycloakOrganization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakOrganizationSpec   `json:"spec,omitempty"`
	Status KeycloakOrganizationStatus `json:"status,omitempty"`
}

func (in *KeycloakOrganization) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

// +kubebuilder:object:root=true

// KeycloakOrganizationList contains a list of KeycloakOrganization.
type KeycloakOrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakOrganization `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakOrganization{}, &KeycloakOrganizationList{})
}
