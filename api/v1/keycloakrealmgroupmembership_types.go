package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/epam/edp-keycloak-operator/api/common"
)

// KeycloakRealmGroupMembershipSpec defines the desired state of KeycloakRealmGroupMembership.
type KeycloakRealmGroupMembershipSpec struct {
	// RealmRef is reference to Realm custom resource.
	// +required
	RealmRef common.RealmRef `json:"realmRef"`

	// Group is the Keycloak group whose members are managed.
	// Either a plain group name (e.g. "developers") or a slash-separated path (e.g. "/parent/child").
	// A plain name resolves against top-level groups only. Use a path to reference a nested group.
	// The group must exist; the operator does not create it.
	// +required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:example="/parent/child"
	Group string `json:"group"`

	// Strategy defines how the group members are reconciled. Possible values: addOnly, full.
	// addOnly: the listed members are added to the group; other members are left alone.
	// full: the group contains exactly the listed members; members not listed are removed.
	// +optional
	// +kubebuilder:validation:Enum=addOnly;full
	// +kubebuilder:default=addOnly
	Strategy string `json:"strategy,omitempty"`

	// Members is a list of users that must be members of the group.
	// The operator never creates users: a member that does not exist in Keycloak yet is reported
	// in status.pendingMembers, looked up again every minute and added once it exists, e.g. after the user's first login.
	// +required
	Members []GroupMember `json:"members"`
}

// GroupMember identifies a Keycloak user by exact username or exact email.
// +kubebuilder:validation:XValidation:rule="has(self.username) != has(self.email)",message="exactly one of username or email must be set"
type GroupMember struct {
	// Username is the exact username of the user.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Username string `json:"username,omitempty"`

	// Email is the exact email of the user.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Email string `json:"email,omitempty"`
}

// KeycloakRealmGroupMembershipStatus defines the observed state of KeycloakRealmGroupMembership.
type KeycloakRealmGroupMembershipStatus struct {
	// +optional
	Value string `json:"value,omitempty"`

	// +optional
	FailureCount int64 `json:"failureCount,omitempty"`

	// ResolvedMembers is the list of members found in Keycloak.
	// +optional
	// +nullable
	ResolvedMembers []GroupMember `json:"resolvedMembers,omitempty"`

	// PendingMembers is the list of members not found in Keycloak yet.
	// They are added to the group on a later reconciliation once they exist.
	// +optional
	// +nullable
	PendingMembers []GroupMember `json:"pendingMembers,omitempty"`
}

func (in *KeycloakRealmGroupMembership) GetFailureCount() int64 {
	return in.Status.FailureCount
}

func (in *KeycloakRealmGroupMembership) SetFailureCount(count int64) {
	in.Status.FailureCount = count
}

func (in *KeycloakRealmGroupMembership) GetStatus() string {
	return in.Status.Value
}

func (in *KeycloakRealmGroupMembership) SetStatus(value string) {
	in.Status.Value = value
}

func (in *KeycloakRealmGroupMembership) GetRealmRef() common.RealmRef {
	return in.Spec.RealmRef
}

// IsStrategyFull reports whether members not listed in the spec are removed from the group.
func (in *KeycloakRealmGroupMembership) IsStrategyFull() bool {
	return in.Spec.Strategy == ReconciliationStrategyFull
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Group",type="string",JSONPath=".spec.group",description="Keycloak group"
// +kubebuilder:printcolumn:name="Strategy",type="string",JSONPath=".spec.strategy",description="Reconciliation strategy"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.value",description="Reconciliation status"

// KeycloakRealmGroupMembership is the Schema for the keycloak group membership API.
type KeycloakRealmGroupMembership struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeycloakRealmGroupMembershipSpec   `json:"spec,omitempty"`
	Status KeycloakRealmGroupMembershipStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeycloakRealmGroupMembershipList contains a list of KeycloakRealmGroupMembership.
type KeycloakRealmGroupMembershipList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KeycloakRealmGroupMembership `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeycloakRealmGroupMembership{}, &KeycloakRealmGroupMembershipList{})
}
