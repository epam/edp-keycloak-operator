package chain

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	keycloakApi "github.com/epam/edp-keycloak-operator/api/v1"
	"github.com/epam/edp-keycloak-operator/internal/controller/helper"
)

// findOwnerCR returns the KeycloakRealmComponent, other than self, whose status.ID equals componentID
// and which targets the same Keycloak realm.
// Match rules:
//   - equal realm refs: match;
//   - differing refs: match when the resolved (endpoint, effective TLS server name, realm name) are equal;
//   - the parent is not compared.
func findOwnerCR(
	ctx context.Context,
	k8sClient client.Client,
	self *keycloakApi.KeycloakRealmComponent,
	componentID string,
) (*keycloakApi.KeycloakRealmComponent, error) {
	if componentID == "" {
		return nil, nil
	}

	// Resolved once per scan; a resolution error surfaces only when a same-ID
	// candidate needs the resolved-target comparison.
	selfTarget, selfResolveErr := helper.ResolveRealmTarget(ctx, k8sClient, self)

	var list keycloakApi.KeycloakRealmComponentList
	if err := k8sClient.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("unable to list KeycloakRealmComponent resources: %w", err)
	}

	for i := range list.Items {
		candidate := &list.Items[i]

		if candidate.Namespace == self.Namespace && candidate.Name == self.Name {
			continue
		}

		if candidate.Status.ID != componentID {
			continue
		}

		if helper.SameRealmRef(self, candidate) {
			return candidate, nil
		}

		candidateTarget, err := helper.ResolveRealmTarget(ctx, k8sClient, candidate)
		if err != nil {
			return nil, fmt.Errorf(
				"unable to resolve realm target of KeycloakRealmComponent %s/%s for ownership check: %w",
				candidate.Namespace, candidate.Name, err,
			)
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
