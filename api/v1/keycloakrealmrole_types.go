package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
)

const StatusDuplicated = "duplicated"

// KeycloakRealmRoleSpec defines the desired state of KeycloakRealmRole.
type KeycloakRealmRoleSpec struct {
	// Name of keycloak role.
	Name string `json:"name"`

	// Deprecated: use RealmRef instead.
	// Realm is name of KeycloakRealm custom resource.
	// +optional
	Realm string `json:"realm"`

	// RealmRef is reference to Realm custom resource.
	// +optional
	RealmRef common.RealmRef `json:"realmRef"`

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

	// CompositesClientRoles is a map of composites client roles assigned to role.
	// +nullable
	// +optional
	// +kubebuilder:example={"client1": {{"name": "role1"}, {"name": "role2"}}, "client2": {"name": "role3"}}
	CompositesClientRoles map[string][]Composite `json:"compositesClientRoles,omitempty"`

	// IsDefault is a flag if role is default.
	// +optional
	IsDefault bool `json:"isDefault,omitempty"`
}

type Composite struct {
	// Name is a name of composite role.
	Name string `json:"name"`
}

// KeycloakRealmRoleStatus defines the observed state of KeycloakRealmRole.
type KeycloakRealmRoleStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// ID is a role ID.
	// +optional
	ID string `json:"id,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconcilation status"

// KeycloakRealmRole is the Schema for the keycloak group API.
type KeycloakRealmRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmRoleSpec   `json:"spec,omitempty"`
	Status KeycloakRealmRoleStatus `json:"status,omitempty"`
}

func (in *KeycloakRealmRole) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmRole) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakRealmRole) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmRole) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmRole) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

// +kubebuilder:object:root=true

// KeycloakRealmRoleList contains a list of KeycloakRealmRole.
type KeycloakRealmRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmRole{}, &KeycloakRealmRoleList{})
}
