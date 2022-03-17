package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakRealmUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KeycloakRealmUserSpec   `json:"spec"`
	Status            KeycloakRealmUserStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KeycloakRealmUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeycloakRealmUser `json:"items"`
}

type KeycloakRealmUserSpec struct {
	Realm                  string            `json:"realm"`
	Username               string            `json:"username"`
	Email                  string            `json:"email"`
	FirstName              string            `json:"firstName"`
	LastName               string            `json:"lastName"`
	Enabled                bool              `json:"enabled"`
	EmailVerified          bool              `json:"emailVerified"`
	RequiredUserActions    []string          `json:"requiredUserActions"`
	Roles                  []string          `json:"roles"`
	Groups                 []string          `json:"groups"`
	Attributes             map[string]string `json:"attributes"`
	ReconciliationStrategy string            `json:"reconciliationStrategy,omitempty"`
	Password               string            `json:"password"`
	KeepResource           bool              `json:"keepResource"`
}

func (in KeycloakRealmUser) GetReconciliationStrategy() string {
	if in.Spec.ReconciliationStrategy == "" {
		return ReconciliationStrategyFull
	}

	return in.Spec.ReconciliationStrategy
}

type KeycloakRealmUserStatus struct {
	Value        string `json:"value"`
	FailureCount int64  `json:"failureCount"`
}

func (in *KeycloakRealmUser) K8SParentRealmName() (string, error) {
	return in.Spec.Realm, nil
}

func (in KeycloakRealmUser) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmUser) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in KeycloakRealmUser) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmUser) SetStatus(value string) {
	in.Status.Value = value
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmUser{}, &KeycloakRealmUserList{})
}
