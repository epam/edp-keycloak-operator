package common

// AuthSpec defines the authentication configuration for connecting to Keycloak.
// Exactly one of passwordGrant or clientCredentials must be set.
// +kubebuilder:object:generate=true
// +kubebuilder:validation:XValidation:rule="has(self.passwordGrant) || has(self.clientCredentials)",message="one of passwordGrant or clientCredentials must be set"
// +kubebuilder:validation:XValidation:rule="!(has(self.passwordGrant) && has(self.clientCredentials))",message="passwordGrant and clientCredentials are mutually exclusive"
type AuthSpec struct {
	// PasswordGrant configures resource owner password grant authentication.
	// +optional
	PasswordGrant *PasswordGrantConfig `json:"passwordGrant,omitempty"`

	// ClientCredentials configures OAuth2 client credentials grant authentication.
	// +optional
	ClientCredentials *ClientCredentialsConfig `json:"clientCredentials,omitempty"`
}

// PasswordGrantConfig holds configuration for resource owner password grant.
// +kubebuilder:object:generate=true
type PasswordGrantConfig struct {
	// Username is the admin username for password grant authentication.
	// Can be a direct value or a reference to a key in a Secret or ConfigMap.
	// +required
	Username SourceRefOrVal `json:"username"`

	// PasswordRef is a reference to a secret key containing the password.
	// +required
	PasswordRef SecretKeySelector `json:"passwordRef"`
}

// ClientCredentialsConfig holds configuration for OAuth2 client credentials grant.
// +kubebuilder:object:generate=true
type ClientCredentialsConfig struct {
	// ClientID is the OAuth2 client ID for authentication.
	// Can be a direct value or a reference to a key in a Secret or ConfigMap.
	// +required
	ClientID SourceRefOrVal `json:"clientId"`

	// ClientSecretRef is a reference to a secret key containing the client secret.
	// +required
	ClientSecretRef SecretKeySelector `json:"clientSecretRef"`
}
