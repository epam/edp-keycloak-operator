package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type KeycloakRealmGroupSpec struct {
	Name  string `json:"name"`
	Realm string `json:"realm"`

	// +optional
	Path string `json:"path,omitempty"`

	// +nullable
	// +optional
	Attributes map[string][]string `json:"attributes,omitempty"`

	// +nullable
	// +optional
	Access map[string]bool `json:"access,omitempty"`

	// +nullable
	// +optional
	RealmRoles []string `json:"realmRoles,omitempty"`

	// +nullable
	// +optional
	SubGroups []string `json:"subGroups,omitempty"`

	// +nullable
	// +optional
	ClientRoles []ClientRole `json:"clientRoles"`
}

type KeycloakRealmGroupStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

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

func (in *KeycloakRealmGroup) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion
type KeycloakRealmGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmGroupSpec   `json:"spec,omitempty"`
	Status KeycloakRealmGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KeycloakRealmGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmGroup{}, &KeycloakRealmGroupList{})
}
