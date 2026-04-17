package keycloakv2

const (
	// DefaultAdminClientID is the default client ID for admin operations
	DefaultAdminClientID = "admin-cli"

	// MasterRealm is the name of the master realm
	MasterRealm = "master"

	// Default admin credentials for testing
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin"

	// RealmManagementClient is the built-in Keycloak client for the realm.
	// This client manages admin fine-grained permissions for other clients.
	RealmManagementClient = "realm-management"

	// FeatureFlagAdminFineGrainedAuthz is the Keycloak server feature flag for admin fine-grained authorization.
	FeatureFlagAdminFineGrainedAuthz = "ADMIN_FINE_GRAINED_AUTHZ"

	// Protocol mapper protocol types
	ProtocolOpenIDConnect = "openid-connect"
	ProtocolSAML          = "saml"

	// OAuth 2.0 / OIDC grant types
	GrantTypeAuthorizationCode = "authorization_code"
	GrantTypeImplicit          = "implicit"
	GrantTypePassword          = "password"
	GrantTypeClientCredentials = "client_credentials"
	GrantTypeRefreshToken      = "refresh_token"
	GrantTypeDeviceCode        = "urn:ietf:params:oauth:grant-type:device_code"
	GrantTypeTokenExchange     = "urn:ietf:params:oauth:grant-type:token-exchange"

	// Client authentication methods
	ClientAuthClientSecret    = "client-secret"
	ClientAuthClientJWT       = "client-jwt"
	ClientAuthClientSecretJWT = "client-secret-jwt"
	ClientAuthX509Certificate = "client-x509"
)
