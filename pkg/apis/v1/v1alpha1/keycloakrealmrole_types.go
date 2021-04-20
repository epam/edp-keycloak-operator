package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:openapi-gen=true
type KeycloakRealmRoleSpec struct {
	Name        string              `json:"name"`
	Realm       string              `json:"realm"` //realm name
	Description string              `json:"description"`
	Attributes  map[string][]string `json:"attributes"`
	Composite   bool                `json:"composite"`
	Composites  []Composite         `json:"composites"`
	IsDefault   bool                `json:"isDefault,omitempty"`
}

// +k8s:openapi-gen=true
type Composite struct {
	Name string `json:"name"`
}

// +k8s:openapi-gen=true
type KeycloakRealmRoleStatus struct {
	Value        string `json:"value"`
	ID           string `json:"id"`
	FailureCount int64  `json:"failureCount"`
}

func (in KeycloakRealmRole) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmRole) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in KeycloakRealmRole) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmRole) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmRole) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KeycloakRealmRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmRoleSpec   `json:"spec,omitempty"`
	Status KeycloakRealmRoleStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakRealmRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakRealmRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmRole{}, &KeycloakRealmRoleList{})
}
