package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// ClusterKeycloakRealmSpec defines the desired state of ClusterKeycloakRealm.
type ClusterKeycloakRealmSpec struct {
	// ClusterKeycloakRef is a name of the ClusterKeycloak instance that owns the realm.
	// +required
	ClusterKeycloakRef string `json:"clusterKeycloakRef"`

	// RealmName specifies the name of the realm.
	RealmName string `json:"realmName"`

	// FrontendURL Set the frontend URL for the realm.
	// Use in combination with the default hostname provider to override the base URL for frontend requests for a specific realm.
	// +optional
	FrontendURL string `json:"frontendUrl,omitempty"`

	// RealmEventConfig is the configuration for events in the realm.
	// +nullable
	// +optional
	RealmEventConfig *RealmEventConfig `json:"realmEventConfig,omitempty"`

	// Themes is a map of themes to apply to the realm.
	// +nullable
	// +optional
	Themes *ClusterRealmThemes `json:"themes,omitempty"`

	// Localization is the configuration for localization in the realm.
	// +nullable
	// +optional
	Localization *RealmLocalization `json:"localization,omitempty"`

	// BrowserSecurityHeaders is a map of security headers to apply to HTTP responses from the realm's browser clients.
	// +nullable
	// +optional
	BrowserSecurityHeaders *map[string]string `json:"browserSecurityHeaders,omitempty"`

	// PasswordPolicies is a list of password policies to apply to the realm.
	// +nullable
	// +optional
	PasswordPolicies []PasswordPolicy `json:"passwordPolicy,omitempty"`

	// TokenSettings is the configuration for tokens in the realm.
	// +nullable
	// +optional
	TokenSettings *common.TokenSettings `json:"tokenSettings,omitempty"`

	// AuthenticationFlow is the configuration for authentication flows in the realm.
	// +nullable
	// +optional
	AuthenticationFlow *AuthenticationFlow `json:"authenticationFlows,omitempty"`
}

type AuthenticationFlow struct {
	// BrowserFlow specifies the authentication flow to use for the realm's browser clients.
	// +optional
	// +kubebuilder:example="browser"
	BrowserFlow string `json:"browserFlow,omitempty"`
}

type ClusterRealmThemes struct {
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
}

type RealmLocalization struct {
	// InternationalizationEnabled indicates whether to enable internationalization.
	// +nullable
	// +optional
	InternationalizationEnabled *bool `json:"internationalizationEnabled"`
}

// ClusterKeycloakRealmStatus defines the observed state of ClusterKeycloakRealm.
type ClusterKeycloakRealmStatus struct {
	// +optional
	Available bool `json:"available,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`

	// +optional
	Value string `json:"value,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Available",type="boolean",JSONPath=".status.available",description="Keycloak realm is available"

// ClusterKeycloakRealm is the Schema for the clusterkeycloakrealms API.
type ClusterKeycloakRealm struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterKeycloakRealmSpec   `json:"spec,omitempty"`
	Status ClusterKeycloakRealmStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterKeycloakRealmList contains a list of ClusterKeycloakRealm.
type ClusterKeycloakRealmList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterKeycloakRealm `json:"items"`
}

func (r *ClusterKeycloakRealm) GetKeycloakRef() common.KeycloakRef {
	return common.KeycloakRef{
		Kind: ClusterKeycloakKind,
		Name: r.Spec.ClusterKeycloakRef,
	}
}

func (in *ClusterKeycloakRealm) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *ClusterKeycloakRealm) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func init() {
	SchemeBuilder.Register(&ClusterKeycloakRealm{}, &ClusterKeycloakRealmList{})
}
