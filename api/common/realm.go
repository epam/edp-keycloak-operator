// +kubebuilder:object:generate=true
package common

// TokenSettings is the configuration for tokens in the realm.
// +kubebuilder:object:generate=true
type TokenSettings struct {
	// DefaultSignatureAlgorithm specifies the default algorithm used to sign tokens for the realm
	// +optional
	// +kubebuilder:validation:Enum=ES256;ES384;ES512;EdDSA;HS256;HS384;HS512;PS256;PS384;PS512;RS256;RS384;RS512
	// +kubebuilder:default=RS256
	// +kubebuilder:example=RS256
	DefaultSignatureAlgorithm string `json:"defaultSignatureAlgorithm,omitempty"`

	// RevokeRefreshToken if enabled a refresh token can only be used up to 'refreshTokenMaxReuse' and
	// is revoked when a different token is used.
	// Otherwise, refresh tokens are not revoked when used and can be used multiple times.
	// +optional
	// +kubebuilder:default=false
	RevokeRefreshToken bool `json:"revokeRefreshToken"`

	// RefreshTokenMaxReuse specifies maximum number of times a refresh token can be reused.
	// When a different token is used, revocation is immediate.
	// +optional
	// +kubebuilder:default=0
	RefreshTokenMaxReuse int `json:"refreshTokenMaxReuse,omitempty"`

	// AccessTokenLifespan specifies max time(in seconds) before an access token is expired.
	// This value is recommended to be short relative to the SSO timeout.
	// +optional
	// +kubebuilder:default=300
	AccessTokenLifespan int `json:"accessTokenLifespan,omitempty"`

	// AccessTokenLifespanForImplicitFlow specifies max time(in seconds) before an access token is expired for implicit flow.
	// +optional
	// +kubebuilder:default=900
	AccessTokenLifespanForImplicitFlow int `json:"accessToken,omitempty"`

	// AccessCodeLifespan specifies max time(in seconds)a client has to finish the access token protocol.
	// This should normally be 1 minute.
	// +optional
	// +kubebuilder:default=60
	AccessCodeLifespan int `json:"accessCodeLifespan,omitempty"`

	// AccessCodeLifespanUserAction specifies max time(in seconds) before an action permit sent by a user (such as a forgot password e-mail) is expired.
	// This value is recommended to be short because it's expected that the user would react to self-created action quickly.
	// +optional
	// +kubebuilder:default=300
	ActionTokenGeneratedByUserLifespan int `json:"actionTokenGeneratedByUserLifespan,omitempty"`

	// ActionTokenGeneratedByAdminLifespan specifies max time(in seconds) before an action permit sent to a user by administrator is expired.
	// This value is recommended to be long to allow administrators to send e-mails for users that are currently offline.
	// The default timeout can be overridden immediately before issuing the token.
	// +optional
	// +kubebuilder:default=43200
	ActionTokenGeneratedByAdminLifespan int `json:"actionTokenGeneratedByAdminLifespan,omitempty"`
}

// UserProfileConfig defines the configuration for user profile in the realm.
type UserProfileConfig struct {
	// UnmanagedAttributePolicy are user attributes not explicitly defined in the user profile configuration.
	// Empty value means that unmanaged attributes are disabled.
	// Possible values:
	// ENABLED - unmanaged attributes are allowed.
	// ADMIN_VIEW - unmanaged attributes are read-only and only available through the administration console and API.
	// ADMIN_EDIT - unmanaged attributes can be managed only through the administration console and API.
	// +optional
	UnmanagedAttributePolicy string `json:"unmanagedAttributePolicy,omitempty"`

	// Attributes specifies the list of user profile attributes.
	Attributes []UserProfileAttribute `json:"attributes,omitempty"`

	// Groups specifies the list of user profile groups.
	Groups []UserProfileGroup `json:"groups,omitempty"`
}

type UserProfileAttribute struct {
	// Name of the user attribute, used to uniquely identify an attribute.
	// +required
	Name string `json:"name"`

	// Display name for the attribute.
	DisplayName string `json:"displayName,omitempty"`

	// Group to which the attribute belongs.
	Group string `json:"group,omitempty"`

	// Multivalued specifies if this attribute supports multiple values.
	// This setting is an indicator and does not enable any validation
	Multivalued bool `json:"multivalued,omitempty"`

	// Permissions specifies the permissions for the attribute.
	Permissions *UserProfileAttributePermissions `json:"permissions,omitempty"`

	// Required indicates that the attribute must be set by users and administrators.
	Required *UserProfileAttributeRequired `json:"required,omitempty"`

	// Selector specifies the scopes for which the attribute is available.
	Selector *UserProfileAttributeSelector `json:"selector,omitempty"`

	// Annotations specifies the annotations for the attribute.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Validations specifies the validations for the attribute.
	Validations map[string]map[string]UserProfileAttributeValidation `json:"validations,omitempty"`
}

type UserProfileAttributeValidation struct {
	// +optional
	StringVal string `json:"stringVal,omitempty"`

	// +optional
	// +nullable
	MapVal map[string]string `json:"mapVal,omitempty"`

	// +optional
	IntVal int `json:"intVal,omitempty"`

	// +optional
	// +nullable
	SliceVal []string `json:"sliceVal,omitempty"`
}

type UserProfileAttributePermissions struct {
	// Edit specifies who can edit the attribute.
	Edit []string `json:"edit,omitempty"`

	// View specifies who can view the attribute.
	View []string `json:"view,omitempty"`
}

// UserProfileAttributeRequired defines model for UserProfileAttributeRequired.
type UserProfileAttributeRequired struct {
	// Roles specifies the roles for whom the attribute is required.
	Roles []string `json:"roles,omitempty"`

	// Scopes specifies the scopes when the attribute is required.
	Scopes []string `json:"scopes,omitempty"`
}

// UserProfileAttributeSelector defines model for UserProfileAttributeSelector.
type UserProfileAttributeSelector struct {
	// Scopes specifies the scopes for which the attribute is available.
	Scopes []string `json:"scopes,omitempty"`
}

type UserProfileGroup struct {
	// Name is unique name of the group.
	// +required
	Name string `json:"name"`

	// Annotations specifies the annotations for the group.
	// +optional
	// nullable
	Annotations map[string]string `json:"annotations,omitempty"`

	// DisplayDescription specifies a user-friendly name for the group that should be used when rendering a group of attributes in user-facing forms.
	DisplayDescription string `json:"displayDescription,omitempty"`

	// DisplayHeader specifies a text that should be used as a header when rendering user-facing forms.
	DisplayHeader string `json:"displayHeader,omitempty"`
}
