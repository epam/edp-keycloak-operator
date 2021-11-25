package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakRealmIdentityProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KeycloakRealmIdentityProviderSpec   `json:"spec"`
	Status            KeycloakRealmIdentityProviderStatus `json:"status"`
}

type KeycloakRealmIdentityProviderSpec struct {
	Realm                     string                   `json:"realm"`
	ProviderID                string                   `json:"providerId"`
	Config                    map[string]string        `json:"config"`
	AddReadTokenRoleOnCreate  bool                     `json:"addReadTokenRoleOnCreate"`
	Alias                     string                   `json:"alias"`
	AuthenticateByDefault     bool                     `json:"authenticateByDefault"`
	DisplayName               string                   `json:"displayName"`
	Enabled                   bool                     `json:"enabled"`
	FirstBrokerLoginFlowAlias string                   `json:"firstBrokerLoginFlowAlias"`
	LinkOnly                  bool                     `json:"linkOnly"`
	StoreToken                bool                     `json:"storeToken"`
	TrustEmail                bool                     `json:"trustEmail"`
	Mappers                   []IdentityProviderMapper `json:"mappers"`
}

type IdentityProviderMapper struct {
	IdentityProviderAlias  string            `json:"identityProviderAlias"`
	IdentityProviderMapper string            `json:"identityProviderMapper"`
	Name                   string            `json:"name"`
	Config                 map[string]string `json:"config"`
}

type KeycloakRealmIdentityProviderStatus struct {
	Value        string `json:"value"`
	FailureCount int64  `json:"failureCount"`
}

func (in KeycloakRealmIdentityProvider) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmIdentityProvider) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in KeycloakRealmIdentityProvider) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmIdentityProvider) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmIdentityProvider) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakRealmIdentityProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakRealmIdentityProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmIdentityProvider{}, &KeycloakRealmIdentityProviderList{})
}
