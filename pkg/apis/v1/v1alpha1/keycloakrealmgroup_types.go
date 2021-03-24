package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:openapi-gen=true
type KeycloakRealmGroupSpec struct {
	Name        string              `json:"name"`
	Realm       string              `json:"realm"`
	Path        string              `json:"path,omitempty"`
	Attributes  map[string][]string `json:"attributes,omitempty"`
	Access      map[string]bool     `json:"access,omitempty"`
	RealmRoles  []string            `json:"realmRoles,omitempty"`
	SubGroups   []string            `json:"subGroups,omitempty"`
	ClientRoles []ClientRole        `json:"clientRoles"`
}

// +k8s:openapi-gen=true
type KeycloakRealmGroupStatus struct {
	Value        string `json:"value"`
	ID           string `json:"id"`
	FailureCount int64  `json:"failureCount"`
}

func (in KeycloakRealmGroupStatus) GetFailureCount() int64 {
	return in.FailureCount
}

func (in *KeycloakRealmGroupStatus) SetFailureCount(count int64) {
	in.FailureCount = count
}

func (in *KeycloakRealmGroup) K8SParentRealmName() string {
	return in.Spec.Realm
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KeycloakRealmGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmGroupSpec   `json:"spec,omitempty"`
	Status KeycloakRealmGroupStatus `json:"status,omitempty"`
}

// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakRealmGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakRealmGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmGroup{}, &KeycloakRealmGroupList{})
}
