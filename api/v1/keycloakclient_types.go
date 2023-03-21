package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ReconciliationStrategyFull    = "full"
	ReconciliationStrategyAddOnly = "addOnly"
	// ClientSecretKey is a key for client secret in secret data.
	ClientSecretKey = "clientSecret"
)

// KeycloakClientSpec defines the desired state of KeycloakClient.
type KeycloakClientSpec struct {
	// ClientId is a unique keycloak client ID referenced in URI and tokens.
	ClientId string `json:"clientId"`

	// +optional
	TargetRealm string `json:"targetRealm,omitempty"`

	// +optional
	Secret string `json:"secret,omitempty"`

	// +nullable
	// +optional
	RealmRoles *[]RealmRole `json:"realmRoles,omitempty"`

	// +optional
	Public bool `json:"public,omitempty"`

	// +optional
	WebUrl string `json:"webUrl,omitempty"`

	// +nullable
	// +optional
	Protocol *string `json:"protocol,omitempty"`

	// +nullable
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`

	// +optional
	DirectAccess bool `json:"directAccess,omitempty"`

	// +optional
	AdvancedProtocolMappers bool `json:"advancedProtocolMappers,omitempty"`

	// +nullable
	// +optional
	ClientRoles []string `json:"clientRoles,omitempty"`

	// +nullable
	// +optional
	ProtocolMappers *[]ProtocolMapper `json:"protocolMappers,omitempty"`

	// +nullable
	// +optional
	ServiceAccount *ServiceAccount `json:"serviceAccount,omitempty"`

	// +optional
	FrontChannelLogout bool `json:"frontChannelLogout,omitempty"`

	// +kubebuilder:validation:Enum=full;addOnly
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`

	// A list of default client scopes for a keycloak client.
	// +nullable
	// +optional
	DefaultClientScopes []string `json:"defaultClientScopes,omitempty"`
}

type ServiceAccount struct {
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// +nullable
	// +optional
	RealmRoles []string `json:"realmRoles"`

	// +nullable
	// +optional
	ClientRoles []ClientRole `json:"clientRoles,omitempty"`

	// +nullable
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
}

type ClientRole struct {
	ClientID string `json:"clientId"`

	// +nullable
	// +optional
	Roles []string `json:"roles,omitempty"`
}

type ProtocolMapper struct {
	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	Protocol string `json:"protocol,omitempty"`

	// +optional
	ProtocolMapper string `json:"protocolMapper,omitempty"`

	// +nullable
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type RealmRole struct {
	// +optional
	Name string `json:"name,omitempty"`

	Composite string `json:"composite"`
}

// KeycloakClientStatus defines the observed state of KeycloakClient.
type KeycloakClientStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	ClientID string `json:"clientId,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`

	// +optional
	ClientSecretName string `json:"clientSecretName,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// KeycloakClient is the Schema for the keycloak clients API.
type KeycloakClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakClientSpec   `json:"spec,omitempty"`
	Status KeycloakClientStatus `json:"status,omitempty"`
}

func (in *KeycloakClient) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakClient) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakClient) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakClient) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakClient) GetReconciliationStrategy() string {
	if in.Spec.ReconciliationStrategy == "" {
		return ReconciliationStrategyFull
	}

	return in.Spec.ReconciliationStrategy
}

// +kubebuilder:object:root=true

// KeycloakClientList contains a list of KeycloakClient.
type KeycloakClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakClient{}, &KeycloakClientList{})
}
