package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:deprecatedversion
type KeycloakRealmUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmUserSpec   `json:"spec"`
	Status KeycloakRealmUserStatus `json:"status"`
}

// +kubebuilder:object:root=true
type KeycloakRealmUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmUser `json:"items"`
}

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

func (in *KeycloakRealmUser) GetReconciliationStrategy() string {
	if in.Spec.ReconciliationStrategy == "" {
		return ReconciliationStrategyFull
	}

	return in.Spec.ReconciliationStrategy
}

type KeycloakRealmUserStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`
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

func init() {
	SchemeBuilder.Register(&KeycloakRealmUser{}, &KeycloakRealmUserList{})
}
