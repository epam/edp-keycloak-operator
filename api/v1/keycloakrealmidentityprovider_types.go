package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// KeycloakRealmIdentityProviderSpec defines the desired state of KeycloakRealmIdentityProvider.
type KeycloakRealmIdentityProviderSpec struct {
	// Deprecated: use RealmRef instead.
	// Realm is name of KeycloakRealm custom resource.
	// +optional
	Realm string `json:"realm"`

	// RealmRef is reference to Realm custom resource.
	// +optional
	RealmRef common.RealmRef `json:"realmRef"`

	// ProviderID is a provider ID of identity provider.
	ProviderID string `json:"providerId"`

	// Alias is a alias of identity provider.
	Alias string `json:"alias"`

	// Config is a map of identity provider configuration.
	// Map key is a name of configuration property, map value is a value of configuration property.
	// Any value can be a reference to k8s secret, in this case value should be in format $secretName:secretKey.
	// +kubebuilder:example={"clientId": "provider-client", "clientSecret": "$clientSecret:secretKey"}
	Config map[string]string `json:"config"`

	// Enabled is a flag to enable/disable identity provider.
	Enabled bool `json:"enabled"`

	// AddReadTokenRoleOnCreate is a flag to add read token role on create.
	// +optional
	AddReadTokenRoleOnCreate bool `json:"addReadTokenRoleOnCreate,omitempty"`

	// AuthenticateByDefault is a flag to authenticate by default.
	// +optional
	AuthenticateByDefault bool `json:"authenticateByDefault,omitempty"`

	// DisplayName is a display name of identity provider.
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// FirstBrokerLoginFlowAlias is a first broker login flow alias.
	// +optional
	FirstBrokerLoginFlowAlias string `json:"firstBrokerLoginFlowAlias,omitempty"`

	// LinkOnly is a flag to link only.
	// +optional
	LinkOnly bool `json:"linkOnly,omitempty"`

	// StoreToken is a flag to store token.
	// +optional
	StoreToken bool `json:"storeToken,omitempty"`

	// TrustEmail is a flag to trust email.
	// +optional
	TrustEmail bool `json:"trustEmail,omitempty"`

	// Mappers is a list of identity provider mappers.
	// +nullable
	// +optional
	Mappers []IdentityProviderMapper `json:"mappers,omitempty"`
}

type IdentityProviderMapper struct {
	// IdentityProviderAlias is a identity provider alias.
	// +optional
	IdentityProviderAlias string `json:"identityProviderAlias,omitempty"`

	// IdentityProviderMapper is a identity provider mapper.
	// +optional
	IdentityProviderMapper string `json:"identityProviderMapper,omitempty"`

	// Name is a name of identity provider mapper.
	// +optional
	Name string `json:"name,omitempty"`

	// Config is a map of identity provider mapper configuration.
	// +nullable
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

// KeycloakRealmIdentityProviderStatus defines the observed state of KeycloakRealmIdentityProvider.
type KeycloakRealmIdentityProviderStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// KeycloakRealmIdentityProvider is the Schema for the keycloak realm identity provider API.
type KeycloakRealmIdentityProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmIdentityProviderSpec   `json:"spec,omitempty"`
	Status KeycloakRealmIdentityProviderStatus `json:"status,omitempty"`
}

func (in *KeycloakRealmIdentityProvider) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmIdentityProvider) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakRealmIdentityProvider) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmIdentityProvider) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmIdentityProvider) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

// +kubebuilder:object:root=true

// KeycloakRealmIdentityProviderList contains a list of KeycloakRealmIdentityProvider.
type KeycloakRealmIdentityProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmIdentityProvider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmIdentityProvider{}, &KeycloakRealmIdentityProviderList{})
}
