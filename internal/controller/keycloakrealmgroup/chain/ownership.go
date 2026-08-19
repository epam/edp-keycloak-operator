package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

// findOwnerCR returns the KeycloakRealmGroup that already recorded groupID in its
// status.ID, excluding self, and only if it targets the same realm instance.
// Comparing on the Keycloak group ID (rather than re-deriving parent identity from
// spec) is what lets a single check work uniformly for namespace-local ParentGroup
// refs. The ID is unique per Keycloak server, not cluster-wide: cloned instances
// reuse UUIDs, so ownership is also scoped to the resolved RealmRef.
func findOwnerCR(
	ctx context.Context,
	k8sClient client.Client,
	self *keycloakApi.KeycloakRealmGroup,
	groupID string,
) (*keycloakApi.KeycloakRealmGroup, error) {
	if groupID == "" {
		return nil, nil
	}

	var list keycloakApi.KeycloakRealmGroupList
	if err := k8sClient.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("unable to list KeycloakRealmGroup resources: %w", err)
	}

	for i := range list.Items {
		candidate := &list.Items[i]

		if candidate.Namespace == self.Namespace && candidate.Name == self.Name {
			continue
		}

		if candidate.Status.ID == groupID && sameRealmTarget(self, candidate) {
			return candidate, nil
		}
	}

	return nil, nil
}

// sameRealmTarget reports whether two KeycloakRealmGroup resources target the same
// realm instance. Mirrors sameKeycloakInstance on KeycloakRealm: the namespaced
// KeycloakRealm kind is resolved in the group's own namespace, so identity is
// (namespace, name). ClusterKeycloakRealm is cluster-scoped, so identity is the
// name alone.
func sameRealmTarget(a, b *keycloakApi.KeycloakRealmGroup) bool {
	if realmRefKind(a) != realmRefKind(b) || a.Spec.RealmRef.Name != b.Spec.RealmRef.Name {
		return false
	}

	if realmRefKind(a) == keycloakApi.KeycloakRealmKind {
		return a.Namespace == b.Namespace
	}

	return true
}

func realmRefKind(group *keycloakApi.KeycloakRealmGroup) string {
	if group.Spec.RealmRef.Kind == "" {
		return keycloakApi.KeycloakRealmKind
	}

	return group.Spec.RealmRef.Kind
}
