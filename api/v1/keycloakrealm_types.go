package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// KeycloakRealmSpec defines the desired state of KeycloakRealm.
type KeycloakRealmSpec struct {
	// RealmName specifies the name of the realm.
	RealmName string `json:"realmName"`

	// Deprecated: use KeycloakRef instead.
	// KeycloakOwner specifies the name of the Keycloak instance that owns the realm.
	// +nullable
	// +optional
	KeycloakOwner string `json:"keycloakOwner,omitempty"`

	// KeycloakRef is reference to Keycloak custom resource.
	// +optional
	KeycloakRef common.KeycloakRef `json:"keycloakRef,omitempty"`

	// Users is a list of users to create in the realm.
	// +nullable
	// +optional
	Users []User `json:"users,omitempty"`

	// BrowserFlow specifies the authentication flow to use for the realm's browser clients.
	// +nullable
	// +optional
	BrowserFlow *string `json:"browserFlow,omitempty"`

	// Themes is a map of themes to apply to the realm.
	// +nullable
	// +optional
	Themes *RealmThemes `json:"themes,omitempty"`

	// BrowserSecurityHeaders is a map of security headers to apply to HTTP responses from the realm's browser clients.
	// +nullable
	// +optional
	BrowserSecurityHeaders *map[string]string `json:"browserSecurityHeaders,omitempty"`

	// ID is the ID of the realm.
	// +nullable
	// +optional
	ID *string `json:"id,omitempty"`

	// RealmEventConfig is the configuration for events in the realm.
	// +nullable
	// +optional
	RealmEventConfig *RealmEventConfig `json:"realmEventConfig,omitempty"`

	// PasswordPolicies is a list of password policies to apply to the realm.
	// +nullable
	// +optional
	PasswordPolicies []PasswordPolicy `json:"passwordPolicy,omitempty"`

	// DisplayHTMLName name to render in the UI
	// +optional
	DisplayHTMLName string `json:"displayHtmlName,omitempty"`

	// FrontendURL Set the frontend URL for the realm. Use in combination with the default hostname provider to override the base URL for frontend requests for a specific realm.
	// +optional
	FrontendURL string `json:"frontendUrl,omitempty"`

	// TokenSettings is the configuration for tokens in the realm.
	// +nullable
	// +optional
	TokenSettings *common.TokenSettings `json:"tokenSettings,omitempty"`

	// DisplayName is the display name of the realm.
	// +optional
	DisplayName string `json:"displayName,omitempty"`
}

type User struct {
	// Username of keycloak user.
	Username string `json:"username"`

	// RealmRoles is a list of roles attached to keycloak user.
	RealmRoles []string `json:"realmRoles,omitempty"`
}

type PasswordPolicy struct {
	// Type of password policy.
	Type string `json:"type"`
	// Value of password policy.
	Value string `json:"value"`
}

type RealmEventConfig struct {
	// AdminEventsDetailsEnabled indicates whether to enable detailed admin events.
	// +optional
	AdminEventsDetailsEnabled bool `json:"adminEventsDetailsEnabled,omitempty"`

	// AdminEventsEnabled indicates whether to enable admin events.
	// +optional
	AdminEventsEnabled bool `json:"adminEventsEnabled,omitempty"`

	// EnabledEventTypes is a list of event types to enable.
	// +optional
	// +nullable.
	EnabledEventTypes []string `json:"enabledEventTypes,omitempty"`

	// EventsEnabled indicates whether to enable events.
	// +optional
	EventsEnabled bool `json:"eventsEnabled,omitempty"`

	// EventsExpiration is the number of seconds after which events expire.
	// +optional
	EventsExpiration int `json:"eventsExpiration,omitempty"`

	// EventsListeners is a list of event listeners to enable.
	// +optional
	// +nullable.
	EventsListeners []string `json:"eventsListeners,omitempty"`
}

type RealmThemes struct {
	// LoginTheme specifies the login theme to use for the realm.
	// +nullable
	// +optional
	LoginTheme *string `json:"loginTheme"`

	// AccountTheme specifies the account theme to use for the realm.
	// +nullable
	// +optional
	AccountTheme *string `json:"accountTheme"`

	// AdminConsoleTheme specifies the admin console theme to use for the realm.
	// +nullable
	// +optional
	AdminConsoleTheme *string `json:"adminConsoleTheme"`

	// EmailTheme specifies the email theme to use for the realm.
	// +nullable
	// +optional
	EmailTheme *string `json:"emailTheme"`

	// InternationalizationEnabled indicates whether to enable internationalization.
	// +nullable
	// +optional
	InternationalizationEnabled *bool `json:"internationalizationEnabled"`
}

func (in *KeycloakRealm) GetKeycloakRef() common.KeycloakRef {
	return in.Spec.KeycloakRef
}

type SSORealmMapper struct {
	// IdentityProviderMapper specifies the identity provider mapper to use.
	// +optional
	IdentityProviderMapper string `json:"identityProviderMapper,omitempty"`

	// Name specifies the name of the SSO realm mapper.
	// +optional
	Name string `json:"name,omitempty"`

	// Config is a map of configuration options for the SSO realm mapper.
	// +nullable
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

// KeycloakRealmStatus defines the observed state of KeycloakRealm.
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
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Available",type="boolean",JSONPath=".status.available",description="Is the resource available"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconcilation status"

// KeycloakRealm is the Schema for the keycloak realms API.
type KeycloakRealm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmSpec   `json:"spec,omitempty"`
	Status KeycloakRealmStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeycloakRealmList contains a list of KeycloakRealm.
type KeycloakRealmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealm `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealm{}, &KeycloakRealmList{})
}
