package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KeycloakClientSpec defines the desired state of KeycloakClient
// +k8s:openapi-gen=true
type KeycloakClientSpec struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	TargetRealm  string `json:"targetRealm"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// KeycloakClientStatus defines the observed state of KeycloakClient
// +k8s:openapi-gen=true
type KeycloakClientStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KeycloakClient is the Schema for the keycloakclients API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KeycloakClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakClientSpec   `json:"spec,omitempty"`
	Status KeycloakClientStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KeycloakClientList contains a list of KeycloakClient
type KeycloakClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakClient{}, &KeycloakClientList{})
}
