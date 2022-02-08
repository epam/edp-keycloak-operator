package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type KeycloakAuthFlow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakAuthFlowSpec   `json:"spec,omitempty"`
	Status KeycloakAuthFlowStatus `json:"status,omitempty"`
}

// +k8s:openapi-gen=true
type KeycloakAuthFlowSpec struct {
	Realm                    string                    `json:"realm"` //realm name
	Alias                    string                    `json:"alias"`
	Description              string                    `json:"description"`
	ProviderID               string                    `json:"providerId"`
	TopLevel                 bool                      `json:"topLevel"`
	BuiltIn                  bool                      `json:"builtIn"`
	AuthenticationExecutions []AuthenticationExecution `json:"authenticationExecutions"`
	ParentName               string                    `json:"parentName"`
	ChildType                string                    `json:"childType"`
}

// +k8s:openapi-gen=true
type AuthenticationExecution struct {
	Authenticator       string               `json:"authenticator"`
	AuthenticatorConfig *AuthenticatorConfig `json:"authenticatorConfig"`
	AuthenticatorFlow   bool                 `json:"authenticatorFlow"`
	Priority            int                  `json:"priority"`
	Requirement         string               `json:"requirement"`
}

// +k8s:openapi-gen=true
type AuthenticatorConfig struct {
	Alias  string            `json:"alias"`
	Config map[string]string `json:"config"`
}

type KeycloakAuthFlowStatus struct {
	Value        string `json:"value"`
	FailureCount int64  `json:"failureCount"`
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakAuthFlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakAuthFlow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakAuthFlow{}, &KeycloakAuthFlowList{})
}
