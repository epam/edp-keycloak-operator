package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion
type KeycloakRealmIdentityProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmIdentityProviderSpec   `json:"spec"`
	Status KeycloakRealmIdentityProviderStatus `json:"status"`
}

type KeycloakRealmIdentityProviderSpec struct {
	Realm      string            `json:"realm"`
	ProviderID string            `json:"providerId"`
	Alias      string            `json:"alias"`
	Config     map[string]string `json:"config"`
	Enabled    bool              `json:"enabled"`

	// +optional
	AddReadTokenRoleOnCreate bool `json:"addReadTokenRoleOnCreate,omitempty"`

	// +optional
	AuthenticateByDefault bool `json:"authenticateByDefault,omitempty"`

	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// +optional
	FirstBrokerLoginFlowAlias string `json:"firstBrokerLoginFlowAlias,omitempty"`

	// +optional
	LinkOnly bool `json:"linkOnly,omitempty"`

	// +optional
	StoreToken bool `json:"storeToken,omitempty"`

	// +optional
	TrustEmail bool `json:"trustEmail,omitempty"`

	// +nullable
	// +optional
	Mappers []IdentityProviderMapper `json:"mappers,omitempty"`
}

type IdentityProviderMapper struct {
	// +optional
	IdentityProviderAlias string `json:"identityProviderAlias,omitempty"`

	// +optional
	IdentityProviderMapper string `json:"identityProviderMapper,omitempty"`

	// +optional
	Name string `json:"name,omitempty"`

	// +nullable
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type KeycloakRealmIdentityProviderStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
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

// +kubebuilder:object:root=true
type KeycloakRealmIdentityProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmIdentityProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmIdentityProvider{}, &KeycloakRealmIdentityProviderList{})
}
