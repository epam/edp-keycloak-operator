package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KeycloakClientScope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakClientScopeSpec   `json:"spec,omitempty"`
	Status KeycloakClientScopeStatus `json:"status,omitempty"`
}

type KeycloakClientScopeStatus struct {
	ID           string `json:"id"`
	Value        string `json:"value"`
	FailureCount int64  `json:"failureCount"`
}

func (in *KeycloakClientScope) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
}

func (in KeycloakClientScope) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakClientScope) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in KeycloakClientScope) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakClientScope) SetStatus(value string) {
	in.Status.Value = value
}

type KeycloakClientScopeSpec struct {
	Name            string            `json:"name"`
	Realm           string            `json:"realm"`
	Description     string            `json:"description"`
	Protocol        string            `json:"protocol"`
	Attributes      map[string]string `json:"attributes"`
	Default         bool              `json:"default"`
	ProtocolMappers []ProtocolMapper  `json:"protocolMappers"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakClientScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakClientScope `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakClientScope{}, &KeycloakClientScopeList{})
}
