package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KeycloakRealmRoleBatchSpec defines the desired state of KeycloakRealmRoleBatch.
type KeycloakRealmRoleBatchSpec struct {
	// Realm is name of KeycloakRealm custom resource.
	Realm string `json:"realm"`

	// Roles is a list of roles to be created.
	Roles []BatchRole `json:"roles"`
}

type BatchRole struct {
	// Name of keycloak role.
	Name string `json:"name"`

	// Description is a role description.
	// +optional
	Description string `json:"description,omitempty"`

	// Attributes is a map of role attributes.
	// +nullable
	// +optional
	Attributes map[string][]string `json:"attributes,omitempty"`

	// Composite is a flag if role is composite.
	// +optional
	Composite bool `json:"composite,omitempty"`

	// Composites is a list of composites roles assigned to role.
	// +nullable
	// +optional
	Composites []Composite `json:"composites,omitempty"`

	// IsDefault is a flag if role is default.
	// +optional
	IsDefault bool `json:"isDefault,omitempty"`
}

// KeycloakRealmRoleBatchStatus defines the observed state of KeycloakRealmRoleBatch.
type KeycloakRealmRoleBatchStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// KeycloakRealmRoleBatch is the Schema for the keycloak roles API.
type KeycloakRealmRoleBatch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmRoleBatchSpec   `json:"spec,omitempty"`
	Status KeycloakRealmRoleBatchStatus `json:"status,omitempty"`
}

func (in *KeycloakRealmRoleBatch) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmRoleBatch) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmRoleBatch) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmRoleBatch) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakRealmRoleBatch) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
}

func (in *KeycloakRealmRoleBatch) FormattedRoleName(baseRoleName string) string {
	return fmt.Sprintf("%s-%s", in.Name, baseRoleName)
}

// +kubebuilder:object:root=true

// KeycloakRealmRoleBatchList contains a list of KeycloakRealmRoleBatch.
type KeycloakRealmRoleBatchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmRoleBatch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmRoleBatch{}, &KeycloakRealmRoleBatchList{})
}
