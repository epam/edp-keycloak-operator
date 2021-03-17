package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:openapi-gen=true
type KeycloakRealmRoleBatchSpec struct {
	Realm string      `json:"realm"` //realm name
	Roles []BatchRole `json:"roles"`
}

// +k8s:openapi-gen=true
type BatchRole struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Attributes  map[string][]string `json:"attributes"`
	Composite   bool                `json:"composite"`
	Composites  []Composite         `json:"composites"`
}

// +k8s:openapi-gen=true
type KeycloakRealmRoleBatchStatus struct {
	Value string `json:"value"`
}


func (in *KeycloakRealmRoleBatch) K8SParentRealmName() string {
	return in.Spec.Realm
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KeycloakRealmRoleBatch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmRoleBatchSpec   `json:"spec,omitempty"`
	Status KeycloakRealmRoleBatchStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakRealmRoleBatchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakRealmRoleBatch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmRoleBatch{}, &KeycloakRealmRoleBatchList{})
}
