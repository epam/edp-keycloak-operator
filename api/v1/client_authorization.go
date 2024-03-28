package v1

const (
	PolicyTypeAggregate = "aggregate"
	PolicyTypeClient    = "client"
	PolicyTypeGroup     = "group"
	PolicyTypeRole      = "role"
	PolicyTypeTime      = "time"
	PolicyTypeUser      = "user"

	PolicyDecisionStrategyUnanimous   = "UNANIMOUS"
	PolicyDecisionStrategyAffirmative = "AFFIRMATIVE"
	PolicyDecisionStrategyConsensus   = "CONSENSUS"

	PolicyLogicPositive = "POSITIVE"
	PolicyLogicNegative = "NEGATIVE"
)

// Policy represents a client authorization policy.
type Policy struct {
	// Type is a policy type.
	// +required
	// +kubebuilder:validation:Enum=aggregate;client;group;role;time;user
	Type string `json:"type"`

	// Name is a policy name.
	// +required
	Name string `json:"name"`

	// Description is a policy description.
	// +optional
	Description string `json:"description,omitempty"`

	// DecisionStrategy is a policy decision strategy.
	// +optional
	// +kubebuilder:validation:Enum=UNANIMOUS;AFFIRMATIVE;CONSENSUS
	// +kubebuilder:default=UNANIMOUS
	DecisionStrategy string `json:"decisionStrategy,omitempty"`

	// Logic is a policy logic.
	// +optional
	// +kubebuilder:validation:Enum=POSITIVE;NEGATIVE
	// +kubebuilder:default=POSITIVE
	Logic string `json:"logic,omitempty"`

	// AggregatedPolicy is an aggregated policy settings.
	AggregatedPolicy *AggregatedPolicyData `json:"aggregatedPolicy,omitempty"`

	// ClientPolicy is a client policy settings.
	ClientPolicy *ClientPolicyData `json:"clientPolicy,omitempty"`

	// GroupPolicy is a group policy settings.
	GroupPolicy *GroupPolicyData `json:"groupPolicy,omitempty"`

	// RolePolicy is a role policy settings.
	RolePolicy *RolePolicyData `json:"rolePolicy,omitempty"`

	// ScopePolicy is a scope policy settings.
	TimePolicy *TimePolicyData `json:"timePolicy,omitempty"`

	// UserPolicy is a user policy settings.
	UserPolicy *UserPolicyData `json:"userPolicy,omitempty"`
}

type ScopePolicyData struct {
	Scopes []string `json:"scopes"`
}

// RolePolicyData represents role based policies.
type RolePolicyData struct {
	// Roles is a list of role.
	// +required
	// +kubebuilder:example={roles:{{name:"role1",required:true},{name:"role2"}}}
	Roles []RoleDefinition `json:"roles"`
}

// RoleDefinition represents a role in a RolePolicyData.
type RoleDefinition struct {
	// Name is a role name.
	// +required
	// +kubebuilder:example="role1"
	Name string `json:"name"`

	// Required is a flag that specifies whether the role is required.
	// +optional
	Required bool `json:"required,omitempty"`
}

// ClientPolicyData represents client based policies.
type ClientPolicyData struct {
	// Clients is a list of client names. Specifies which client(s) are allowed by this policy.
	// +required
	// +kubebuilder:example={clients1,clients2}
	Clients []string `json:"clients"`
}

// TimePolicyData represents time based policies.
type TimePolicyData struct {
	// NotBefore defines the time before which the policy MUST NOT be granted.
	// Only granted if current date/time is after or equal to this value.
	// +required
	// +kubebuilder:example="2024-03-03 00:00:00"
	NotBefore string `json:"notBefore"`

	// NotOnOrAfter defines the time after which the policy MUST NOT be granted.
	// Only granted if current date/time is before or equal to this value.
	// +required
	// +kubebuilder:example="2024-04-04 00:00:00"
	NotOnOrAfter string `json:"notOnOrAfter"`

	// Day defines the month which the policy MUST be granted.
	// You can also provide a range by filling the dayMonthEnd field.
	// In this case, permission is granted only if current month is between or equal to the two values you provided.
	// +optional
	// +kubebuilder:example="1"
	DayMonth string `json:"dayMonth,omitempty"`
	// +optional
	// +kubebuilder:example="2"
	DayMonthEnd string `json:"dayMonthEnd,omitempty"`

	// Month defines the month which the policy MUST be granted.
	// You can also provide a range by filling the monthEnd.
	// In this case, permission is granted only if current month is between or equal to the two values you provided.
	// +optional
	// +kubebuilder:example="1"
	Month string `json:"month,omitempty"`
	// +optional
	// +kubebuilder:example="2"
	MonthEnd string `json:"monthEnd,omitempty"`

	// Hour defines the hour when the policy MUST be granted.
	// You can also provide a range by filling the hourEnd.
	// In this case, permission is granted only if current hour is between or equal to the two values you provided.
	// +optional
	// +kubebuilder:example="1"
	Hour string `json:"hour,omitempty"`
	// +optional
	// +kubebuilder:example="2"
	HourEnd string `json:"hourEnd,omitempty"`

	// Minute defines the minute when the policy MUST be granted.
	// You can also provide a range by filling the minuteEnd field.
	// In this case, permission is granted only if current minute is between or equal to the two values you provided.
	// +optional
	// +kubebuilder:example="1"
	Minute string `json:"minute,omitempty"`
	// +optional
	// +kubebuilder:example="2"
	MinuteEnd string `json:"minuteEnd,omitempty"`
}

// UserPolicyData represents user based policies.
type UserPolicyData struct {
	// Users is a list of usernames. Specifies which user(s) are allowed by this policy.
	// +required
	// +kubebuilder:example={users1,users2}
	Users []string `json:"users"`
}

// AggregatedPolicyData represents aggregated policies.
type AggregatedPolicyData struct {
	// Policies is a list of aggregated policies names.
	// Specifies all the policies that must be applied to the scopes defined by this policy or permission.
	// +required
	// +kubebuilder:example={policies:{policy1,policy2}}
	Policies []string `json:"policies"`
}

// GroupPolicyData represents group based policies.
type GroupPolicyData struct {
	// Groups is a list of group names. Specifies which group(s) are allowed by this policy.
	// +required
	// +kubebuilder:example=`{"groups":[{"name":"group1","extendChildren":true},{"name":"group2"}]}`
	Groups []GroupDefinition `json:"groups,omitempty"`

	// GroupsClaim is a group claim.
	// If defined, the policy will fetch user's groups from the given claim
	// within an access token or ID token representing the identity asking permissions.
	// If not defined, user's groups are obtained from your realm configuration.
	GroupsClaim string `json:"groupsClaim,omitempty"`
}

// GroupDefinition represents a group in a GroupPolicyData.
type GroupDefinition struct {
	// Name is a group name.
	// +required
	// +kubebuilder:example="group1"
	Name string `json:"name"`

	// ExtendChildren is a flag that specifies whether to extend children.
	// +optional
	ExtendChildren bool `json:"extendChildren,omitempty"`
}
