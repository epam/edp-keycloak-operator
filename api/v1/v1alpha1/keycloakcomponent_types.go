package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion
type KeycloakRealmComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakComponentSpec   `json:"spec,omitempty"`
	Status KeycloakComponentStatus `json:"status,omitempty"`
}

type KeycloakComponentSpec struct {
	Name         string `json:"name"`
	Realm        string `json:"realm"`
	ProviderID   string `json:"providerId"`
	ProviderType string `json:"providerType"`

	// +nullable
	// +optional
	Config map[string][]string `json:"config,omitempty"`
}

type KeycloakComponentStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// +kubebuilder:object:root=true
type KeycloakRealmComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmComponent `json:"items"`
}

func (in *KeycloakRealmComponent) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmComponent) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakRealmComponent) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmComponent) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmComponent) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmComponent{}, &KeycloakRealmComponentList{})
}
