package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReconciliationStrategyFull    = "full"
	ReconciliationStrategyAddOnly = "addOnly"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KeycloakClientSpec defines the desired state of KeycloakClient
// +k8s:openapi-gen=true
type KeycloakClientSpec struct {
	TargetRealm             string            `json:"targetRealm"`
	Secret                  string            `json:"secret"`
	RealmRoles              *[]RealmRole      `json:"realmRoles,omitempty"`
	Public                  bool              `json:"public"`
	ClientId                string            `json:"clientId"`
	WebUrl                  string            `json:"webUrl"`
	Protocol                *string           `json:"protocol,omitempty"`
	Attributes              map[string]string `json:"attributes,omitempty"`
	DirectAccess            bool              `json:"directAccess"`
	AdvancedProtocolMappers bool              `json:"advancedProtocolMappers"`
	ClientRoles             []string          `json:"clientRoles,omitempty"`
	ProtocolMappers         *[]ProtocolMapper `json:"protocolMappers,omitempty"`
	ServiceAccount          *ServiceAccount   `json:"serviceAccount,omitempty"`
	FrontChannelLogout      bool              `json:"frontChannelLogout,omitempty"`
	ReconciliationStrategy  string            `json:"reconciliationStrategy,omitempty"`
	// A list of default client scopes for a keycloak client.
	// +nullable
	// +optional
	DefaultClientScopes []string `json:"defaultClientScopes,omitempty"`
}

func (in KeycloakClient) GetReconciliationStrategy() string {
	if in.Spec.ReconciliationStrategy == "" {
		return ReconciliationStrategyFull
	}

	return in.Spec.ReconciliationStrategy
}

// +k8s:openapi-gen=true
type ServiceAccount struct {
	Enabled     bool              `json:"enabled"`
	RealmRoles  []string          `json:"realmRoles"`
	ClientRoles []ClientRole      `json:"clientRoles"`
	Attributes  map[string]string `json:"attributes"`
}

// +k8s:openapi-gen=true
type ClientRole struct {
	ClientID string   `json:"clientId"`
	Roles    []string `json:"roles"`
}

type ProtocolMapper struct {
	Name           string            `json:"name"`
	Protocol       string            `json:"protocol"`
	ProtocolMapper string            `json:"protocolMapper"`
	Config         map[string]string `json:"config,omitempty"`
}

type RealmRole struct {
	Name      string `json:"name"`
	Composite string `json:"composite"`
}

// KeycloakClientStatus defines the observed state of KeycloakClient
// +k8s:openapi-gen=true
type KeycloakClientStatus struct {
	Value            string `json:"value"`
	ClientID         string `json:"clientId"`
	FailureCount     int64  `json:"failureCount"`
	ClientSecretName string `json:"clientSecretName"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

func (in KeycloakClient) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakClient) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in KeycloakClient) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakClient) SetStatus(value string) {
	in.Status.Value = value
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
