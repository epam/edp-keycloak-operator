package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
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

	// Deprecated: use RealmRef instead.
	// TargetRealm is a realm name where client will be created.
	// It has higher priority than RealmRef for backward compatibility.
	// If both TargetRealm and RealmRef are specified, TargetRealm will be used for client creation.
	// +optional
	TargetRealm string `json:"targetRealm,omitempty"`

	// RealmRef is reference to Realm custom resource.
	// +optional
	RealmRef common.RealmRef `json:"realmRef"`

	// Secret is kubernetes secret name where the client's secret will be stored.
	// Secret should have the following format: $secretName:secretKey.
	// If not specified, a client secret will be generated and stored in a secret with the name keycloak-client-{metadata.name}-secret.
	// If keycloak client is public, secret property will be ignored.
	// +optional
	// +kubebuilder:example="$keycloak-secret:client_secret"
	Secret string `json:"secret,omitempty"`

	// RealmRoles is a list of realm roles assigned to client.
	// +nullable
	// +optional
	RealmRoles *[]RealmRole `json:"realmRoles,omitempty"`

	// Public is a flag to set client as public.
	// +optional
	Public bool `json:"public,omitempty"`

	// WebUrl is a client web url.
	// +optional
	WebUrl string `json:"webUrl,omitempty"`

	// Protocol is a client protocol.
	// +nullable
	// +optional
	Protocol *string `json:"protocol,omitempty"`

	// Attributes is a map of client attributes.
	// +nullable
	// +optional
	// +kubebuilder:default={"post.logout.redirect.uris": "+"}
	Attributes map[string]string `json:"attributes,omitempty"`

	// DirectAccess is a flag to set client as direct access.
	// +optional
	DirectAccess bool `json:"directAccess,omitempty"`

	// AdvancedProtocolMappers is a flag to enable advanced protocol mappers.
	// +optional
	AdvancedProtocolMappers bool `json:"advancedProtocolMappers,omitempty"`

	// ClientRoles is a list of client roles names assigned to client.
	// +nullable
	// +optional
	ClientRoles []string `json:"clientRoles,omitempty"`

	// ProtocolMappers is a list of protocol mappers assigned to client.
	// +nullable
	// +optional
	ProtocolMappers *[]ProtocolMapper `json:"protocolMappers,omitempty"`

	// ServiceAccount is a service account configuration.
	// +nullable
	// +optional
	ServiceAccount *ServiceAccount `json:"serviceAccount,omitempty"`

	// FrontChannelLogout is a flag to enable front channel logout.
	// +optional
	FrontChannelLogout bool `json:"frontChannelLogout,omitempty"`

	// ReconciliationStrategy is a strategy to reconcile client.
	// +kubebuilder:validation:Enum=full;addOnly
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`

	// DefaultClientScopes is a list of default client scopes assigned to client.
	// +nullable
	// +optional
	DefaultClientScopes []string `json:"defaultClientScopes,omitempty"`

	// RedirectUris is a list of valid URI pattern a browser can redirect to after a successful login.
	// Simple wildcards are allowed such as 'https://example.com/*'.
	// Relative path can be specified too, such as /my/relative/path/*. Relative paths are relative to the client root URL.
	// If not specified, spec.webUrl + "/*" will be used.
	// +nullable
	// +optional
	// +kubebuilder:example={"https://example.com/*", "/my/relative/path/*"}
	RedirectUris []string `json:"redirectUris,omitempty"`

	// WebOrigins is a list of allowed CORS origins.
	// To permit all origins of Valid Redirect URIs, add '+'. This does not include the '*' wildcard though.
	// To permit all origins, explicitly add '*'.
	// If not specified, the value from `WebUrl` is used
	// +nullable
	// +optional
	// +kubebuilder:example={"https://example.com/*"}
	WebOrigins []string `json:"webOrigins,omitempty"`

	// ImplicitFlowEnabled is a flag to enable support for OpenID Connect redirect based authentication without authorization code.
	// +optional
	ImplicitFlowEnabled bool `json:"implicitFlowEnabled,omitempty"`

	// ServiceAccountsEnabled enable/disable fine-grained authorization support for a client.
	// +optional
	AuthorizationServicesEnabled bool `json:"authorizationServicesEnabled,omitempty"`

	// BearerOnly is a flag to enable bearer-only.
	// +optional
	BearerOnly bool `json:"bearerOnly,omitempty"`

	// ClientAuthenticatorType is a client authenticator type.
	// +optional
	// +kubebuilder:default="client-secret"
	ClientAuthenticatorType string `json:"clientAuthenticatorType,omitempty"`

	// ConsentRequired is a flag to enable consent.
	// +optional
	ConsentRequired bool `json:"consentRequired,omitempty"`

	// Description is a client description.
	// +optional
	Description string `json:"description,omitempty"`

	// Enabled is a flag to enable client.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// FullScopeAllowed is a flag to enable full scope.
	// +optional
	// +kubebuilder:default=true
	FullScopeAllowed bool `json:"fullScopeAllowed,omitempty"`

	// Name is a client name.
	// +optional
	Name string `json:"name,omitempty"`

	// StandardFlowEnabled is a flag to enable standard flow.
	// +optional
	// +kubebuilder:default=true
	StandardFlowEnabled bool `json:"standardFlowEnabled,omitempty"`

	// SurrogateAuthRequired is a flag to enable surrogate auth.
	SurrogateAuthRequired bool `json:"surrogateAuthRequired,omitempty"`
}

type ServiceAccount struct {
	// Enabled is a flag to enable service account.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// RealmRoles is a list of realm roles assigned to service account.
	// +nullable
	// +optional
	RealmRoles []string `json:"realmRoles"`

	// ClientRoles is a list of client roles assigned to service account.
	// +nullable
	// +optional
	ClientRoles []ClientRole `json:"clientRoles,omitempty"`

	// Attributes is a map of service account attributes.
	// +nullable
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`
}

type ClientRole struct {
	// ClientID is a client ID.
	ClientID string `json:"clientId"`

	// Roles is a list of client roles names assigned to service account.
	// +nullable
	// +optional
	Roles []string `json:"roles,omitempty"`
}

type ProtocolMapper struct {
	// Name is a protocol mapper name.
	// +optional
	Name string `json:"name,omitempty"`

	// Protocol is a protocol name.
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// ProtocolMapper is a protocol mapper name.
	// +optional
	ProtocolMapper string `json:"protocolMapper,omitempty"`

	// Config is a map of protocol mapper configuration.
	// +nullable
	// +optional
	Config map[string]string `json:"config,omitempty"`
}

type RealmRole struct {
	// Name is a realm role name.
	// +optional
	Name string `json:"name,omitempty"`

	// Composite is a realm composite role name.
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
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconcilation status"

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

func (in *KeycloakClient) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
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
