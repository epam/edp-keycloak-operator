package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// KeycloakRealmUserSpec defines the desired state of KeycloakRealmUser.
type KeycloakRealmUserSpec struct {
	// Deprecated: use RealmRef instead.
	// Realm is name of KeycloakRealm custom resource.
	// +optional
	Realm string `json:"realm"`

	// RealmRef is reference to Realm custom resource.
	// +optional
	RealmRef common.RealmRef `json:"realmRef"`

	// Username is a username in keycloak.
	Username string `json:"username"`

	// Email is a user email.
	// +optional
	Email string `json:"email,omitempty"`

	// FirstName is a user first name.
	// +optional
	FirstName string `json:"firstName,omitempty"`

	// LastName is a user last name.
	// +optional
	LastName string `json:"lastName,omitempty"`

	// Enabled is a user enabled flag.
	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// EmailVerified is a user email verified flag.
	// +optional
	EmailVerified bool `json:"emailVerified,omitempty"`

	// RequiredUserActions is required action when user log in, example: CONFIGURE_TOTP, UPDATE_PASSWORD, UPDATE_PROFILE, VERIFY_EMAIL.
	// +nullable
	// +optional
	RequiredUserActions []string `json:"requiredUserActions,omitempty"`

	// Roles is a list of roles assigned to user.
	// +nullable
	// +optional
	Roles []string `json:"roles,omitempty"`

	// Groups is a list of groups assigned to user.
	// +nullable
	// +optional
	Groups []string `json:"groups,omitempty"`

	// Attributes is a map of user attributes.
	// +nullable
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`

	// ReconciliationStrategy is a strategy for reconciliation. Possible values: full, create-only.
	// Default value: full. If set to create-only, user will be created only if it does not exist. If user exists, it will not be updated.
	// If set to full, user will be created if it does not exist, or updated if it exists.
	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`

	// Password is a user password. Allows to keep user password within Custom Resource. For security concerns, it is recommended to use PasswordSecret instead.
	// +optional
	Password string `json:"password,omitempty"`

	// KeepResource, when set to false, results in the deletion of the KeycloakRealmUser Custom Resource (CR)
	// from the cluster after the corresponding user is created in Keycloak. The user will continue to exist in Keycloak.
	// When set to true, the CR will not be deleted after processing.
	// +optional
	// +kubebuilder:default=true
	KeepResource bool `json:"keepResource"`

	// PasswordSecret defines Kubernetes secret Name and Key, which holds User secret.
	// +nullable
	// +optional
	PasswordSecret PasswordSecret `json:"passwordSecret,omitempty"`
}

// PasswordSecret defines struct which contains reference to secret name and key.
type PasswordSecret struct {
	// Name is the name of the secret.
	Name string `json:"name"`

	// Key is the key in the secret.
	Key string `json:"key"`
}

// KeycloakRealmUserStatus defines the observed state of KeycloakRealmUser.
type KeycloakRealmUserStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconcilation status"

// KeycloakRealmUser is the Schema for the keycloak user API.
type KeycloakRealmUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmUserSpec   `json:"spec,omitempty"`
	Status KeycloakRealmUserStatus `json:"status,omitempty"`
}

func (in *KeycloakRealmUser) GetReconciliationStrategy() string {
	if in.Spec.ReconciliationStrategy == "" {
		return ReconciliationStrategyFull
	}

	return in.Spec.ReconciliationStrategy
}

func (in *KeycloakRealmUser) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmUser) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakRealmUser) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmUser) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmUser) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

// +kubebuilder:object:root=true

// KeycloakRealmUserList contains a list of KeycloakRealmUser.
type KeycloakRealmUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmUser `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmUser{}, &KeycloakRealmUserList{})
}
