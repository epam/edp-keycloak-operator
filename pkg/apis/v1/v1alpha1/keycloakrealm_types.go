package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KeycloakRealmSpec defines the desired state of KeycloakRealm
type KeycloakRealmSpec struct {
	RealmName string `json:"realmName"`

	// +optional
	KeycloakOwner string `json:"keycloakOwner,omitempty"`

	// +optional
	SsoRealmName string `json:"ssoRealmName,omitempty"`

	// +nullable
	// +optional
	SsoRealmEnabled *bool `json:"ssoRealmEnabled,omitempty"`

	// +nullable
	// +optional
	SsoAutoRedirectEnabled *bool `json:"ssoAutoRedirectEnabled,omitempty"`

	// +nullable
	// +optional
	Users []User `json:"users,omitempty"`

	// +nullable
	// +optional
	SSORealmMappers *[]SSORealmMapper `json:"ssoRealmMappers,omitempty"`

	// +nullable
	// +optional
	BrowserFlow *string `json:"browserFlow,omitempty"`

	// +nullable
	// +optional
	Themes *RealmThemes `json:"themes,omitempty"`

	// +nullable
	// +optional
	BrowserSecurityHeaders *map[string]string `json:"browserSecurityHeaders,omitempty"`

	// +nullable
	// +optional
	ID *string `json:"id,omitempty"`

	// +nullable
	// +optional
	RealmEventConfig *RealmEventConfig `json:"realmEventConfig,omitempty"`

	// +optional
	DisableCentralIDPMappers bool `json:"disableCentralIDPMappers,omitempty"`

	// +nullable
	// +optional
	PasswordPolicies []PasswordPolicy `json:"passwordPolicy,omitempty"`
}

type PasswordPolicy struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type RealmEventConfig struct {
	// +optional
	AdminEventsDetailsEnabled bool `json:"adminEventsDetailsEnabled,omitempty"`

	// +optional
	AdminEventsEnabled bool `json:"adminEventsEnabled,omitempty"`

	// +optional
	// +nullable
	EnabledEventTypes []string `json:"enabledEventTypes,omitempty"`

	// +optional
	EventsEnabled bool `json:"eventsEnabled,omitempty"`

	// +optional
	EventsExpiration int `json:"eventsExpiration,omitempty"`

	// +optional
	// +nullable
	EventsListeners []string `json:"eventsListeners,omitempty"`
}

type RealmThemes struct {
	// +nullable
	// +optional
	LoginTheme *string `json:"loginTheme"`

	// +nullable
	// +optional
	AccountTheme *string `json:"accountTheme"`

	// +nullable
	// +optional
	AdminConsoleTheme *string `json:"adminConsoleTheme"`

	// +nullable
	// +optional
	EmailTheme *string `json:"emailTheme"`

	// +nullable
	// +optional
	InternationalizationEnabled *bool `json:"internationalizationEnabled"`
}

func (in *KeycloakRealmSpec) SSOEnabled() bool {
	return in.SsoRealmEnabled == nil || *in.SsoRealmEnabled
}

func (in *KeycloakRealmSpec) SSOAutoRedirectEnabled() bool {
	return in.SsoAutoRedirectEnabled == nil || *in.SsoAutoRedirectEnabled
}

type SSORealmMapper struct {
	// +optional
	IdentityProviderMapper string `json:"identityProviderMapper,omitempty"`

	// +optional
	Name string `json:"name,omitempty"`

	// +nullable
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

// KeycloakRealmStatus defines the observed state of KeycloakRealm
type KeycloakRealmStatus struct {
	// +optional
	Available bool `json:"available,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`

	// +optional
	Value string `json:"value,omitempty"`
}

func (in *KeycloakRealm) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealm) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion

// KeycloakRealm is the Schema for the keycloakrealms API
type KeycloakRealm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmSpec   `json:"spec,omitempty"`
	Status KeycloakRealmStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeycloakRealmList contains a list of KeycloakRealm
type KeycloakRealmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealm{}, &KeycloakRealmList{})
}
