package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KeycloakSpec defines the desired state of Keycloak.
type KeycloakSpec struct {
	// URL of keycloak service
	Url string `json:"url"`

	// Secret is the name of the k8s object Secret related to keycloak
	Secret string `json:"secret"`

	// +optional
	RealmName string `json:"realmName,omitempty"`

	// +optional
	SsoRealmName string `json:"ssoRealmName,omitempty"`

	// Users is a list of keycloak users
	// +nullable
	// +optional
	Users []User `json:"users,omitempty"`

	// +nullable
	// +optional
	InstallMainRealm *bool `json:"installMainRealm,omitempty"`

	// AdminType can be user or serviceAccount, if serviceAccount was specified, then client_credentials grant type should be used for getting admin realm token
	// +optional
	// +kubebuilder:validation:Enum=serviceAccount;user
	AdminType string `json:"adminType,omitempty"`
}

const (
	KeycloakAdminTypeUser           = "user"
	KeycloakAdminTypeServiceAccount = "serviceAccount"
)

func (in *Keycloak) GetAdminType() string {
	if in.Spec.AdminType == "" {
		in.Spec.AdminType = KeycloakAdminTypeUser
	}

	return in.Spec.AdminType
}

// KeycloakStatus defines the observed state of Keycloak.
type KeycloakStatus struct {
	Connected bool `json:"connected"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion

// Keycloak is the Schema for the keycloaks API.
type Keycloak struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakSpec   `json:"spec,omitempty"`
	Status KeycloakStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeycloakList contains a list of Keycloak.
type KeycloakList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Keycloak `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Keycloak{}, &KeycloakList{})
}
