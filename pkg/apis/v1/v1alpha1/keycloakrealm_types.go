package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KeycloakRealmSpec defines the desired state of KeycloakRealm
// +k8s:openapi-gen=true
type KeycloakRealmSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	RealmName              string            `json:"realmName"`
	KeycloakOwner          string            `json:"keycloakOwner,omitempty"`
	SsoRealmName           string            `json:"ssoRealmName,omitempty"`
	SsoRealmEnabled        *bool             `json:"ssoRealmEnabled,omitempty"` // default (nil, not set) must be true
	SsoAutoRedirectEnabled *bool             `json:"ssoAutoRedirectEnabled,omitempty"`
	Users                  []User            `json:"users,omitempty"`
	SSORealmMappers        *[]SSORealmMapper `json:"ssoRealmMappers,omitempty"`
}

func (in KeycloakRealmSpec) SSOEnabled() bool {
	return in.SsoRealmEnabled == nil || *in.SsoRealmEnabled
}

func (in KeycloakRealmSpec) SSOAutoRedirectEnabled() bool {
	return in.SsoAutoRedirectEnabled == nil || *in.SsoAutoRedirectEnabled
}

type SSORealmMapper struct {
	IdentityProviderMapper string            `json:"identityProviderMapper"`
	Name                   string            `json:"name"`
	Config                 map[string]string `json:"config"`
}

// KeycloakRealmStatus defines the observed state of KeycloakRealm
// +k8s:openapi-gen=true
type KeycloakRealmStatus struct {
	Available    bool  `json:"available,omitempty"`
	FailureCount int64 `json:"failureCount"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

func (in KeycloakRealm) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealm) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KeycloakRealm is the Schema for the keycloakrealms API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KeycloakRealm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmSpec   `json:"spec,omitempty"`
	Status KeycloakRealmStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KeycloakRealmList contains a list of KeycloakRealm
type KeycloakRealmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakRealm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealm{}, &KeycloakRealmList{})
}
