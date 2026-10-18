package secretref

import (
	"slices"

	"github.com/epam/edp-keycloak-operator/pkg/client/keycloakapi"
	"github.com/epam/edp-keycloak-operator/pkg/maputil"
)

// ConfigMatches reports whether the fetched config already holds every comparable key of
// desired. A key is skipped when existing holds the mask sentinel or rawSpec holds a secret
// ref for it; drift inside a skipped key is repaired only by a forced write. A nil rawSpec
// skips sentinel keys only, so a ref-backed key Keycloak returns in clear is compared against
// its resolved value and its drift is repaired.
func ConfigMatches(existing, desired, rawSpec map[string]string) bool {
	comparable := make(map[string]string, len(desired))

	for k, v := range desired {
		if HasSecretRef(rawSpec[k]) || existing[k] == keycloakapi.MaskedSecretValue {
			continue
		}

		comparable[k] = v
	}

	return maputil.ContainsSubset(existing, comparable)
}

// ConfigMatchesMulti is ConfigMatches for multi-valued config. A key is skipped when any raw
// spec value is a secret ref or any fetched value is the mask sentinel.
func ConfigMatchesMulti(existing, desired, rawSpec map[string][]string) bool {
	comparable := make(map[string][]string, len(desired))

	for k, v := range desired {
		if HasAnySecretRef(rawSpec[k]) || slices.Contains(existing[k], keycloakapi.MaskedSecretValue) {
			continue
		}

		comparable[k] = v
	}

	return maputil.ContainsSubsetMulti(existing, comparable)
}
