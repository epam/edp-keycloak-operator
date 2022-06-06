package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const StatusDuplicated = "duplicated"

type KeycloakRealmRoleSpec struct {
	Name  string `json:"name"`
	Realm string `json:"realm"`

	// +optional
	Description string `json:"description,omitempty"`

	// +nullable
	// +optional
	Attributes map[string][]string `json:"attributes,omitempty"`

	// +optional
	Composite bool `json:"composite,omitempty"`

	// +nullable
	// +optional
	Composites []Composite `json:"composites,omitempty"`

	// +optional
	IsDefault bool `json:"isDefault,omitempty"`
}

type Composite struct {
	Name string `json:"name"`
}

type KeycloakRealmRoleStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	ID string `json:"id,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion
type KeycloakRealmRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmRoleSpec   `json:"spec,omitempty"`
	Status KeycloakRealmRoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KeycloakRealmRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmRole `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmRole{}, &KeycloakRealmRoleList{})
}
