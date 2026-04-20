package keycloakapi

// PolicyBodyBase holds fields common to all policy types sent to Keycloak.
type PolicyBodyBase struct {
	Name             string           `json:"name"`
	Type             string           `json:"type"`
	Description      string           `json:"description,omitempty"`
	DecisionStrategy DecisionStrategy `json:"decisionStrategy,omitempty"`
	Logic            Logic            `json:"logic,omitempty"`
	ID               string           `json:"id,omitempty"`
}

// AggregatePolicyBody is the request body for an aggregate policy.
type AggregatePolicyBody struct {
	PolicyBodyBase
	Policies []string `json:"policies"`
}

// ClientPolicyBody is the request body for a client policy.
type ClientPolicyBody struct {
	PolicyBodyBase
	Clients []string `json:"clients"`
}

// GroupDefinition represents a group reference inside a group policy.
type GroupDefinition struct {
	ID             string `json:"id"`
	ExtendChildren bool   `json:"extendChildren"`
}

// GroupPolicyBody is the request body for a group policy.
type GroupPolicyBody struct {
	PolicyBodyBase
	Groups      []GroupDefinition `json:"groups"`
	GroupsClaim string            `json:"groupsClaim,omitempty"`
}

// RoleDefinition represents a role reference inside a role policy.
type RoleDefinition struct {
	ID       string `json:"id"`
	Required bool   `json:"required"`
}

// RolePolicyBody is the request body for a role policy.
type RolePolicyBody struct {
	PolicyBodyBase
	Roles []RoleDefinition `json:"roles"`
}

// TimePolicyBody is the request body for a time-based policy.
type TimePolicyBody struct {
	PolicyBodyBase
	NotBefore    string `json:"notBefore,omitempty"`
	NotOnOrAfter string `json:"notOnOrAfter,omitempty"`
	DayMonth     string `json:"dayMonth,omitempty"`
	DayMonthEnd  string `json:"dayMonthEnd,omitempty"`
	Month        string `json:"month,omitempty"`
	MonthEnd     string `json:"monthEnd,omitempty"`
	Hour         string `json:"hour,omitempty"`
	HourEnd      string `json:"hourEnd,omitempty"`
	Minute       string `json:"minute,omitempty"`
	MinuteEnd    string `json:"minuteEnd,omitempty"`
}

// UserPolicyBody is the request body for a user policy.
type UserPolicyBody struct {
	PolicyBodyBase
	Users []string `json:"users"`
}
