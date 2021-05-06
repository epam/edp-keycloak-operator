package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	IsDefault   bool                `json:"isDefault"`
}

// +k8s:openapi-gen=true
type KeycloakRealmRoleBatchStatus struct {
	Value        string `json:"value"`
	FailureCount int64  `json:"failureCount"`
}

func (in KeycloakRealmRoleBatch) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmRoleBatch) SetStatus(value string) {
	in.Status.Value = value
}

func (in KeycloakRealmRoleBatch) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmRoleBatch) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakRealmRoleBatch) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
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

func (in *KeycloakRealmRoleBatch) FormattedRoleName(baseRoleName string) string {
	return fmt.Sprintf("%s-%s", in.Name, baseRoleName)
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
