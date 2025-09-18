package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// KeycloakAuthFlowSpec defines the desired state of KeycloakAuthFlow
type KeycloakAuthFlowSpec struct {
	// Realm is name of keycloak realm
	Realm string `json:"realm"`

	// Alias is display name for authentication flow
	Alias string `json:"alias"`

	// +optional
	Description string `json:"description,omitempty"`

	// ProviderID for root auth flow and provider for child auth flows
	ProviderID string `json:"providerId"`

	TopLevel bool `json:"topLevel"`

	BuiltIn bool `json:"builtIn"`

	// +nullable
	// +optional
	AuthenticationExecutions []AuthenticationExecution `json:"authenticationExecutions,omitempty"`

	// +optional
	ParentName string `json:"parentName,omitempty"`

	// ChildType is type for auth flow if it has a parent, available options: basic-flow, form-flow
	// +optional
	ChildType string `json:"childType,omitempty"`
}

// AuthenticationExecution defines keycloak authentication execution
type AuthenticationExecution struct {
	// +optional
	Authenticator string `json:"authenticator,omitempty"`

	// +nullable
	// +optional
	AuthenticatorConfig *AuthenticatorConfig `json:"authenticatorConfig,omitempty"`

	// +optional
	AuthenticatorFlow bool `json:"authenticatorFlow,omitempty"`

	// +optional
	Priority int `json:"priority,omitempty"`

	// +optional
	Requirement string `json:"requirement,omitempty"`

	// +optional
	Alias string `json:"alias,omitempty"`
}

type AuthenticatorConfig struct {
	// +optional
	Alias string `json:"alias,omitempty"`

	// +optional
	// +nullable
	Config map[string]string `json:"config,omitempty"`
}

// KeycloakAuthFlowStatus defines the observed state of KeycloakAuthFlow
type KeycloakAuthFlowStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// KeycloakAuthFlow is the Schema for the keycloak authentication flow API
type KeycloakAuthFlow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakAuthFlowSpec   `json:"spec,omitempty"`
	Status KeycloakAuthFlowStatus `json:"status,omitempty"`
}

func (in *KeycloakAuthFlow) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
}

func (in KeycloakAuthFlow) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakAuthFlow) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in KeycloakAuthFlow) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakAuthFlow) SetStatus(value string) {
	in.Status.Value = value
}

// +kubebuilder:object:root=true

// KeycloakAuthFlowList contains a list of KeycloakAuthFlow
type KeycloakAuthFlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakAuthFlow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakAuthFlow{}, &KeycloakAuthFlowList{})
}
