package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// KeycloakRealmGroupSpec defines the desired state of KeycloakRealmGroup.
type KeycloakRealmGroupSpec struct {
	// Name of keycloak group.
	Name string `json:"name"`

	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// Path is a group path.
	// +optional
	Path string `json:"path,omitempty"`

	// Attributes is a map of group attributes.
	// +nullable
	// +optional
	Attributes map[string][]string `json:"attributes,omitempty"`

	// Access is a map of group access.
	// +nullable
	// +optional
	Access map[string]bool `json:"access,omitempty"`

	// RealmRoles is a list of realm roles assigned to group.
	// +nullable
	// +optional
	RealmRoles []string `json:"realmRoles,omitempty"`

	// SubGroups is a list of subgroups assigned to group.
	// +nullable
	// +optional
	SubGroups []string `json:"subGroups,omitempty"`

	// ClientRoles is a list of client roles assigned to group.
	// +nullable
	// +optional
	ClientRoles []UserClientRole `json:"clientRoles,omitempty"`
}

// KeycloakRealmGroupStatus defines the observed state of KeycloakRealmGroup.
type KeycloakRealmGroupStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// ID is a group ID.
	// +optional
	ID string `json:"id,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

func (in *KeycloakRealmGroup) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmGroup) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakRealmGroup) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmGroup) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmGroup) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconciliation status"

// KeycloakRealmGroup is the Schema for the keycloak group API.
type KeycloakRealmGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmGroupSpec   `json:"spec,omitempty"`
	Status KeycloakRealmGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeycloakRealmGroupList contains a list of KeycloakRealmGroup.
type KeycloakRealmGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmGroup{}, &KeycloakRealmGroupList{})
}
