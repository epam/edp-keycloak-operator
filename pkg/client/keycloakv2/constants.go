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
)
