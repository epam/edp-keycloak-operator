package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KeycloakSpec defines the desired state of Keycloak
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

func (in *KeycloakSpec) GetInstallMainRealm() bool {
	return in.InstallMainRealm == nil || *in.InstallMainRealm
}

type User struct {
	// Username of keycloak user
	Username string `json:"username"`

	// RealmRoles is a list of roles attached to keycloak user
	RealmRoles []string `json:"realmRoles,omitempty"`
}

// KeycloakStatus defines the observed state of Keycloak
type KeycloakStatus struct {
	// Connected shows if keycloak service is up and running
	Connected bool `json:"connected"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// Keycloak is the Schema for the keycloaks API
type Keycloak struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakSpec   `json:"spec,omitempty"`
	Status KeycloakStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeycloakList contains a list of Keycloak
type KeycloakList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Keycloak `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Keycloak{}, &KeycloakList{})
}
