package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// KeycloakComponentSpec defines the desired state of KeycloakRealmComponent.
type KeycloakComponentSpec struct {
	// Name of keycloak component.
	Name string `json:"name"`

	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// ProviderID is a provider ID of component.
	ProviderID string `json:"providerId"`

	// ProviderType is a provider type of component.
	ProviderType string `json:"providerType"`

	// ParentRef specifies a parent resource.
	// If not specified, then parent is realm specified in realm field.
	// +nullable
	// +optional
	ParentRef *ParentComponent `json:"parentRef,omitempty"`

	// Config is a map of component configuration.
	// Map key is a name of configuration property, map value is an array value of configuration properties.
	// Any configuration property can be a reference to k8s secret, in this case the property should be in format $secretName:secretKey.
	// +kubebuilder:example={"bindDn": ["provider-client"], "bindCredential": ["$clientSecret:secretKey"]}
	// +nullable
	// +optional
	Config map[string][]string `json:"config,omitempty"`
}

// KeycloakComponentStatus defines the observed state of KeycloakRealmComponent.
type KeycloakComponentStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// ParentComponent defines the parent component of KeycloakRealmComponent.
type ParentComponent struct {
	// Kind is a kind of parent component. By default, it is KeycloakRealm.
	// +optional
	// +kubebuilder:default=KeycloakRealm
	// +kubebuilder:validation:Enum=KeycloakRealm;KeycloakRealmComponent
	Kind string `json:"kind,omitempty"`

	// Name is a name of parent component custom resource.
	// For example, if Kind is KeycloakRealm, then Name is name of KeycloakRealm custom resource.
	Name string `json:"name"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconcilation status"

// KeycloakRealmComponent is the Schema for the keycloak component API.
type KeycloakRealmComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakComponentSpec   `json:"spec,omitempty"`
	Status KeycloakComponentStatus `json:"status,omitempty"`
}

func (in *KeycloakRealmComponent) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmComponent) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakRealmComponent) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmComponent) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmComponent) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

// +kubebuilder:object:root=true

// KeycloakRealmComponentList contains a list of KeycloakRealmComponent.
type KeycloakRealmComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmComponent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmComponent{}, &KeycloakRealmComponentList{})
}
