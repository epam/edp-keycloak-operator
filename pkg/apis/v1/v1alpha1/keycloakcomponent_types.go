package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakRealmComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KeycloakComponentSpec   `json:"spec"`
	Status            KeycloakComponentStatus `json:"status"`
}

type KeycloakComponentSpec struct {
	Name         string              `json:"name"`
	Realm        string              `json:"realm"`
	ProviderID   string              `json:"providerId"`
	ProviderType string              `json:"providerType"`
	Config       map[string][]string `json:"config"`
}

type KeycloakComponentStatus struct {
	Value        string `json:"value"`
	FailureCount int64  `json:"failureCount"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakRealmComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakRealmComponent `json:"items"`
}

func (in KeycloakRealmComponent) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmComponent) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in KeycloakRealmComponent) GetStatus() string {
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
