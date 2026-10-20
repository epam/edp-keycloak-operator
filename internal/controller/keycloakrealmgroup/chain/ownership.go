package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
)

// findOwnerCR returns the KeycloakRealmGroup, other than self, whose status.ID equals groupID
// and which targets the same Keycloak realm. Group IDs are unique per Keycloak server only:
// cloned instances reuse UUIDs, so equal realm refs match directly and differing refs are
// compared by resolved (endpoint, effective TLS server name, realm name) from the informer cache.
func findOwnerCR(
	ctx context.Context,
	k8sClient client.Client,
	self *keycloakApi.KeycloakRealmGroup,
	groupID string,
) (*keycloakApi.KeycloakRealmGroup, error) {
	if groupID == "" {
		return nil, nil
	}

	// Resolved once per scan; a resolution error surfaces only when a same-ID
	// candidate actually needs the resolved-target comparison.
	selfTarget, selfResolveErr := helper.ResolveRealmTarget(ctx, k8sClient, self)

	var list keycloakApi.KeycloakRealmGroupList
	if err := k8sClient.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("unable to list KeycloakRealmGroup resources: %w", err)
	}

	for i := range list.Items {
		candidate := &list.Items[i]

		if candidate.Namespace == self.Namespace && candidate.Name == self.Name {
			continue
		}

		if candidate.Status.ID != groupID {
			continue
		}

		if helper.SameRealmRef(self, candidate) {
			return candidate, nil
		}

		candidateTarget, err := helper.ResolveRealmTarget(ctx, k8sClient, candidate)
		if err != nil {
			return nil, fmt.Errorf("unable to resolve realm target for ownership check: %w", err)
		}

		if selfResolveErr != nil {
			return nil, fmt.Errorf("unable to resolve realm target for ownership check: %w", selfResolveErr)
		}

		if selfTarget == candidateTarget {
			return candidate, nil
		}
	}

	return nil, nil
}
