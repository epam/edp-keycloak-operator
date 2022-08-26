package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// KeycloakRealmUserSpec defines the desired state of KeycloakRealmUser.
type KeycloakRealmUserSpec struct {
	Realm    string `json:"realm"`
	Username string `json:"username"`

	// +optional
	Email string `json:"email,omitempty"`

	// +optional
	FirstName string `json:"firstName,omitempty"`

	// +optional
	LastName string `json:"lastName,omitempty"`

	// +optional
	Enabled bool `json:"enabled,omitempty"`

	// +optional
	EmailVerified bool `json:"emailVerified,omitempty"`

	// RequiredUserActions is required action when user log in, example: CONFIGURE_TOTP, UPDATE_PASSWORD, UPDATE_PROFILE, VERIFY_EMAIL
	// +nullable
	// +optional
	RequiredUserActions []string `json:"requiredUserActions,omitempty"`

	// +nullable
	// +optional
	Roles []string `json:"roles,omitempty"`

	// +nullable
	// +optional
	Groups []string `json:"groups,omitempty"`

	// +nullable
	// +optional
	Attributes map[string]string `json:"attributes,omitempty"`

	// +optional
	ReconciliationStrategy string `json:"reconciliationStrategy,omitempty"`

	// +optional
	Password string `json:"password,omitempty"`

	// +optional
	KeepResource bool `json:"keepResource,omitempty"`
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

// KeycloakRealmUser is the Schema for the keycloak user API.
type KeycloakRealmUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmUserSpec   `json:"spec"`
	Status KeycloakRealmUserStatus `json:"status"`
}

func (in *KeycloakRealmUser) GetReconciliationStrategy() string {
	if in.Spec.ReconciliationStrategy == "" {
		return ReconciliationStrategyFull
	}

	return in.Spec.ReconciliationStrategy
}

func (in *KeycloakRealmUser) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
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
