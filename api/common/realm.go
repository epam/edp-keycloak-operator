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
	RevokeRefreshToken bool `json:"revokeRefreshToken,omitempty"`

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
