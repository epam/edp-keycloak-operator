package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// KeycloakClientScopeSpec defines the desired state of KeycloakClientScope.
type KeycloakClientScopeSpec struct {
	// Name of keycloak client scope
	Name string `json:"name"`

	// Realm is name of keycloak realm
	Realm string `json:"realm"`

	// Protocol is SSO protocol configuration which is being supplied by this client scope
	Protocol string `json:"protocol"`

	// +optional
	Description string `json:"description,omitempty"`

	// +nullable
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`

	// +optional
	Default bool `json:"default,omitempty"`

	// +nullable
	// +optional
	ProtocolMappers []ProtocolMapper `json:"protocolMappers,omitempty"`
}

// KeycloakClientScopeStatus defines the observed state of KeycloakClientScope.
type KeycloakClientScopeStatus struct {
	// +optional
	ID string `json:"id,omitempty"`

	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// KeycloakClientScope is the Schema for the keycloakclientscopes API.
type KeycloakClientScope struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakClientScopeSpec   `json:"spec,omitempty"`
	Status KeycloakClientScopeStatus `json:"status,omitempty"`
}

func (in *KeycloakClientScope) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
}

func (in *KeycloakClientScope) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakClientScope) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakClientScope) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakClientScope) SetStatus(value string) {
	in.Status.Value = value
}

// +kubebuilder:object:root=true

// KeycloakClientScopeList contains a list of KeycloakClientScope.
type KeycloakClientScopeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakClientScope `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakClientScope{}, &KeycloakClientScopeList{})
}
