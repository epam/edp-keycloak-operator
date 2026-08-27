package chain

import (
	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

// Update endpoints PUT the full representation: a spec edit must force the
// write so keys removed from the spec are dropped in Keycloak.
func specChanged(flow *keycloakApi.KeycloakAuthFlow) bool {
	return flow.Generation != flow.Status.ObservedGeneration
}

// Keycloak may inject default config keys server-side;
// exact comparison would never converge.
func containsConfig(existing, desired map[string]string) bool {
	for k, v := range desired {
		existingValue, ok := existing[k]
		if !ok || existingValue != v {
			return false
		}
	}

	return true
}
