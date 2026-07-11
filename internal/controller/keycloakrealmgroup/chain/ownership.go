package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
)

// findOwnerCR returns the KeycloakRealmGroup that already recorded groupID in its
// status.ID, excluding self. Comparing on the actual Keycloak group ID (rather than
// re-deriving realm/parent identity from spec) is what lets a single check work
// uniformly for both RealmRef kinds and for namespace-local ParentGroup refs.
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

		if candidate.Status.ID == groupID {
			return candidate, nil
		}
	}

	return nil, nil
}
